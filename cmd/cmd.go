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

func (c *RSSToHtml) Run() (err error) {
	kong.Parse(&c.options,
		kong.Name("rss-to-html"),
		kong.Description("Command line tool to save RSS articles as html files."),
		kong.UsageOnError(),
	)
	switch {
	case c.About:
		fmt.Println("Visit https://github.com/gonejack/rss-to-html")
		return
	case c.Verbose:
		logrus.SetLevel(logrus.DebugLevel)
	}

	err = os.MkdirAll(c.Output, 0766)
	if err != nil {
		return fmt.Errorf("output directory %s already exist", c.Output)
	}

	err = c.parseFeeds()
	if err != nil {
		return fmt.Errorf("parse %s failed: %s", c.Feeds, err)
	}
	if len(c.feeds) == 0 {
		return fmt.Errorf("no feeds given, put your feeds in %s", c.Feeds)
	}

	c.db, err = gorm.Open("sqlite3", c.Db)
	if err != nil {
		return fmt.Errorf("open db file %s failed: %s", c.Db, err)
	}
	c.db.AutoMigrate(new(record))
	c.db.Unscoped().Delete(new(record), "updated_at < ?", carbon.Now().SubMonth().String())
	defer c.db.Close()

	return c.run()
}
func (c *RSSToHtml) run() (err error) {
	for _, url := range c.feeds {
		logger := logrus.WithField("feed", url)

		logger.Debugf("fetching")
		feed, err := fetchFeed(url)
		if err != nil {
			return fmt.Errorf("fetch %s failed: %w", url, err)
		}

		logger.Debugf("processing %s", feed.Title)
		err = c.process(feed)
		if err != nil {
			return fmt.Errorf("process %s failed: %w", feed.Title, err)
		}
	}
	return
}
func (c *RSSToHtml) process(feed *gofeed.Feed) (err error) {
	for _, ft := range feed.Items {
		item := NewFeedItem(ft)
		name := htmlName(feed.Title, item.filename())
		logger := logrus.WithField("html", name)
		output := filepath.Join(c.Output, name)

		stat, err := os.Stat(output)
		if err == nil && stat.Size() > 0 {
			logger.Infof("output exist")
			continue
		}
		content, err := item.patchContent(feed)
		if err != nil {
			logger.Errorf("patch content failed: %s", err)
			continue
		}

		var rcd record
		c.db.First(&rcd, "filename == ?", name)
		if rcd.Content == content {
			logger.Debugf("skip")
			continue
		}

		logger.Debugf("saving")
		err = os.WriteFile(output, []byte(content), 0666)
		if err != nil {
			return fmt.Errorf("write %s failed: %s", output, err)
		}
		_ = os.Chtimes(output, item.PublishedParsed.UTC(), item.PublishedParsed.UTC())
		c.db.Save(&record{Filename: name, Content: content})
	}
	return
}
func (c *RSSToHtml) parseFeeds() (err error) {
	if len(c.Args) > 0 {
		c.feeds = c.Args
		return
	}

	f, err := os.OpenFile(c.Feeds, os.O_RDONLY, 0766)
	if errors.Is(err, os.ErrNotExist) {
		f, err = os.Create(c.Feeds)
	}
	if err != nil {
		return fmt.Errorf("open %s failed", c.Feeds)
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
		c.feeds = append(c.feeds, feed)
	}

	err = scan.Err()
	if err != nil {
		return fmt.Errorf("scan %s failed", c.Feeds)
	}

	return
}
