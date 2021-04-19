package cmd

import (
	"context"
	"crypto/md5"
	"fmt"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

func md5str(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}
func htmlName(feedName, itemName string) string {
	return fmt.Sprintf("[%s]%s.html", strings.ReplaceAll(feedName, "/", "."), itemName)
}
func timeout(duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.TODO(), duration)
}
func timeout10s() (context.Context, context.CancelFunc) {
	return timeout(time.Second * 10)
}
func fetchFeed(url string) (*gofeed.Feed, error) {
	timeout, cancel := timeout10s()
	defer cancel()

	return gofeed.NewParser().ParseURLWithContext(url, timeout)
}
