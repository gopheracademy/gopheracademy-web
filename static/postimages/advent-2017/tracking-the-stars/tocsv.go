package main

import (
	"bufio"
	"compress/gzip"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
)

type rawline struct {
	Ts    struct{ N string }
	Stars struct{ N string }
	Repo  struct{ S string }
}

func main() {
	id := flag.String("id", "", "Input directory (contains dynamodb dump files)")
	of := flag.String("of", "", "Output file name")
	nw := flag.Int("nw", runtime.NumCPU(), "Number of workers to run to process files")
	flag.Parse()

	if *id == "" || *of == "" {
		fmt.Println("Must include both id and of parameters")
		os.Exit(1)
	}

	datachan := make(chan rawline, 10000)
	writerdone := &sync.WaitGroup{}
	writerdone.Add(1)

	go func(filename string, datachan <-chan rawline, wg *sync.WaitGroup) {
		file, err := os.Create(filename)
		if err != nil {
			panic(err)
		}

		bufwriter := bufio.NewWriter(file)
		gzipwriter := gzip.NewWriter(bufwriter)
		out := csv.NewWriter(gzipwriter)

		for raw := range datachan {
			if err := out.Write([]string{raw.Repo.S, raw.Ts.N, raw.Stars.N}); err != nil {
				panic(err)
			}
		}

		// ordering is important here
		out.Flush()

		if err := gzipwriter.Close(); err != nil {
			panic(err)
		}
		if err := bufwriter.Flush(); err != nil {
			panic(err)
		}
		if err := file.Close(); err != nil {
			panic(err)
		}

		wg.Done()
	}(*of, datachan, writerdone)

	pathchan := make(chan string)
	readersdone := &sync.WaitGroup{}

	for i := 0; i < *nw; i++ {
		readersdone.Add(1)
		go func(pathchan <-chan string, datachan chan<- rawline, wg *sync.WaitGroup) {
			for path := range pathchan {

				fmt.Println("Processing file ", path)

				file, err := os.Open(path)
				if err != nil {
					panic(err)
				}

				dec := json.NewDecoder(bufio.NewReader(file))
				var raw rawline

				for {
					if err = dec.Decode(&raw); err == io.EOF {
						break
					} else if err != nil {
						panic(err)
					}

					// sanity check the integers
					if _, err = strconv.ParseUint(raw.Ts.N, 10, 64); err != nil {
						panic(err)
					}
					if _, err = strconv.ParseUint(raw.Stars.N, 10, 64); err != nil {
						panic(err)
					}

					datachan <- raw
				}
			}

			wg.Done()
		}(pathchan, datachan, readersdone)
	}

	err := filepath.Walk(*id, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			panic(err) // probably if there's no access
		}

		if info.IsDir() {
			return nil
		}

		pathchan <- path
		return nil
	})

	if err != nil {
		fmt.Println(err)
	}

	close(pathchan)
	readersdone.Wait()
	close(datachan)
	writerdone.Wait()
}
