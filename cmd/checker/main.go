// Native fast GoChain activity+balance checker.
//
// Pipeline (two-phase, per batch):
//   1. eth_getTransactionCount(addr, "latest")   -> activity (nonce > 0)
//      eth_getCode(addr, "latest")               -> activity (contract)
//      Both are packed into ONE JSON-RPC batch request (2*N items).
//   2. For addresses marked active in phase 1:
//      eth_getBalance(addr, "latest")            -> native balance
//
// Input:  adress.txt  - one address per line (0x-prefixed, 20 bytes hex).
// Output: result.txt  - "<address> <wei> <GO> nonce=<n> contract=<bool>"
//                       (active addresses only, including zero balance).
//
// No external deps (stdlib only).
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ------------------------------------------------------------------
// JSON-RPC
// ------------------------------------------------------------------

type rpcReq struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type rpcResp struct {
	ID    int    `json:"id"`
	Result string `json:"result"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type rpcPool struct {
	endpoints []string
	idx       uint64
	client    *http.Client
}

func newPool(urls []string, timeout time.Duration) *rpcPool {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        512,
		MaxIdleConnsPerHost: 256,
		MaxConnsPerHost:     256,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		ForceAttemptHTTP2:   true,
	}
	return &rpcPool{
		endpoints: urls,
		client:    &http.Client{Transport: tr, Timeout: timeout},
	}
}

func (p *rpcPool) next() string {
	i := atomic.AddUint64(&p.idx, 1)
	return p.endpoints[int(i)%len(p.endpoints)]
}

// doBatch sends a pre-built batch and returns results by id -> hex string.
func (p *rpcPool) doBatch(reqs []rpcReq, retries int) (map[int]string, error) {
	body, err := json.Marshal(reqs)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		endpoint := p.next()
		req, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := p.client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(backoff(attempt))
			continue
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			time.Sleep(backoff(attempt))
			continue
		}
		if resp.StatusCode/100 != 2 {
			lastErr = fmt.Errorf("rpc %s: status %d: %s",
				endpoint, resp.StatusCode, truncate(string(data), 200))
			time.Sleep(backoff(attempt))
			continue
		}
		var responses []rpcResp
		if err := json.Unmarshal(data, &responses); err != nil {
			var single rpcResp
			if err2 := json.Unmarshal(data, &single); err2 == nil {
				responses = []rpcResp{single}
			} else {
				lastErr = fmt.Errorf("rpc decode: %w (body=%s)",
					err, truncate(string(data), 200))
				time.Sleep(backoff(attempt))
				continue
			}
		}
		out := make(map[int]string, len(responses))
		for _, r := range responses {
			if r.Error != nil {
				continue
			}
			out[r.ID] = r.Result
		}
		return out, nil
	}
	return nil, lastErr
}

func backoff(attempt int) time.Duration {
	d := time.Duration(200<<attempt) * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// ------------------------------------------------------------------
// Hex helpers
// ------------------------------------------------------------------

var weiPerGo = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

func weiToGO(wei *big.Int) string {
	if wei.Sign() == 0 {
		return "0"
	}
	quo, rem := new(big.Int).QuoRem(wei, weiPerGo, new(big.Int))
	remStr := fmt.Sprintf("%018s", rem.String())
	remStr = strings.TrimRight(remStr, "0")
	if remStr == "" {
		return quo.String()
	}
	return quo.String() + "." + remStr
}

func parseHexBig(hexStr string) (*big.Int, bool) {
	s := strings.TrimPrefix(strings.TrimSpace(hexStr), "0x")
	if s == "" {
		return new(big.Int), true
	}
	n, ok := new(big.Int).SetString(s, 16)
	return n, ok
}

func isNonEmptyCode(hexStr string) bool {
	s := strings.TrimPrefix(strings.TrimSpace(hexStr), "0x")
	return s != "" && strings.Trim(s, "0") != ""
}

// ------------------------------------------------------------------
// Worker: two-phase check on a batch of addresses.
// ------------------------------------------------------------------

type activeRec struct {
	address  string
	nonce    uint64
	contract bool
	wei      *big.Int
}

func processBatch(pool *rpcPool, batch []string, retries int) ([]activeRec, int, error) {
	// Phase 1: getTransactionCount + getCode, packed into one batch.
	// IDs: 0..N-1 -> txCount for addresses[i]
	//      N..2N-1 -> getCode for addresses[i-N]
	N := len(batch)
	reqs := make([]rpcReq, 0, 2*N)
	for i, a := range batch {
		reqs = append(reqs, rpcReq{
			JSONRPC: "2.0", ID: i,
			Method: "eth_getTransactionCount",
			Params: []interface{}{a, "latest"},
		})
	}
	for i, a := range batch {
		reqs = append(reqs, rpcReq{
			JSONRPC: "2.0", ID: N + i,
			Method: "eth_getCode",
			Params: []interface{}{a, "latest"},
		})
	}
	phase1, err := pool.doBatch(reqs, retries)
	if err != nil {
		return nil, 0, err
	}

	// Determine active addresses.
	type prelim struct {
		addr     string
		nonce    uint64
		contract bool
	}
	actives := make([]prelim, 0, N)
	for i, a := range batch {
		nonceHex, okN := phase1[i]
		codeHex, okC := phase1[N+i]
		if !okN && !okC {
			continue // RPC failure on both -> skip
		}
		var nonce uint64
		if okN {
			if n, ok := parseHexBig(nonceHex); ok {
				nonce = n.Uint64()
			}
		}
		contract := okC && isNonEmptyCode(codeHex)
		if nonce > 0 || contract {
			actives = append(actives, prelim{addr: a, nonce: nonce, contract: contract})
		}
	}
	if len(actives) == 0 {
		return nil, N, nil
	}

	// Phase 2: getBalance only for active addresses.
	balReqs := make([]rpcReq, len(actives))
	for i, p := range actives {
		balReqs[i] = rpcReq{
			JSONRPC: "2.0", ID: i,
			Method: "eth_getBalance",
			Params: []interface{}{p.addr, "latest"},
		}
	}
	phase2, err := pool.doBatch(balReqs, retries)
	if err != nil {
		return nil, 0, err
	}

	out := make([]activeRec, 0, len(actives))
	for i, p := range actives {
		wei := new(big.Int)
		if bh, ok := phase2[i]; ok {
			if v, ok2 := parseHexBig(bh); ok2 {
				wei = v
			}
		}
		out = append(out, activeRec{
			address:  p.addr,
			nonce:    p.nonce,
			contract: p.contract,
			wei:      wei,
		})
	}
	return out, N, nil
}

// ------------------------------------------------------------------
// main
// ------------------------------------------------------------------

func main() {
	inFile := flag.String("in", "adress.txt", "input file with addresses")
	outFile := flag.String("out", "result.txt", "output file for active addresses")
	rpcList := flag.String("rpc",
		"https://rpc.gochain.io,https://rpc.gochain.org",
		"comma-separated RPC endpoints")
	workers := flag.Int("w", runtime.NumCPU()*4, "number of HTTP worker goroutines")
	batch := flag.Int("batch", 100, "addresses per JSON-RPC batch")
	timeout := flag.Duration("timeout", 30*time.Second, "HTTP timeout per request")
	retries := flag.Int("retries", 3, "retries per batch on network/HTTP error")
	flag.Parse()

	runtime.GOMAXPROCS(runtime.NumCPU())

	endpoints := strings.Split(*rpcList, ",")
	for i := range endpoints {
		endpoints[i] = strings.TrimSpace(endpoints[i])
	}
	if len(endpoints) == 0 || endpoints[0] == "" {
		log.Fatal("no RPC endpoints provided")
	}
	pool := newPool(endpoints, *timeout)

	// load addresses
	fIn, err := os.Open(*inFile)
	if err != nil {
		log.Fatalf("open %s: %v", *inFile, err)
	}
	defer fIn.Close()

	var addresses []string
	sc := bufio.NewScanner(fIn)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		a := strings.TrimSpace(sc.Text())
		if a == "" {
			continue
		}
		if !strings.HasPrefix(a, "0x") && !strings.HasPrefix(a, "0X") {
			a = "0x" + a
		}
		if len(a) != 42 {
			continue
		}
		addresses = append(addresses, a)
	}
	if err := sc.Err(); err != nil {
		log.Fatalf("scan: %v", err)
	}
	if len(addresses) == 0 {
		log.Fatal("no addresses loaded")
	}
	log.Printf("loaded %d addresses, %d workers, batch=%d, endpoints=%v",
		len(addresses), *workers, *batch, endpoints)

	// output
	fOut, err := os.Create(*outFile)
	if err != nil {
		log.Fatalf("create %s: %v", *outFile, err)
	}
	defer fOut.Close()
	bw := bufio.NewWriterSize(fOut, 1<<20)
	var writeMu sync.Mutex

	jobs := make(chan []string, *workers*2)
	results := make(chan []activeRec, *workers*2)

	var checked, failed, active, nonZero uint64

	var wgW sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wgW.Add(1)
		go func() {
			defer wgW.Done()
			for b := range jobs {
				recs, done, err := processBatch(pool, b, *retries)
				if err != nil {
					atomic.AddUint64(&failed, uint64(len(b)))
					continue
				}
				atomic.AddUint64(&checked, uint64(done))
				if len(recs) > 0 {
					results <- recs
				}
			}
		}()
	}

	var wgR sync.WaitGroup
	wgR.Add(1)
	go func() {
		defer wgR.Done()
		for recs := range results {
			writeMu.Lock()
			for _, r := range recs {
				fmt.Fprintf(bw, "%s %s %s nonce=%d contract=%t\n",
					r.address, r.wei.String(), weiToGO(r.wei),
					r.nonce, r.contract)
				atomic.AddUint64(&active, 1)
				if r.wei.Sign() > 0 {
					atomic.AddUint64(&nonZero, 1)
				}
			}
			writeMu.Unlock()
		}
	}()

	stop := make(chan struct{})
	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		start := time.Now()
		for {
			select {
			case <-t.C:
				c := atomic.LoadUint64(&checked)
				fl := atomic.LoadUint64(&failed)
				ac := atomic.LoadUint64(&active)
				nz := atomic.LoadUint64(&nonZero)
				el := time.Since(start).Seconds()
				fmt.Fprintf(os.Stderr,
					"[+%6.0fs] checked=%d failed=%d active=%d non-zero=%d rate=%.0f addr/s\n",
					el, c, fl, ac, nz, float64(c)/el)
			case <-stop:
				return
			}
		}
	}()

	start := time.Now()
	for i := 0; i < len(addresses); i += *batch {
		end := i + *batch
		if end > len(addresses) {
			end = len(addresses)
		}
		jobs <- addresses[i:end]
	}
	close(jobs)

	wgW.Wait()
	close(results)
	wgR.Wait()
	close(stop)

	if err := bw.Flush(); err != nil {
		log.Fatalf("flush: %v", err)
	}

	fmt.Fprintf(os.Stderr,
		"Done in %.2fs: total=%d checked=%d failed=%d active=%d non-zero=%d\n",
		time.Since(start).Seconds(), len(addresses),
		atomic.LoadUint64(&checked),
		atomic.LoadUint64(&failed),
		atomic.LoadUint64(&active),
		atomic.LoadUint64(&nonZero))
}
