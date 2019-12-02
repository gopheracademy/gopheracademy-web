package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
)

var (
	wordRe = regexp.MustCompile(`[a-zA-Z]+`)
)

func wordFreq(r io.Reader) (map[string]int, error) {
	counts := make(map[string]int)
	s := bufio.NewScanner(r)
	for s.Scan() {
		for _, w := range wordRe.FindAllString(s.Text(), -1) {
			counts[strings.ToLower(w)]++
		}
	}
	return counts, s.Err()
}

func keys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func topN(freqs map[string]int, n int) []string {
	words := keys(freqs)
	less := func(i, j int) bool {
		// Sort in reverse order
		return freqs[words[i]] > freqs[words[j]]
	}
	sort.Slice(words, less)
	return words[:n]
}

func main() {
	var count int

	flag.IntVar(&count, "count", 10, "number of top words to show")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s\nword frequency\n\noptions:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	freqs, err := wordFreq(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	for _, w := range topN(freqs, count) {
		fmt.Printf("%v\t%d\n", w, freqs[w])
	}
}
