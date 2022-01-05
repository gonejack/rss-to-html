package main

import (
	"github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"

	"github.com/gonejack/rss-to-html/cmd"
)

func init() {
	logrus.SetFormatter(&formatter.Formatter{
		TimestampFormat: "2006-01-02 15:04:05",
		//NoColors:        true,
		HideKeys:    true,
		CallerFirst: true,
	})
}
func main() {
	var c cmd.RSSToHtml
	if err := c.Run(); err != nil {
		logrus.Fatal(err)
	}
}
