package metadata

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
	"github.com/wkbae/go-toon-downloader/fetcher"
	"golang.org/x/net/html"
)

type Metadata struct {
	Id           int
	Title        string
	Author       string
	Description  string
	ThumbnailUrl string
}

const listUrlFormat = "https://comic.naver.com/webtoon/list.nhn?titleId=%d"

func Get(toonId int) (Metadata, error) {
	listUrl := fmt.Sprintf(listUrlFormat, toonId)
	resp, err := fetcher.Get(listUrl)
	if err != nil {
		return Metadata{}, errors.Wrapf(err, "failed to get metadata from \"%s\"", listUrl)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return Metadata{}, errors.Wrapf(err, "failed to parse body from \"%s\"", listUrl)
	}

	info := doc.Find(".comicinfo:has(.thumb):has(.detail)").Eq(0)

	thumbSrc := info.Find(".thumb img[src*=\"thumb\"]").AttrOr("src", "")
	var thumbUrlStr string
	thumbUrl, err := url.Parse(thumbSrc)
	if err == nil {
		baseUrl, _ := url.Parse(listUrl)
		absUrl := baseUrl.ResolveReference(thumbUrl)
		thumbUrlStr = absUrl.String()
	}

	detail := info.Find(".detail")
	header := detail.Find("h2")
	title := ""
	// find direct child text node
	for _, node := range header.Nodes {
		for n := node.FirstChild; n != nil; n = n.NextSibling {
			if n.Type == html.TextNode {
				title += n.Data
			}
		}
	}
	author := header.Find(".wrt_nm").Text()
	desc := detail.Find("p:not(.detail_info)").Text()

	return Metadata{
		Id:           toonId,
		Title:        strings.TrimSpace(title),
		Author:       strings.TrimSpace(author),
		Description:  strings.TrimSpace(desc),
		ThumbnailUrl: thumbUrlStr,
	}, nil
}
