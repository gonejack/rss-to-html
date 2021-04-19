# rss-to-html
Command line tool to save RSS articles as html files.

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/gonejack/rss-to-html)
![Build](https://github.com/gonejack/rss-to-html/actions/workflows/go.yml/badge.svg)
[![GitHub license](https://img.shields.io/github/license/gonejack/rss-to-html.svg?color=blue)](LICENSE)

### Install
```shell
> go get github.com/gonejack/rss-to-html
```

### Usage

Save your feed urls into `feeds.txt`
```shell
> rss-to-html -f feeds.txt
```
```
Usage:
  rss-to-html [-f feeds.txt] [flags]

Flags:
  -f, --feeds string    feed list (default "./feeds.txt")
  -o, --outdir string   output directory (default ".")
  -v, --verbose         verbose
  -h, --help            help for rss-to-html
```
