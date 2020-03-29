package main

import (
	"fmt"
	"github.com/pkg/errors"
	"go-ntoon-downloader/image"
	"go-ntoon-downloader/list"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var entryCount int32
var downloadCount int32

func main() {
	errCh := make(chan error)
	go errorLogger(errCh)
	toonId := 12345
	l := list.Loader{
		TitleId:     toonId,
		Parallelism: 2,
	}
	entryCh := l.Start(errCh)
	downloadCh := make(chan list.Entry)

	go bufferingWorker(entryCh, downloadCh)

	wg := &sync.WaitGroup{}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go downloadEntryWorker(wg, toonId, downloadCh, errCh)
	}
	go reportingWorker()
	wg.Wait()

	fmt.Println("Finished! Downloaded:", downloadCount)
}

func errorLogger(ch <-chan error) {
	for err := range ch {
		fmt.Fprintf(os.Stderr, "Error: %T %+v\n", errors.Cause(err), err)
	}
}

func bufferingWorker(in <-chan list.Entry, out chan<- list.Entry) {
	var buffer []list.Entry
BufferLoop:
	for entry := range in {
		buffer = append(buffer, entry)
		atomic.AddInt32(&entryCount, 1)

		for len(buffer) > 0 {
			select {
			case page, ok := <-in:
				if !ok {
					break BufferLoop
				}
				buffer = append(buffer, page)
				atomic.AddInt32(&entryCount, 1)
			case out <- buffer[0]:
				buffer = buffer[1:]
			}
		}
	}
	for _, entry := range buffer {
		out <- entry
	}
	close(out)
}

func downloadEntryWorker(wg *sync.WaitGroup, toonId int, ch <-chan list.Entry, errCh chan<- error) {
	defer wg.Done()
	for entry := range ch {
		dirName := fmt.Sprintf("result/%d/%d/", toonId, entry.Number)
		err := os.MkdirAll(dirName, 0700)
		if err != nil {
			errCh <- errors.Wrapf(err, "failed to make directory %s", dirName)
			continue
		}
		l := image.Loader{
			DetailUrl:    entry.DetailUrl,
			DownloadPath: dirName,
			Parallelism:  8,
		}
		l.Run(errCh)
		atomic.AddInt32(&downloadCount, 1)
	}
}

func reportingWorker() {
	for {
		total := atomic.LoadInt32(&entryCount)
		downloaded := atomic.LoadInt32(&downloadCount)
		fmt.Printf("\rProgress: %d/%d", downloaded, total)
		time.Sleep(1 * time.Second)
	}
}
