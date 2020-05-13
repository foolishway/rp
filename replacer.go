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

func newReplacer(cfuncs ...cfunc) *Replacer {
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
						if _, ok := r.extents[filepath.Ext(p)]; ok {
							absPath, err := filepath.Abs(p)

							checkErr(err)
							dumper(absPath)
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
					if !f.IsDir() {
						if _, ok := r.extents[filepath.Ext(f.Name())]; ok {
							absPath, err := filepath.Abs(filepath.Join(path, f.Name()))
							checkErr(err)
							dumper(absPath)
						}
					}
				}
			}
		} else {
			if _, ok := r.extents[path]; ok {
				absPath, err := filepath.Abs(path)
				checkErr(err)
				dumper(absPath)
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
		var exist bool
		for i := 0; i < len(paths); i++ {
			for j := 0; j < len(replacer.paths); j++ {
				if paths[i] == replacer.paths[j] {
					exist = true
					break
				}
			}
			if !exist {
				replacer.paths = append(replacer.paths, paths[i])
			}
		}
		if len(replacer.paths) == 0 {
			replacer.paths = []string{"./"}
		}
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
	checkErr(err)

	var bs bytes.Buffer
	checkErr(err)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, content) {
			once.Do(func() {
				atomic.AddInt32(&count, 1)
				fmt.Printf("Replacing %s...\n", src)
			})
			line = strings.Replace(line, content, replace, -1)
		}
		bs.WriteString(line + "\n")
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	f.Close()

	//remove the source file
	err = os.Remove(src)
	checkErr(err)

	//copy template file
	tf, err := os.Create(src)
	checkErr(err)

	defer tf.Close()
	_, err = io.Copy(tf, &bs)
	checkErr(err)
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
