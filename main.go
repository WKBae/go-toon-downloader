package main

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/wkbae/go-toon-downloader/content"
	"github.com/wkbae/go-toon-downloader/detail"
	"github.com/wkbae/go-toon-downloader/list"
	"github.com/wkbae/go-toon-downloader/metadata"
	"github.com/wkbae/go-toon-downloader/viewer"
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
	ch1 := make(chan list.Entry)
	go bufferingWorker(entryCh, ch1)
	ch21 := make(chan list.Entry)
	ch22 := make(chan list.Entry)
	go cloner(ch1, ch21, ch22)

	wg := &sync.WaitGroup{}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go downloadEntryWorker(wg, toonId, ch21, errCh)
	}
	wg.Add(1)
	go viewerGenerateWorker(wg, toonId, ch22, errCh)
	go reportingWorker()
	wg.Wait()

	fmt.Println("\nFinished! Downloaded:", downloadCount)
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

func cloner(in <-chan list.Entry, out1, out2 chan<- list.Entry) {
	for entry := range in {
		var sent1, sent2 = false, false
		select {
		case out1 <- entry:
			sent1 = true
		case out2 <- entry:
			sent2 = true
		}
		if !sent1 {
			out1 <- entry
		}
		if !sent2 {
			out2 <- entry
		}
	}
	close(out1)
	close(out2)
}

func downloadEntryWorker(wg *sync.WaitGroup, toonId int, ch <-chan list.Entry, errCh chan<- error) {
	defer wg.Done()
	for entry := range ch {
		d := detail.Loader{
			DetailUrl: entry.DetailUrl,
		}
		urls, err := d.GetUrls()
		if err != nil {
			errCh <- err
			continue
		}

		dirName := fmt.Sprintf("result/%d/%d/", toonId, entry.Number)
		err = os.MkdirAll(dirName, 0700)
		if err != nil {
			errCh <- errors.Wrapf(err, "failed to make directory %s", dirName)
			continue
		}

		c := content.Loader{
			ImageUrls:    urls,
			DownloadPath: dirName,
			Parallelism:  8,
		}
		c.Run(errCh)

		f := viewer.Files{
			BasePath: dirName,
		}
		filenames := make([]string, len(urls))
		for i, s := range urls {
			u, err := url.Parse(s)
			if err != nil {
				continue
			}
			_, name := path.Split(u.Path)
			filenames[i] = name
		}
		err = f.Write(filenames)
		if err != nil {
			errCh <- err
			continue
		}

		atomic.AddInt32(&downloadCount, 1)
	}
}

func viewerGenerateWorker(wg *sync.WaitGroup, id int, ch <-chan list.Entry, errCh chan<- error) {
	defer wg.Done()

	meta, err := metadata.Get(id)
	if err != nil {
		errCh <- err
		meta = metadata.Metadata{Id: id}
	}
	info := viewer.Info{
		Id:          id,
		Title:       meta.Title,
		Author:      meta.Author,
		Description: meta.Description,
	}

	for entry := range ch {
		var thumbName string
		if u, err := url.Parse(entry.ThumbnailUrl); err == nil {
			_, thumbName = path.Split(u.Path)
		}
		ent := viewer.Entry{
			Number:            entry.Number,
			Title:             entry.Title,
			Path:              fmt.Sprintf("%d/", entry.Number),
			ThumbnailFileName: thumbName,
		}
		info.Entries = append(info.Entries, ent)
	}
	sort.Slice(info.Entries, func(i, j int) bool {
		return info.Entries[i].Number < info.Entries[j].Number
	})

	m := viewer.Meta{
		BasePath: fmt.Sprintf("result/%d/", id),
	}
	err = m.Write(info)
	if err != nil {
		errCh <- err
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
