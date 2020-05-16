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
	extents      []string = []string{".txt", ".go", ".html", ".js", ".css"}
)

func main() {
	flag.StringVar(&content, "con", "", "The content to be replaced.")
	flag.StringVar(&rep, "rep", "", "The content to replace.")
	flag.BoolVar(&recursive, "rec", false, "Allow repalce recursivly.")

	flag.Parse()

	src = flag.Args()

	start := time.Now()

	options := []cfunc{}
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
