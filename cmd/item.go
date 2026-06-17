package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html"
)

type item struct {
	*gofeed.Item
}

func (it *item) UUID() string {
	if it.GUID == "" {
		return it.Link
	}
	return it.GUID
}
func (it *item) Content() string {
	if it.Item.Content == "" {
		return it.Item.Description
	}
	return it.Item.Content
}
func (it *item) FullContent() (content string) {
	switch {
	case strings.Contains(it.Link, "darkserika.com"):
		htm, err := it.grabDoc()
		if err == nil {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(htm))
			if err != nil {
				break
			}
			if doc.Find("figure").Length() <= 0 {
				break
			}
			doc.Find("head > style, head > script, head > link[rel=stylesheet]").Remove()
			doc.Find("*").Contents().Each(func(i int, s *goquery.Selection) {
				if s.Get(0).Type == html.CommentNode {
					s.Remove()
				}
			})
			var sb strings.Builder
			sb.WriteString("<p>")
			doc.Find("figure > a > img").Each(func(i int, e *goquery.Selection) {
				_ = goquery.Render(&sb, e)
			})
			sb.WriteString("</p>")
			doc.Find("body").Children().Remove()
			doc.Find("body").AppendHtml(sb.String())
			x, _ := doc.Html()
			return x
		}
	}
	return it.Content()
}
func (it *item) filename() string {
	title := it.Title
	digest := md5str(it.UUID())[0:4]
	if len([]rune(title)) > 30 {
		title = string([]rune(title)[0:30]) + "..."
	}
	title = strings.ReplaceAll(title, "/", ".")
	return fmt.Sprintf("[%s.%s][%s]", title, digest, it.Item.PublishedParsed.Format("2006-01-02 15.04.05"))
}
func (it *item) header(feed *gofeed.Feed) string {
	const tpl = `
<p>
	<a title="Published: {published}" href="{link}" style="display:block; color: #000; padding-bottom: 10px; text-decoration: none; font-size:1em; font-weight: normal;">
		<span style="display: block; color: #666; font-size:1.0em; font-weight: normal;">{origin}</span>
		<span style="font-size: 1.5em;">{title}</span>
	</a>
</p>`

	replacer := strings.NewReplacer(
		"{link}", it.Link,
		"{origin}", html.EscapeString(feed.Title),
		"{published}", it.PublishedParsed.Format("2006-01-02 15:04:05"),
		"{title}", html.EscapeString(it.Title),
	)

	return replacer.Replace(tpl)
}
func (it *item) footer() string {
	const footerTPL = `<br><br>
<a style="display: block; display:inline-block; border-top: 1px solid #ccc; padding-top: 5px; color: #666; text-decoration: none;"
   href="${href}"
>${href}</a>
<p style="color:#999;">
Sent with <a style="color:#666; text-decoration:none; font-weight: bold;" href="https://github.com/gonejack/rss-to-html">RSS</a>
</p>`

	return strings.NewReplacer(
		"${href}", it.Link,
		"${pub_time}", it.PublishedParsed.Format("2006-01-02 15:04:05"),
	).Replace(footerTPL)
}
func (it *item) patchContent(feed *gofeed.Feed) (content string, err error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(it.FullContent()))
	if err != nil {
		return
	}

	if doc.Find("title").Length() == 0 {
		doc.Find("head").AppendHtml("<title></title>")
	}
	if doc.Find("title").Text() == "" {
		doc.Find("title").SetText(it.Title)
	}
	doc.Find("img").Each(func(_ int, img *goquery.Selection) {
		img.RemoveAttr("loading")
		img.RemoveAttr("srcset")

		src, _ := img.Attr("src")
		if src != "" {
			img.SetAttr("src", it.patchReference(src))
		}
	})
	doc.Find("iframe").Each(func(i int, iframe *goquery.Selection) {
		src, _ := iframe.Attr("src")
		if src != "" {
			iframe.ReplaceWithHtml(fmt.Sprintf(`<a href="%s">%s</a>`, src, src))
		}
	})
	doc.Find("script").Each(func(i int, script *goquery.Selection) {
		script.Remove()
	})
	doc.Find("body").PrependHtml(it.header(feed)).AppendHtml(it.footer())

	return doc.Html()
}
func (it *item) patchReference(ref string) string {
	u, err := url.Parse(it.Link)
	if err != nil {
		return ref
	}
	uu, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	if uu.Scheme == "" {
		uu.Scheme = u.Scheme
	}
	if uu.Host == "" {
		uu.Host = u.Host
	}
	return uu.String()
}
func (it *item) grabDoc() (html string, err error) {
	timeout, cancel := context.WithTimeout(context.TODO(), time.Second*20)
	defer cancel()

	rq, err := http.NewRequestWithContext(timeout, http.MethodGet, it.Link, nil)
	if err != nil {
		log.Printf("cannot request link %s", it.Link)
		return
	}
	rq.Header.Set("referer", it.Link)
	rq.Header.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:94.0) Gecko/20100101 Firefox/94.0")

	rp, err := http.DefaultClient.Do(rq)
	if err != nil {
		log.Printf("cannot grab link %s", it.Link)
		return
	}
	dat, err := io.ReadAll(rp.Body)
	if err != nil {
		log.Printf("cannot read response of %s", it.Link)
		return
	}
	return strings.ReplaceAll(string(dat), "<!--!-->", ""), nil
}

func NewFeedItem(gf *gofeed.Item) (i *item) {
	i = &item{
		Item: gf,
	}
	now := time.Now()
	if i.UpdatedParsed == nil {
		i.UpdatedParsed = &now
	}
	if i.PublishedParsed == nil {
		i.PublishedParsed = &now
	}
	*i.UpdatedParsed = i.UpdatedParsed.Local()
	*i.PublishedParsed = i.PublishedParsed.Local()
	return
}
