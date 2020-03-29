package list

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
	"github.com/wkbae/go-toon-downloader/fetcher"
)

type Entry struct {
	Number       int
	Title        string
	ThumbnailUrl string
	DetailUrl    string
}

const (
	listUrlFormat = "https://comic.naver.com/webtoon/list.nhn?titleId=%d&page=%d"
)

var toonNumberPattern = regexp.MustCompile("[?&]no=(\\d+)(?:&.+|$)")

//var ErrAuthenticationRequired = fmt.Errorf("authentication required")

type Loader struct {
	TitleId     int
	Parallelism int
}

func (l Loader) Start(errCh chan<- error) <-chan Entry {
	pageCh := make(chan int)
	resCh := make(chan Entry)
	wg := &sync.WaitGroup{}

	wg.Add(l.Parallelism)
	go l.seekingWorker(wg, pageCh, resCh, errCh)
	for i := 1; i < l.Parallelism; i++ {
		go l.worker(wg, pageCh, resCh, errCh)
	}
	go func() {
		wg.Wait()
		close(resCh)
	}()

	return resCh
}

func (l Loader) seekingWorker(wg *sync.WaitGroup, pageCh chan int, resCh chan<- Entry, errCh chan<- error) {
	currentPage := 1
	// expandable buffered channel
	bufferedCh := make(chan int)
	go unlimitedBufferer(bufferedCh, pageCh)
	for {
		entries, otherPages, err := l.getEntriesWithPaginator(currentPage)
		if err != nil {
			errCh <- err
			break
		}
		for _, entry := range entries {
			resCh <- entry
		}
		maxPage := currentPage
		for _, page := range otherPages {
			if page > currentPage {
				bufferedCh <- page
			}
			if page > maxPage {
				maxPage = page
			}
		}
		if maxPage == currentPage {
			break
		}
		currentPage = maxPage + 1
	}
	close(bufferedCh)

	// run as a normal worker after completing to seek
	l.worker(wg, pageCh, resCh, errCh)
}

func unlimitedBufferer(in <-chan int, out chan<- int) {
	var pages []int
ConsumeLoop:
	for page := range in {
		pages = append(pages, page)
		for len(pages) > 0 {
			select {
			case page, ok := <-in:
				if !ok {
					break ConsumeLoop
				}
				pages = append(pages, page)
			case out <- pages[0]:
				pages = pages[1:]
			}
		}
	}
	for _, page := range pages {
		out <- page
	}
	close(out)
}

func (l Loader) worker(wg *sync.WaitGroup, pageCh <-chan int, resCh chan<- Entry, errCh chan<- error) {
	defer wg.Done()
	for page := range pageCh {
		entries, err := l.getEntries(page)
		if err != nil {
			errCh <- err
			continue
		}
		for _, entry := range entries {
			resCh <- entry
		}
	}
}

func (l Loader) getUrl(page int) *url.URL {
	u, err := url.Parse(fmt.Sprintf(listUrlFormat, l.TitleId, page))
	if err != nil {
		panic(err)
	}
	return u
}

func (l Loader) loadPage(url string) (*goquery.Document, error) {
	resp, err := fetcher.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse body from \"%s\"", url)
	}
	return doc, nil
}

func (l Loader) getEntriesWithPaginator(page int) ([]Entry, []int, error) {
	url := l.getUrl(page)
	doc, err := l.loadPage(url.String())
	if err != nil {
		return nil, nil, err
	}

	otherPages, currPage := parsePaginator(doc, page)
	if currPage != page {
		// we already passed the maximum page
		return nil, nil, nil
	}

	entries := parseEntries(doc, url)

	return entries, otherPages, nil
}

func (l Loader) getEntries(page int) ([]Entry, error) {
	url := l.getUrl(page)
	doc, err := l.loadPage(url.String())
	if err != nil {
		return nil, err
	}

	entries := parseEntries(doc, url)

	return entries, nil
}

func parseEntries(doc *goquery.Document, baseUrl *url.URL) []Entry {
	var entries []Entry
	doc.Find(".viewList :haschild(.title):has(img[src*=thumb])").Each(func(_ int, sel *goquery.Selection) {
		thumbImg := sel.Find("img[src*=thumb]")
		thumbPath := thumbImg.AttrOr("src", "")

		titleLink := sel.Find(".title a[href*=\"detail.nhn\"]")
		title := titleLink.Text()
		detailPath, ok := titleLink.Attr("href")
		if !ok {
			return
		}
		mat := toonNumberPattern.FindStringSubmatch(detailPath)
		if len(mat) < 2 {
			return
		}
		toonNumber, err := strconv.Atoi(mat[1])
		if err != nil {
			return
		}

		thumbUrl, err := url.Parse(thumbPath)
		if err != nil {
			return
		}
		detailUrl, err := url.Parse(detailPath)
		if err != nil {
			return
		}

		entries = append(entries, Entry{
			Number:       toonNumber,
			Title:        title,
			ThumbnailUrl: baseUrl.ResolveReference(thumbUrl).String(),
			DetailUrl:    baseUrl.ResolveReference(detailUrl).String(),
		})
	})
	return entries
}

func parsePaginator(doc *goquery.Document, sourcePage int) (otherPages []int, currentPage int) {
	doc.Find(".paginate a.page .num_page").Each(func(_ int, sel *goquery.Selection) {
		page, err := strconv.Atoi(sel.Text())
		if err != nil {
			return
		}
		if page == sourcePage {
			return
		}
		otherPages = append(otherPages, page)
	})

	currPage := doc.Find(".paginate .page:not(a) .num_page").Text()
	var err error
	currentPage, err = strconv.Atoi(currPage)
	if err != nil {
		currentPage = -1
	}

	return
}
