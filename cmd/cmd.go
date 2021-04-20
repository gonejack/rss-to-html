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
	feeds   string
	outdir  string
	verbose bool
	cmd     = &cobra.Command{
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

	var feedURLs []string
	file, err := os.OpenFile(feeds, os.O_RDONLY, 0766)
	if errors.Is(err, os.ErrNotExist) {
		file, err = os.Create(feeds)
	}
	if err == nil {
		scan := bufio.NewScanner(file)
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
			feedURLs = append(feedURLs, feed)
		}
		err = scan.Err()
		_ = file.Close()
	}
	if err != nil {
		logrus.WithError(err).Fatalf("parse %s failed", feeds)
		return
	}

	if len(feedURLs) == 0 {
		logrus.Errorf("no feeds given, put your feeds in %s", feeds)
		return
	}

	for _, feedURL := range feedURLs {
		log := logrus.WithField("feed", feedURL)

		log.Debugf("fetching")
		feed, err := fetchFeed(feedURL)
		if err != nil {
			log.WithError(err).Errorf("fetch failed")
			continue
		}

		log.Debugf("processing")
		err = process(feed)
		if err != nil {
			logrus.WithError(err).Errorf("process feed %s error", feed.Title)
		}
	}

	return
}
func process(feed *gofeed.Feed) (err error) {
	log := logrus.WithField("feed", feed.Title)

	for _, it := range feed.Items {
		item := newFeedItem(it)

		log := log.WithFields(logrus.Fields{
			"feed":  feed.Title,
			"title": item.Title,
		})

		target := filepath.Join(outdir, htmlName(feed.Title, item.filename()))
		if s, e := os.Stat(target); e == nil && s.Size() > 0 {
			log.Debugf("skip")
			continue
		}

		content, err := item.patchContent(feed)
		if err != nil {
			log.Errorf("patch content failed: %s", err)
			continue
		}

		log.Debugf("save")
		err = ioutil.WriteFile(target, []byte(content), 0666)
		if err != nil {
			log.Fatal(err)
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
