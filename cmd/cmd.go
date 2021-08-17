package cmd

import (
	"bufio"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/antonfisher/nested-logrus-formatter"
	"github.com/mmcdole/gofeed"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	feeds    string
	outdir   string
	cachedir string
	verbose  bool
	cmd      = &cobra.Command{
		Use:   "rss-to-html [-f feeds.txt]",
		Short: "Command line tool to save RSS articles as html files.",
		RunE:  run,
	}
)

func init() {
	cmd.Flags().SortFlags = false
	cmd.PersistentFlags().SortFlags = false

	flags := cmd.PersistentFlags()
	{
		flags.StringVarP(&feeds, "feeds", "f", "./feeds.txt", "feed list")
		flags.StringVarP(&outdir, "outdir", "o", ".", "output directory")
		flags.StringVarP(&cachedir, "cachedir", "c", "./seen", "cache directory")
		flags.BoolVarP(&verbose, "verbose", "v", false, "verbose")
	}

	logrus.SetFormatter(&formatter.Formatter{
		TimestampFormat: "2006-01-02 15:04:05",
		//NoColors:        true,
		HideKeys:    true,
		CallerFirst: true,
		FieldsOrder: []string{"feed", "feedItem", "link", "file"},
	})
}
func run(c *cobra.Command, args []string) (err error) {
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	// read feeds
	var urls []string
	file, err := os.OpenFile(feeds, os.O_RDONLY, 0766)
	{
		if errors.Is(err, os.ErrNotExist) {
			file, err = os.Create(feeds)
		}
		if err != nil {
			logrus.WithError(err).Fatalf("open %s failed", feeds)
			return
		}

		sc := bufio.NewScanner(file)
		for sc.Scan() {
			feed := strings.TrimSpace(sc.Text())
			switch {
			case feed == "":
				continue
			case strings.HasPrefix(feed, "//"):
				continue
			case strings.HasPrefix(feed, "#"):
				continue
			}
			urls = append(urls, feed)
		}
		err = sc.Err()
		_ = file.Close()

		if err != nil {
			logrus.WithError(err).Fatalf("scan %s failed", feeds)
			return
		}
	}
	if len(urls) == 0 {
		urls = args
	}
	if len(urls) == 0 {
		logrus.Errorf("no feeds given, put your feeds in %s", feeds)
		return
	}

	// mkdir
	err = os.MkdirAll(cachedir, 0766)
	if err != nil {
		return
	}

	for _, u := range urls {
		logger := logrus.WithField("feed", u)

		logger.Debugf("fetching %s", u)
		feed, err := fetchFeed(u)
		if err != nil {
			logger.WithError(err).Errorf("fetch failed")
			continue
		}

		logger.Debugf("processing %s", feed.Title)
		err = process(feed)
		if err != nil {
			logrus.WithError(err).Errorf("process feed %s error", feed.Title)
		}
	}

	return
}
func process(feed *gofeed.Feed) (err error) {
	for _, it := range feed.Items {
		item := NewFeedItem(it)
		logger := logrus.WithFields(logrus.Fields{
			"feed":  feed.Title,
			"title": item.Title,
		})

		html := htmlName(feed.Title, item.filename())

		seen := filepath.Join(cachedir, html)
		if _, e := os.Stat(seen); e == nil {
			logger.Debugf("seen before: %s", seen)
			continue
		}
		target := filepath.Join(outdir, html)
		if s, e := os.Stat(target); e == nil && s.Size() > 0 {
			logger.Infof("output exist: %s", target)
			continue
		}
		content, err := item.patchContent(feed)
		if err != nil {
			logger.Errorf("patch content failed: %s", err)
			continue
		}

		logger.Debugf("save %s", target)
		err = ioutil.WriteFile(target, []byte(content), 0666)
		if err != nil {
			logger.Fatal(err)
		}

		logger.Debugf("create %s", seen)
		fd, err := os.OpenFile(seen, os.O_CREATE|os.O_EXCL, 0666)
		if err == nil {
			_ = fd.Close()
		} else {
			logger.Errorf("cannot create cache %s", err)
		}
	}

	return
}
func Execute() {
	err := cmd.Execute()
	if err != nil {
		logrus.Fatal(err)
	}
}
