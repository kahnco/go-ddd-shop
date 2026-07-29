// promotion-loadtest — 응모 엔드포인트에 부하를 주고 지연 분포·처리량·상태코드를 측정한다.
// 동기(즉시 순번) vs 큐 완충(202 접수)의 접수 지연 차이를 재는 데 쓴다.
//
// 사용: go run ./cmd/promotion-loadtest -url http://localhost:8091 -event ev -c 50 -n 5000
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	url := flag.String("url", "http://localhost:8080", "대상 서비스 base URL")
	event := flag.String("event", "launch-2026", "이벤트 ID")
	label := flag.String("label", "", "결과 라벨(예: sync/queue)")
	concurrency := flag.Int("c", 50, "동시 워커 수")
	total := flag.Int("n", 5000, "총 요청 수")
	maxP99 := flag.Float64("max-p99-ms", 0, "p99 상한(ms). 초과하면 실패 종료(0=검사 안 함)")
	maxErr := flag.Int("max-err", 0, "허용 오류/5xx 수. 초과하면 실패 종료")
	flag.Parse()

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        *concurrency * 2,
			MaxIdleConnsPerHost: *concurrency * 2,
		},
	}
	endpoint := fmt.Sprintf("%s/events/%s/entries", *url, *event)

	var next int64 = -1
	latencies := make([]time.Duration, *total)
	codes := make([]int, *total)

	start := time.Now()
	var wg sync.WaitGroup
	for w := 0; w < *concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := atomic.AddInt64(&next, 1)
				if int(i) >= *total {
					return
				}
				user := "lt-" + strconv.FormatInt(i, 10)
				t0 := time.Now()
				code := doPost(client, endpoint, user)
				latencies[i] = time.Since(t0)
				codes[i] = code
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	p99, bad := report(*label, *total, elapsed, latencies, codes)

	// CI 게이트: 임계값을 넘으면 실패 종료.
	failed := false
	if *maxErr >= 0 && bad > *maxErr {
		fmt.Printf("FAIL: 오류/5xx %d개 > 허용 %d개\n", bad, *maxErr)
		failed = true
	}
	if *maxP99 > 0 && float64(p99.Milliseconds()) > *maxP99 {
		fmt.Printf("FAIL: p99 %dms > 상한 %.0fms\n", p99.Milliseconds(), *maxP99)
		failed = true
	}
	if failed {
		os.Exit(1)
	}
	if *maxP99 > 0 || *maxErr > 0 {
		fmt.Println("PASS: 임계값 통과")
	}
}

func doPost(client *http.Client, endpoint, user string) int {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, nil)
	if err != nil {
		return 0
	}
	req.Header.Set("X-User-Id", user)
	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

// report 는 요약을 출력하고 (p99, 오류·5xx 수)를 돌려준다(CI 게이트용).
func report(label string, n int, elapsed time.Duration, lat []time.Duration, codes []int) (time.Duration, int) {
	sorted := append([]time.Duration(nil), lat...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	pct := func(p float64) time.Duration {
		if len(sorted) == 0 {
			return 0
		}
		idx := int(p / 100 * float64(len(sorted)-1))
		return sorted[idx]
	}
	var sum time.Duration
	for _, d := range lat {
		sum += d
	}
	byCode := map[int]int{}
	for _, c := range codes {
		byCode[c]++
	}

	if label != "" {
		fmt.Printf("── %s ──\n", label)
	}
	fmt.Printf("요청 %d개, 소요 %.2fs, 처리량 %.0f req/s\n", n, elapsed.Seconds(), float64(n)/elapsed.Seconds())
	fmt.Printf("지연  평균 %s  p50 %s  p95 %s  p99 %s  max %s\n",
		round(sum/time.Duration(n)), round(pct(50)), round(pct(95)), round(pct(99)), round(pct(100)))
	fmt.Printf("상태코드 ")
	for code, cnt := range byCode {
		name := strconv.Itoa(code)
		if code == 0 {
			name = "err"
		}
		fmt.Printf("%s=%d ", name, cnt)
	}
	fmt.Println()

	// 오류(연결 실패=0) + 5xx 합계.
	bad := 0
	for code, cnt := range byCode {
		if code == 0 || code >= 500 {
			bad += cnt
		}
	}
	return pct(99), bad
}

func round(d time.Duration) time.Duration { return d.Round(100 * time.Microsecond) }
