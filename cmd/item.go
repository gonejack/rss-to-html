package cmd

import (
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
)

type item struct {
	*gofeed.Item
}

func (i *item) UUID() string {
	if i.GUID == "" {
		return i.Link
	}
	return i.GUID
}
func (i *item) Content() string {
	if i.Item.Content == "" {
		return i.Item.Description
	}
	return i.Item.Content
}
func (i *item) filename() string {
	title := i.Title
	digest := md5str(i.UUID())[0:4]
	if len([]rune(title)) > 30 {
		title = string([]rune(title)[0:30]) + "..."
	}
	title = strings.ReplaceAll(title, "/", ".")
	return fmt.Sprintf("[%s.%s][%s]", title, digest, i.Item.PublishedParsed.Format("2006-01-02 15.04.05"))
}
func (i *item) header(feed *gofeed.Feed) string {
	const tpl = `
<p>
	<a title="Published: {published}" href="{link}" style="display:block; color: #000; padding-bottom: 10px; text-decoration: none; font-size:1em; font-weight: normal;">
		<span style="display: block; color: #666; font-size:1.0em; font-weight: normal;">{origin}</span>
		<span style="font-size: 1.5em;">{title}</span>
	</a>
</p>`

	replacer := strings.NewReplacer(
		"{link}", i.Link,
		"{origin}", html.EscapeString(feed.Title),
		"{published}", i.PublishedParsed.Format("2006-01-02 15:04:05"),
		"{title}", html.EscapeString(i.Title),
	)

	return replacer.Replace(tpl)
}
func (i *item) footer() string {
	const footerTPL = `<br><br>
<a style="display: block; display:inline-block; border-top: 1px solid #ccc; padding-top: 5px; color: #666; text-decoration: none;"
   href="${href}"
>${href}</a>
<p style="color:#999;">
Sent with <a style="color:#666; text-decoration:none; font-weight: bold;" href="https://github.com/gonejack/rss-to-html">RSS</a>
</p>`

	return strings.NewReplacer(
		"${href}", i.Link,
		"${pub_time}", i.PublishedParsed.Format("2006-01-02 15:04:05"),
	).Replace(footerTPL)
}
func (i *item) patchContent(feed *gofeed.Feed) (content string, err error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(i.Content()))
	if err != nil {
		return
	}

	if doc.Find("title").Length() == 0 {
		doc.Find("head").AppendHtml("<title></title>")
	}
	if doc.Find("title").Text() == "" {
		doc.Find("title").SetText(i.Title)
	}
	doc.Find("img").Each(func(_ int, img *goquery.Selection) {
		img.RemoveAttr("loading")
		img.RemoveAttr("srcset")

		src, _ := img.Attr("src")
		if src != "" {
			img.SetAttr("src", i.patchReference(src))
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
	doc.Find("body").PrependHtml(i.header(feed)).AppendHtml(i.footer())

	return doc.Html()
}
func (i *item) patchReference(ref string) string {
	u, err := url.Parse(i.Link)
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
