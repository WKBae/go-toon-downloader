package detail

import (
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
	"github.com/wkbae/go-toon-downloader/fetcher"
)

type Loader struct {
	DetailUrl string
}

func (l Loader) GetUrls() ([]string, error) {
	resp, err := fetcher.Get(l.DetailUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse body from \"%s\"", l.DetailUrl)
	}

	var urls []string
	baseUrl, err := url.Parse(l.DetailUrl)
	if err != nil {
		return nil, err
	}
	doc.Find(".wt_viewer img[id^=\"content_image_\"]").Each(func(_ int, sel *goquery.Selection) {
		src, ok := sel.Attr("src")
		if !ok {
			return
		}
		srcUrl, err := url.Parse(src)
		if err != nil {
			return
		}
		absUrl := baseUrl.ResolveReference(srcUrl)
		urls = append(urls, absUrl.String())
	})

	return urls, nil
}
