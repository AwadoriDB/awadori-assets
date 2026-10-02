package network

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/awadoriproj/awadori-assets/rich"
)

const MaxRetries = 3

type Job struct {
	Name string
	URL  string
	Dst  string
	Size int64
}

var client = &http.Client{
	Timeout:   1200 * time.Second,
	Transport: &http.Transport{DisableKeepAlives: true},
}

func get(url string) (*http.Response, error) {
	res, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return nil, fmt.Errorf("status %s", res.Status)
	}
	return res, nil
}

func Bytes(url string) ([]byte, error) {
	var lastErr error
	for i := 0; i < MaxRetries; i++ {
		res, err := get(url)
		if err != nil {
			lastErr = err
			continue
		}
		data, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		return data, nil
	}
	return nil, lastErr
}

func fetch(job Job) error {
	res, err := get(job.URL)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if err := os.MkdirAll(filepath.Dir(job.Dst), 0755); err != nil {
		return err
	}
	tmp := job.Dst + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	written, err := io.Copy(f, res.Body)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && job.Size > 0 && written != job.Size {
		err = fmt.Errorf("size mismatch: got %d, catalog says %d", written, job.Size)
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, job.Dst)
}

func Download(jobs []Job, workers int, done func(index int)) []int {
	var (
		mu     sync.Mutex
		failed []int
		count  int
		wg     sync.WaitGroup
	)
	queue := make(chan int)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range queue {
				job := jobs[i]
				var err error
				for attempt := 1; attempt <= MaxRetries; attempt++ {
					if err = fetch(job); err == nil {
						break
					}
					rich.Warning("Failed to download %s (%v), retrying...(%d/%d)", job.Name, err, attempt, MaxRetries)
				}
				mu.Lock()
				if err != nil {
					failed = append(failed, i)
					rich.Error("Giving up on %s.", job.Name)
				} else {
					count++
					rich.Info("(%d/%d) Download completed: %q.", count, len(jobs), job.Name)
					done(i)
				}
				mu.Unlock()
			}
		}()
	}
	for i := range jobs {
		queue <- i
	}
	close(queue)
	wg.Wait()
	return failed
}
