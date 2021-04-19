package cmd

import (
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
)

type leveItem struct {
	*gofeed.Item
}

func (o *leveItem) UUID() string {
	if o.Item.GUID == "" {
		return o.Item.Link
	}
	return o.Item.GUID
}
func (o *leveItem) Content() string {
	if o.Item.Content == "" {
		return o.Item.Description
	}
	return o.Item.Content
}
func (o *leveItem) filename() string {
	title := o.Title
	digest := md5str(o.UUID())[0:4]
	if len([]rune(title)) > 30 {
		title = string([]rune(title)[0:30]) + "..."
	}
	title = strings.ReplaceAll(title, "/", ".")
	return fmt.Sprintf("[%s.%s][%s]", title, digest, o.Item.PublishedParsed.Format("2006-01-02 15.04.05"))
}
func (o *leveItem) header(feed *gofeed.Feed) string {
	const tpl = `
<p>
	<a title="Published: {published}" href="{link}" style="display:block; color: #000; padding-bottom: 10px; text-decoration: none; font-size:1em; font-weight: normal;">
		<span style="display: block; color: #666; font-size:1.0em; font-weight: normal;">{origin}</span>
		<span style="font-size: 1.5em;">{title}</span>
	</a>
</p>`

	replacer := strings.NewReplacer(
		"{link}", o.Link,
		"{origin}", html.EscapeString(feed.Title),
		"{published}", o.PublishedParsed.Format("2006-01-02 15:04:05"),
		"{title}", html.EscapeString(o.Title),
	)

	return replacer.Replace(tpl)
}
func (o *leveItem) footer() string {
	const footerTPL = `<br><br>
<a style="display: block; display:inline-block; border-top: 1px solid #ccc; padding-top: 5px; color: #666; text-decoration: none;"
   href="${href}"
>${href}</a>
<p style="color:#999;">
Sent with <a style="color:#666; text-decoration:none; font-weight: bold;" href="https://github.com/gonejack/rss-to-html">RSS</a>
</p>`

	return strings.NewReplacer(
		"${href}", o.Link,
		"${pub_time}", o.PublishedParsed.Format("2006-01-02 15:04:05"),
	).Replace(footerTPL)
}
func (o *leveItem) patchContent(feed *gofeed.Feed) (content string, err error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(o.Content()))
	if err != nil {
		return
	}

	doc.Find("img").Each(func(i int, img *goquery.Selection) {
		src, _ := img.Attr("src")

		img.RemoveAttr("loading")
		img.RemoveAttr("srcset")

		if src != "" {
			img.SetAttr("src", o.patchRef(src))
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
	doc.Find("body").PrependHtml(o.header(feed)).AppendHtml(o.footer())

	if doc.Find("title").Length() == 0 {
		doc.Find("head").AppendHtml("<title></title>")
	}
	if doc.Find("title").Text() == "" {
		doc.Find("title").SetText(o.Item.Title)
	}

	return doc.Html()
}
func (o *leveItem) patchRef(ref string) string {
	itemURL, err := url.Parse(o.Item.Link)
	if err != nil {
		return ref
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	if refURL.Scheme == "" {
		refURL.Scheme = itemURL.Scheme
	}
	if refURL.Host == "" {
		refURL.Host = itemURL.Host
	}
	return refURL.String()
}

func newLeveItem(item *gofeed.Item) *leveItem {
	it := &leveItem{Item: item}

	return it
}
