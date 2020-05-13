package main

import (
	"flag"
	"fmt"
	"time"
)

var (
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
	if len(flag.Args()) == 0 {
		flag.Usage()
		return
	}

	src = flag.Args()

	start := time.Now()

	replacer := newReplacer(withPaths(src...), withContent(content), withRep(rep), withRec(recursive))
	replacer.init()
	replacer.start()

	reportContent := "Replace \033[1;31m%d\033[0m files;\nTotal \033[1;31m%.4f\033[0m seconds;\n"
	fmt.Printf(reportContent, count, time.Now().Sub(start).Seconds())
}
