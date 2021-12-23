# rss-to-html

This command line saves RSS articles as .html files.

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/gonejack/rss-to-html)
![Build](https://github.com/gonejack/rss-to-html/actions/workflows/go.yml/badge.svg)
[![GitHub license](https://img.shields.io/github/license/gonejack/rss-to-html.svg?color=blue)](LICENSE)

### Install

```shell
> go get github.com/gonejack/rss-to-html
```

### Usage

- Save your feed urls into `feeds.txt`

```shell
> rss-to-html -f feeds.txt
```

- or pass URLs directly

```shell
> rss-to-html urls...
```

```
Flags:
  -h, --help                 Show context-sensitive help.
  -f, --feeds="feeds.txt"    Feed list file.
  -o, --output="./"          Output directory.
      --db="record.db"       sqlite3 db file.
  -v, --verbose              Verbose printing.
```
