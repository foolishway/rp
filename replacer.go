package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

type Replacer struct {
	wg        sync.WaitGroup
	paths     []string
	extents   map[string]struct{}
	content   string
	replace   string
	recursive bool
	ch        chan string
}

type cfunc func(*Replacer)

func newRepalcer(cfuncs ...cfunc) *Replacer {
	var wg sync.WaitGroup
	defaultPaths := []string{"./"}
	defaultExtents := map[string]struct{}{".go": {}, ".js": {}, ".jsx": {}, ".html": {}, ".txt": {}}
	ch := make(chan string, 10)
	replace := &Replacer{wg: wg, paths: defaultPaths, extents: defaultExtents, ch: ch}
	for _, f := range cfuncs {
		f(replace)
	}
	return replace
}

func (r *Replacer) init() {
	cpuNums := runtime.NumCPU()
	for i := 0; i < cpuNums; i++ {
		go extracter(&r.wg, r.ch, r.content, r.replace)
	}
}

func (r *Replacer) start() {
	var once sync.Once
	once.Do(r.init)

	dumper := func(path string) {
		r.wg.Add(1)
		r.ch <- path
	}
	for _, path := range r.paths {
		s, err := os.Stat(path)
		if os.IsNotExist(err) {
			log.Printf("%s not found", path)
			continue
		}
		if s.IsDir() {
			if r.recursive {
				filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
					if !info.IsDir() {
						if _, ok := r.extents[filepath.Ext(info.Name())]; ok {
							dumper(p)
						}
					}
					return nil
				})
			} else {
				files, err := ioutil.ReadDir(path)
				if err != nil {
					log.Printf("Read dir %s error: %v", path, err)
					continue
				}

				for _, f := range files {
					if _, ok := r.extents[filepath.Ext(f.Name())]; ok {
						dumper(filepath.Join(path, f.Name()))
					}
				}
			}
		} else {
			if _, ok := r.extents[path]; ok {
				dumper(path)
			}
		}
	}
	//wait utill all the replace complete
	r.wg.Wait()
	//close the channel
	close(r.ch)
}

func withPaths(paths ...string) cfunc {
	return func(replacer *Replacer) {
		replacer.paths = append(replacer.paths, paths...)
	}
}
func withExtents(exts ...string) cfunc {
	return func(replacer *Replacer) {
		for _, ext := range exts {
			if _, ok := replacer.extents[ext]; !ok {
				replacer.extents[ext] = struct{}{}
			}
		}
	}
}

func withContent(content string) cfunc {
	return func(replace *Replacer) {
		replace.content = content
	}
}

func withRep(rep string) cfunc {
	return func(replace *Replacer) {
		replace.replace = rep
	}
}

func withRec(rec bool) cfunc {
	return func(replace *Replacer) {
		replace.recursive = rec
	}
}
func extracter(wg *sync.WaitGroup, ch <-chan string, content, rep string) {
	for path := range ch {
		go replace(wg, path, content, rep)
	}
}

func replace(wg *sync.WaitGroup, src, content, replace string) {
	defer wg.Done()

	var once sync.Once
	f, err := os.Open(src)
	defer f.Close()
	if err != nil {
		panic(fmt.Sprint("Open %s error: %v", err))
	}

	var bs bytes.Buffer
	if err != nil {
		panic(fmt.Sprintf("Create template file error %v", err))
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, content) {
			once.Do(func() {
				atomic.AddInt32(&count, 1)
			})
			fmt.Printf("Replacing %s...\n", src)
			line = strings.Replace(line, content, replace, -1)
		}
		bs.WriteString(line + "\n")
	}
	if err := scanner.Err(); err != nil {
		panic(fmt.Sprintf("Scan file %s error: %v", f.Name(), err))
	}

	//remove the source file
	err = os.Remove(src)
	if err != nil {
		panic(fmt.Sprintf("Remove %s error: %v", src, err))
	}

	//copy template file
	tf, err := os.Create(src)
	if err != nil {
		panic(fmt.Sprintf("Create %s error: %v", src, err))
	}
	defer tf.Close()
	_, err = io.Copy(tf, &bs)
	if err != nil {
		panic(fmt.Sprintf("Copy %s error: %v", src, err))
	}
}
