// Native fast GoChain activity+balance checker (pipelined, batched).
//
// Pipeline:
//   [input] -> activity-workers (phase 1: getTransactionCount+getCode batch)
//           -> aggregator (re-batches active addresses into full batches)
//           -> balance-workers (phase 2: getBalance batch)
//           -> writer
//
// Input:  adress.txt - one address per line (0x-prefixed, 20 bytes hex).
// Output: result.txt - "<address> <wei> <GO> nonce=<n> contract=<bool>"
//                      (active addresses only, including zero balance).
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
		MaxIdleConns:        1024,
		MaxIdleConnsPerHost: 512,
		MaxConnsPerHost:     512,
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
// Records flowing between stages
// ------------------------------------------------------------------

type activeAddr struct {
	address  string
	nonce    uint64
	contract bool
}

type balanced struct {
	activeAddr
	wei *big.Int
}

// ------------------------------------------------------------------
// Phase 1: activity check (batched)
// ------------------------------------------------------------------

func activityBatch(pool *rpcPool, batch []string, retries int) ([]activeAddr, error) {
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
	res, err := pool.doBatch(reqs, retries)
	if err != nil {
		return nil, err
	}
	actives := make([]activeAddr, 0, N/8)
	for i, a := range batch {
		nonceHex, okN := res[i]
		codeHex, okC := res[N+i]
		if !okN && !okC {
			continue
		}
		var nonce uint64
		if okN {
			if n, ok := parseHexBig(nonceHex); ok {
				nonce = n.Uint64()
			}
		}
		contract := okC && isNonEmptyCode(codeHex)
		if nonce > 0 || contract {
			actives = append(actives, activeAddr{address: a, nonce: nonce, contract: contract})
		}
	}
	return actives, nil
}

// ------------------------------------------------------------------
// Phase 2: balance check (batched)
// ------------------------------------------------------------------

func balanceBatch(pool *rpcPool, batch []activeAddr, retries int) ([]balanced, error) {
	reqs := make([]rpcReq, len(batch))
	for i, p := range batch {
		reqs[i] = rpcReq{
			JSONRPC: "2.0", ID: i,
			Method: "eth_getBalance",
			Params: []interface{}{p.address, "latest"},
		}
	}
	res, err := pool.doBatch(reqs, retries)
	if err != nil {
		return nil, err
	}
	out := make([]balanced, len(batch))
	for i, p := range batch {
		wei := new(big.Int)
		if bh, ok := res[i]; ok {
			if v, ok2 := parseHexBig(bh); ok2 {
				wei = v
			}
		}
		out[i] = balanced{activeAddr: p, wei: wei}
	}
	return out, nil
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
	actWorkers := flag.Int("wa", runtime.NumCPU()*4, "activity-phase workers")
	balWorkers := flag.Int("wb", runtime.NumCPU()*2, "balance-phase workers")
	actBatch := flag.Int("ab", 200, "addresses per activity batch")
	balBatch := flag.Int("bb", 500, "addresses per balance batch")
	flushMs := flag.Int("flush", 500, "max ms to wait before flushing a partial balance batch")
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

	// ---- load addresses ----
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
	log.Printf("loaded %d addresses; activity: w=%d batch=%d | balance: w=%d batch=%d | rpc=%v",
		len(addresses), *actWorkers, *actBatch, *balWorkers, *balBatch, endpoints)

	// ---- output ----
	fOut, err := os.Create(*outFile)
	if err != nil {
		log.Fatalf("create %s: %v", *outFile, err)
	}
	defer fOut.Close()
	bw := bufio.NewWriterSize(fOut, 1<<20)
	var writeMu sync.Mutex

	// ---- pipeline channels ----
	actJobs := make(chan []string, *actWorkers*2)        // raw address batches
	activeStream := make(chan activeAddr, *actWorkers*256) // individual active addresses
	balJobs := make(chan []activeAddr, *balWorkers*2)    // aggregated active batches
	results := make(chan []balanced, *balWorkers*2)

	var checked, failActivity, failBalance, activeCnt, nonZero uint64

	// ---- phase 1 workers ----
	var wg1 sync.WaitGroup
	for i := 0; i < *actWorkers; i++ {
		wg1.Add(1)
		go func() {
			defer wg1.Done()
			for b := range actJobs {
				actives, err := activityBatch(pool, b, *retries)
				if err != nil {
					atomic.AddUint64(&failActivity, uint64(len(b)))
					continue
				}
				atomic.AddUint64(&checked, uint64(len(b)))
				for _, a := range actives {
					activeStream <- a
				}
			}
		}()
	}

	// ---- aggregator: packs activeStream into balJobs of size balBatch ----
	var wgAgg sync.WaitGroup
	wgAgg.Add(1)
	go func() {
		defer wgAgg.Done()
		buf := make([]activeAddr, 0, *balBatch)
		timer := time.NewTimer(time.Duration(*flushMs) * time.Millisecond)
		defer timer.Stop()
		flush := func() {
			if len(buf) == 0 {
				return
			}
			out := make([]activeAddr, len(buf))
			copy(out, buf)
			balJobs <- out
			buf = buf[:0]
		}
		for {
			select {
			case a, ok := <-activeStream:
				if !ok {
					flush()
					return
				}
				buf = append(buf, a)
				if len(buf) >= *balBatch {
					flush()
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					timer.Reset(time.Duration(*flushMs) * time.Millisecond)
				}
			case <-timer.C:
				flush()
				timer.Reset(time.Duration(*flushMs) * time.Millisecond)
			}
		}
	}()

	// ---- phase 2 workers ----
	var wg2 sync.WaitGroup
	for i := 0; i < *balWorkers; i++ {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			for b := range balJobs {
				recs, err := balanceBatch(pool, b, *retries)
				if err != nil {
					atomic.AddUint64(&failBalance, uint64(len(b)))
					continue
				}
				results <- recs
			}
		}()
	}

	// ---- writer ----
	var wgW sync.WaitGroup
	wgW.Add(1)
	go func() {
		defer wgW.Done()
		for recs := range results {
			writeMu.Lock()
			for _, r := range recs {
				fmt.Fprintf(bw, "%s %s %s nonce=%d contract=%t\n",
					r.address, r.wei.String(), weiToGO(r.wei),
					r.nonce, r.contract)
				atomic.AddUint64(&activeCnt, 1)
				if r.wei.Sign() > 0 {
					atomic.AddUint64(&nonZero, 1)
				}
			}
			writeMu.Unlock()
		}
	}()

	// ---- progress ----
	stop := make(chan struct{})
	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		start := time.Now()
		for {
			select {
			case <-t.C:
				c := atomic.LoadUint64(&checked)
				f1 := atomic.LoadUint64(&failActivity)
				f2 := atomic.LoadUint64(&failBalance)
				ac := atomic.LoadUint64(&activeCnt)
				nz := atomic.LoadUint64(&nonZero)
				el := time.Since(start).Seconds()
				fmt.Fprintf(os.Stderr,
					"[+%6.0fs] checked=%d failA=%d failB=%d active=%d non-zero=%d rate=%.0f addr/s\n",
					el, c, f1, f2, ac, nz, float64(c)/el)
			case <-stop:
				return
			}
		}
	}()

	// ---- feed ----
	start := time.Now()
	for i := 0; i < len(addresses); i += *actBatch {
		end := i + *actBatch
		if end > len(addresses) {
			end = len(addresses)
		}
		actJobs <- addresses[i:end]
	}
	close(actJobs)

	// order: phase1 -> aggregator -> phase2 -> writer
	wg1.Wait()
	close(activeStream)
	wgAgg.Wait()
	close(balJobs)
	wg2.Wait()
	close(results)
	wgW.Wait()
	close(stop)

	if err := bw.Flush(); err != nil {
		log.Fatalf("flush: %v", err)
	}

	fmt.Fprintf(os.Stderr,
		"Done in %.2fs: total=%d checked=%d failA=%d failB=%d active=%d non-zero=%d\n",
		time.Since(start).Seconds(), len(addresses),
		atomic.LoadUint64(&checked),
		atomic.LoadUint64(&failActivity),
		atomic.LoadUint64(&failBalance),
		atomic.LoadUint64(&activeCnt),
		atomic.LoadUint64(&nonZero))
}
