package fetcher

import (
	"bufio"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"
)

const maxRetries = 10

func backoffTime(base time.Duration, retries int) time.Duration {
	if retries > 62 {
		retries = 62
	}
	// exponential backoff
	maxDur := base * (time.Duration(1) << retries)
	// ... with jitter
	dur := rand.Int63n(int64(maxDur))
	return time.Duration(dur)
}

func Get(url string) (*http.Response, error) {
	var (
		resp *http.Response
		err  error
	)
	for trial := 0; trial < maxRetries; trial++ {
		resp, err = HttpClient.Get(url)
		if err != nil || resp.StatusCode/100 == 5 {
			time.Sleep(backoffTime(50*time.Millisecond, trial))
			continue
		}
		if resp.StatusCode != 200 {
			_ = resp.Body.Close()
			return nil, errors.New(fmt.Sprintf("server responded \"%s\" with status: %s", url, resp.Status))
		}
		return resp, err
	}

	if err != nil {
		return nil, errors.Wrapf(err, "error while getting resource \"%s\"", url)
	} else if resp.StatusCode != 200 {
		_ = resp.Body.Close()
		return nil, errors.New(fmt.Sprintf("failed to fetch after %d retries: %s", maxRetries, resp.Status))
	}
	return resp, err
}

func GetTo(url string, w io.WriteSeeker) error {
	var err error
	for trial := 0; trial < maxRetries; trial++ {
		var resp *http.Response
		resp, err = Get(url)
		if err != nil {
			return err
		}
		r := bufio.NewReader(resp.Body)
		_, err = r.WriteTo(w)
		_ = resp.Body.Close()
		if err != nil {
			// retry on network temporary or timeout errors
			if err, ok := err.(net.Error); ok {
				if err.Temporary() || err.Timeout() {
					time.Sleep(backoffTime(50*time.Millisecond, trial))
					continue
				}
			}
			// retry on goaway error
			if strings.Contains(err.Error(), "http2: server sent GOAWAY") {
				_, err := w.Seek(0, io.SeekStart)
				if err != nil {
					return errors.WithStack(err)
				}
				time.Sleep(backoffTime(50*time.Millisecond, trial))
				continue
			}
			// retry on connection abort/reset
			if isDisconnectedError(err) {
				time.Sleep(backoffTime(50*time.Millisecond, trial))
				continue
			}

			return errors.WithStack(err)
		}
		return nil
	}
	return err
}
