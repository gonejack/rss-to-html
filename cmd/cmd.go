package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/mmcdole/gofeed"
	"github.com/sirupsen/logrus"
	"github.com/uniplaces/carbon"
)

type RSSToHtml struct {
	options
	feeds []string
	db    *gorm.DB
}
type options struct {
	Feeds   string `short:"f" default:"feeds.txt" help:"Feed list file."`
	Output  string `short:"o" default:"./" help:"Output directory."`
	Db      string `default:"record.db" help:"SQLLite3 db file."`
	Verbose bool   `short:"v" help:"Verbose printing."`
	About   bool   `help:"Show about."`

	Args []string `name:"feed" arg:"" optional:""`
}
type record struct {
	gorm.Model
	Filename string `gorm:"index"`
	Content  string
}

func (r *RSSToHtml) Run() (err error) {
	kong.Parse(&r.options,
		kong.Name("rss-to-html"),
		kong.Description("Command line tool to save RSS articles as html files."),
		kong.UsageOnError(),
	)
	switch {
	case r.About:
		fmt.Println("Visit https://github.com/gonejack/rss-to-html")
		return
	case r.Verbose:
		logrus.SetLevel(logrus.DebugLevel)
	}

	err = os.MkdirAll(r.Output, 0766)
	if err != nil {
		return fmt.Errorf("output directory %s already exist", r.Output)
	}

	err = r.parseFeeds()
	if err != nil {
		return fmt.Errorf("parse %s failed: %s", r.Feeds, err)
	}
	if len(r.feeds) == 0 {
		return fmt.Errorf("no feeds given, put your feeds in %s", r.Feeds)
	}

	r.db, err = gorm.Open("sqlite3", r.Db)
	if err != nil {
		return fmt.Errorf("open db file %s failed: %s", r.Db, err)
	}
	r.db.AutoMigrate(new(record))
	r.db.Unscoped().Delete(new(record), "updated_at < ?", carbon.Now().SubMonth().String())
	defer r.db.Close()

	return r.run()
}
func (r *RSSToHtml) run() (err error) {
	for _, url := range r.feeds {
		logger := logrus.WithField("feed", url)

		logger.Debugf("fetching")
		feed, err := fetchFeed(url)
		if err != nil {
			return fmt.Errorf("fetch %s failed: %w", url, err)
		}

		logger.Debugf("processing %s", feed.Title)
		err = r.process(feed)
		if err != nil {
			return fmt.Errorf("process %s failed: %w", feed.Title, err)
		}
	}
	return
}
func (r *RSSToHtml) process(feed *gofeed.Feed) (err error) {
	for _, it := range feed.Items {
		item := NewFeedItem(it)
		html := htmlName(feed.Title, item.filename())
		logger := logrus.WithField("html", html)

		target := filepath.Join(r.Output, html)
		if s, e := os.Stat(target); e == nil && s.Size() > 0 {
			logger.Infof("output exist")
			continue
		}
		content, err := item.patchContent(feed)
		if err != nil {
			logger.Errorf("patch content failed: %s", err)
			continue
		}

		var rec record
		r.db.First(&rec, "filename == ?", html)
		if rec.Content == content {
			logger.Debugf("skip")
			continue
		}

		logger.Debugf("saving")
		err = os.WriteFile(target, []byte(content), 0666)
		if err != nil {
			return fmt.Errorf("write %s failed: %s", target, err)
		}
		_ = os.Chtimes(target, item.PublishedParsed.UTC(), item.PublishedParsed.UTC())

		r.db.Save(&record{Filename: html, Content: content})
	}

	return
}
func (r *RSSToHtml) parseFeeds() (err error) {
	if len(r.Args) > 0 {
		r.feeds = r.Args
		return
	}

	f, err := os.OpenFile(r.Feeds, os.O_RDONLY, 0766)
	if errors.Is(err, os.ErrNotExist) {
		f, err = os.Create(r.Feeds)
	}
	if err != nil {
		return fmt.Errorf("open %s failed", r.Feeds)
	}
	defer f.Close()

	scan := bufio.NewScanner(f)
	for scan.Scan() {
		feed := strings.TrimSpace(scan.Text())
		switch {
		case feed == "":
			continue
		case strings.HasPrefix(feed, "//"):
			continue
		case strings.HasPrefix(feed, "#"):
			continue
		}
		r.feeds = append(r.feeds, feed)
	}

	err = scan.Err()
	if err != nil {
		return fmt.Errorf("scan %s failed", r.Feeds)
	}

	return
}

func New() *RSSToHtml {
	return new(RSSToHtml)
}
