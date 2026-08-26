package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
)

// TestConcurrentReadWriteRaceSafe 通过并发读写同一电梯验证仓储深拷贝策略：
// 修复前 store 读接口直接返回内部指针，此测试会在 -race 下报数据竞态。
func TestConcurrentReadWriteRaceSafe(t *testing.T) {
	srv := newTestServer(t)
	client := &http.Client{}

	var wg sync.WaitGroup
	errCh := make(chan error, 64)

	postBody := map[string]any{
		"floor": 5, "position": "5F", "direction": "idle",
		"door": "closed", "leveling": true, "fault_code": "",
		"passenger_signal": "none",
	}
	buf, err := json.Marshal(postBody)
	if err != nil {
		t.Fatal(err)
	}

	workers := 8
	iterations := 25
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				if i%2 == 0 {
					resp, err := client.Get(srv.URL + "/api/elevators/ELEV-001")
					if err != nil {
						errCh <- fmt.Errorf("GET: %w", err)
						return
					}
					resp.Body.Close()
					continue
				}
				req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/elevators/ELEV-001/states", bytes.NewReader(buf))
				if err != nil {
					errCh <- fmt.Errorf("new request: %w", err)
					return
				}
				req.Header.Set("Content-Type", "application/json")
				resp, err := client.Do(req)
				if err != nil {
					errCh <- fmt.Errorf("POST: %w", err)
					return
				}
				resp.Body.Close()
				if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusUnprocessableEntity {
					errCh <- fmt.Errorf("unexpected POST status %d", resp.StatusCode)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}
}
