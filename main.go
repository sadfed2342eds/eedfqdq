// Fast multi-threaded BIP39 -> BIP32 derivator for path m/44'/6060'/0'/0/%d
// (GoChain, SLIP-44 = 6060, EVM-compatible addresses).
//
// Input:  seed.txt   - one BIP39 mnemonic per line.
// Output: result.txt - <address>|<privkey_hex>|<mnemonic>
//         adress.txt - <address>
package main

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"hash"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	secp256k1 "github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/sha3"
)

const hardened uint32 = 0x80000000

// ------------------------------------------------------------------
// Minimal, fast BIP32 (private keys only).
// ------------------------------------------------------------------

type extKey struct {
	key [32]byte
	cc  [32]byte
}

func newMasterKey(seed []byte) *extKey {
	mac := hmac.New(sha512.New, []byte("Bitcoin seed"))
	mac.Write(seed)
	I := mac.Sum(nil)
	var k extKey
	copy(k.key[:], I[:32])
	copy(k.cc[:], I[32:])
	return &k
}

// deriveChild: general-purpose, handles hardened + non-hardened.
// Used only for the shallow path (5 calls per mnemonic at most).
func (p *extKey) deriveChild(index uint32) (*extKey, error) {
	mac := hmac.New(sha512.New, p.cc[:])
	var data [37]byte
	if index >= hardened {
		// 0x00 || parent_priv || index
		data[0] = 0
		copy(data[1:33], p.key[:])
	} else {
		// compressed parent pubkey || index
		priv := secp256k1.PrivKeyFromBytes(p.key[:])
		pub := priv.PubKey().SerializeCompressed()
		copy(data[0:33], pub)
	}
	binary.BigEndian.PutUint32(data[33:37], index)
	mac.Write(data[:])
	I := mac.Sum(nil)
	return combine(p.key, I)
}

// combine: child_priv = (IL + parent_priv) mod n ; child_cc = IR.
func combine(parentKey [32]byte, I []byte) (*extKey, error) {
	var ilBytes, pBytes [32]byte
	copy(ilBytes[:], I[:32])
	copy(pBytes[:], parentKey[:])
	var il, pb secp256k1.ModNScalar
	if overflow := il.SetBytes(&ilBytes); overflow != 0 {
		return nil, fmt.Errorf("IL overflow")
	}
	pb.SetBytes(&pBytes)
	il.Add(&pb)
	if il.IsZero() {
		return nil, fmt.Errorf("zero key")
	}
	child := &extKey{}
	b := il.Bytes()
	copy(child.key[:], b[:])
	copy(child.cc[:], I[32:])
	return child, nil
}

// ------------------------------------------------------------------
// Hot loop: non-hardened derivation reusing HMAC + parent pubkey.
// One parent -> many children (this is where 99% of time is spent
// when -n is large).
// ------------------------------------------------------------------

type nhCtx struct {
	mac       hash.Hash // hmac-sha512 keyed with parent chaincode (Reset() is cheap)
	parentKey [32]byte
	data      [37]byte // first 33 bytes = compressed parent pubkey, last 4 bytes = index
}

func newNHCtx(p *extKey) *nhCtx {
	c := &nhCtx{}
	c.mac = hmac.New(sha512.New, p.cc[:])
	priv := secp256k1.PrivKeyFromBytes(p.key[:])
	pub := priv.PubKey().SerializeCompressed()
	copy(c.data[0:33], pub)
	copy(c.parentKey[:], p.key[:])
	return c
}

func (c *nhCtx) derive(index uint32) ([32]byte, error) {
	var out [32]byte
	c.mac.Reset()
	binary.BigEndian.PutUint32(c.data[33:37], index)
	c.mac.Write(c.data[:])
	I := c.mac.Sum(nil)

	var ilBytes, pBytes [32]byte
	copy(ilBytes[:], I[:32])
	copy(pBytes[:], c.parentKey[:])
	var il, pb secp256k1.ModNScalar
	if overflow := il.SetBytes(&ilBytes); overflow != 0 {
		return out, fmt.Errorf("IL overflow")
	}
	pb.SetBytes(&pBytes)
	il.Add(&pb)
	if il.IsZero() {
		return out, fmt.Errorf("zero key")
	}
	b := il.Bytes()
	copy(out[:], b[:])
	return out, nil
}

// priv -> EVM-style address (last 20 bytes of keccak256(uncompressed_pub[1:])).
func privToAddress(privBytes [32]byte, hasher hash.Hash) string {
	priv := secp256k1.PrivKeyFromBytes(privBytes[:])
	pub := priv.PubKey().SerializeUncompressed() // 65 bytes, 0x04 || X || Y
	hasher.Reset()
	hasher.Write(pub[1:])
	sum := hasher.Sum(nil)
	return "0x" + hex.EncodeToString(sum[12:])
}

// ------------------------------------------------------------------
// Worker pool
// ------------------------------------------------------------------

type outRec struct {
	addr     string
	privHex  string
	mnemonic string
}

func worker(jobs <-chan string, out chan<- outRec, count int,
	processed, failed *uint64) {

	hasher := sha3.NewLegacyKeccak256()

	for mnemonic := range jobs {
		mnemonic = strings.TrimSpace(mnemonic)
		if mnemonic == "" {
			continue
		}
		if !bip39.IsMnemonicValid(mnemonic) {
			atomic.AddUint64(failed, 1)
			continue
		}

		seed := bip39.NewSeed(mnemonic, "")
		master := newMasterKey(seed)

		// m/44' -> 6060' -> 0' -> 0
		c1, err := master.deriveChild(hardened + 44)
		if err != nil {
			atomic.AddUint64(failed, 1)
			continue
		}
		c2, err := c1.deriveChild(hardened + 6060)
		if err != nil {
			atomic.AddUint64(failed, 1)
			continue
		}
		c3, err := c2.deriveChild(hardened + 0)
		if err != nil {
			atomic.AddUint64(failed, 1)
			continue
		}
		c4, err := c3.deriveChild(0) // non-hardened
		if err != nil {
			atomic.AddUint64(failed, 1)
			continue
		}

		ctx := newNHCtx(c4)
		for i := 0; i < count; i++ {
			privBytes, err := ctx.derive(uint32(i))
			if err != nil {
				continue
			}
			addr := privToAddress(privBytes, hasher)
			out <- outRec{
				addr:     addr,
				privHex:  hex.EncodeToString(privBytes[:]),
				mnemonic: mnemonic,
			}
		}
		atomic.AddUint64(processed, 1)
	}
}

// ------------------------------------------------------------------
// main
// ------------------------------------------------------------------

func main() {
	inputFile := flag.String("in", "seed.txt", "input file with mnemonics (one per line)")
	resultFile := flag.String("out", "result.txt", "output file: addr|priv|mnemonic")
	addrFile := flag.String("addr", "adress.txt", "output file: addresses only")
	count := flag.Int("n", 1, "addresses per mnemonic (indexes 0..n-1)")
	workers := flag.Int("w", runtime.NumCPU(), "number of worker goroutines")
	flag.Parse()

	runtime.GOMAXPROCS(*workers)

	fIn, err := os.Open(*inputFile)
	if err != nil {
		log.Fatalf("open %s: %v", *inputFile, err)
	}
	defer fIn.Close()

	fRes, err := os.Create(*resultFile)
	if err != nil {
		log.Fatalf("create %s: %v", *resultFile, err)
	}
	defer fRes.Close()
	fAddr, err := os.Create(*addrFile)
	if err != nil {
		log.Fatalf("create %s: %v", *addrFile, err)
	}
	defer fAddr.Close()

	bwRes := bufio.NewWriterSize(fRes, 1<<20)
	bwAddr := bufio.NewWriterSize(fAddr, 1<<20)

	jobs := make(chan string, *workers*8)
	results := make(chan outRec, *workers*16)

	var processed, failed, written uint64

	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(jobs, results, *count, &processed, &failed)
		}()
	}

	// writer goroutine
	var wgWriter sync.WaitGroup
	wgWriter.Add(1)
	go func() {
		defer wgWriter.Done()
		for r := range results {
			bwRes.WriteString(r.addr)
			bwRes.WriteByte('|')
			bwRes.WriteString(r.privHex)
			bwRes.WriteByte('|')
			bwRes.WriteString(r.mnemonic)
			bwRes.WriteByte('\n')

			bwAddr.WriteString(r.addr)
			bwAddr.WriteByte('\n')

			atomic.AddUint64(&written, 1)
		}
	}()

	// progress reporter
	stopProgress := make(chan struct{})
	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		start := time.Now()
		for {
			select {
			case <-t.C:
				p := atomic.LoadUint64(&processed)
				fl := atomic.LoadUint64(&failed)
				w := atomic.LoadUint64(&written)
				el := time.Since(start).Seconds()
				rate := float64(p) / el
				fmt.Fprintf(os.Stderr,
					"[+%6.0fs] mnemonics=%d failed=%d written=%d  rate=%.0f mnem/s\n",
					el, p, fl, w, rate)
			case <-stopProgress:
				return
			}
		}
	}()

	// feed jobs
	start := time.Now()
	sc := bufio.NewScanner(fIn)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	var fed uint64
	for sc.Scan() {
		jobs <- sc.Text()
		fed++
	}
	if err := sc.Err(); err != nil {
		log.Fatalf("scan: %v", err)
	}
	close(jobs)

	wg.Wait()
	close(results)
	wgWriter.Wait()
	close(stopProgress)

	if err := bwRes.Flush(); err != nil {
		log.Fatalf("flush result: %v", err)
	}
	if err := bwAddr.Flush(); err != nil {
		log.Fatalf("flush addr: %v", err)
	}

	fmt.Fprintf(os.Stderr,
		"Done in %.2fs: fed=%d processed=%d failed=%d written=%d\n",
		time.Since(start).Seconds(), fed,
		atomic.LoadUint64(&processed),
		atomic.LoadUint64(&failed),
		atomic.LoadUint64(&written))
}
