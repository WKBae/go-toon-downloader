package content

import (
	"github.com/pkg/errors"
	"go-ntoon-downloader/fetcher"
	"os"
	"path"
	"sync"
)

type Loader struct {
	ImageUrls []string
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

	for _, url := range l.ImageUrls {
		downloadCh <- url
	}
	close(downloadCh)

	wg.Wait()
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
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return errors.Wrapf(err, "failed to open file \"%s\"", fileName)
	}

	stat, err := file.Stat()
	if err != nil {
		return errors.Wrapf(err, "failed to get stat of file \"%s\"", fileName)
	}
	err = fetcher.GetTo(url, file, stat.Size())
	if err != nil {
		return err
	}

	return nil
}
