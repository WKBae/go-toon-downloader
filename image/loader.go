package image

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
	"go-ntoon-downloader/fetcher"
	"os"
	"path"
	"sync"
)

type Loader struct {
	DetailUrl    string
	DownloadPath string
	Parallelism  int
}

func (l Loader) Run(errCh chan<- error) {
	downloadCh := make(chan string)
	wg := &sync.WaitGroup{}
	wg.Add(l.Parallelism)
	for i := 0; i < l.Parallelism; i++ {
		go l.downloadWorker(wg, downloadCh, errCh)
	}
	l.loadUrls(downloadCh, errCh)
	wg.Wait()
}

func (l Loader) loadUrls(downloadCh chan<- string, errCh chan<- error) {
	resp, err := fetcher.Get(l.DetailUrl)
	if err != nil {
		errCh <- err
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		errCh <- errors.Wrapf(err, "failed to parse body from \"%s\"", l.DetailUrl)
		return
	}

	doc.Find(".wt_viewer img[id^=\"content_image_\"]").Each(func(_ int, sel *goquery.Selection) {
		src, ok := sel.Attr("src")
		if !ok {
			return
		}
		downloadCh <- src
	})

	close(downloadCh)
}

func (l Loader) downloadWorker(wg *sync.WaitGroup, downloadCh <-chan string, errCh chan<- error) {
	defer wg.Done()
	for url := range downloadCh {
		err := l.downloadFile(url)
		if err != nil {
			errCh <- err
		}
	}
}

func (l Loader) downloadFile(url string) error {
	_, name := path.Split(url)
	fileName := path.Join(l.DownloadPath, name)
	file, err := os.Create(fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to create file \"%s\"", fileName)
	}

	err = fetcher.GetTo(url, file)
	if err != nil {
		return err
	}

	return nil
}
