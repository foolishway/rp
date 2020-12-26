package main

import (
	"flag"
	"fmt"
	"time"
)

var (
	bucket       chan struct{} = make(chan struct{}, 100)
	src          []string
	content, rep string
	recursive    bool
	count        int32
	extents      []string = []string{".txt", ".go", ".html", ".js", ".css", ".vue", ".ts"}
)

func main() {
	flag.StringVar(&content, "con", "", "The content to be replaced.")
	flag.StringVar(&rep, "rep", "", "The new content to replace.")
	flag.BoolVar(&recursive, "rec", false, "Allow repalce recursivly.")

	flag.Parse()

	if content == "" || rep == "" {
		flag.Usage()
		return
	}

	src = flag.Args()

	start := time.Now()

	options := []option{}
	if len(src) != 0 {
		options = append(options, withPaths(src...))
	}

	if content != "" {
		options = append(options, withContent(content))
	}

	if rep != "" {
		options = append(options, withRep(rep))
	}

	if recursive {
		options = append(options, withRec(recursive))
	}
	replacer := newReplacer(options...)
	replacer.init()
	replacer.start()

	reportContent := "Replace \033[1;31m%d\033[0m files;\nTotal \033[1;31m%.4f\033[0m seconds;\n"
	fmt.Printf(reportContent, count, time.Now().Sub(start).Seconds())
}
