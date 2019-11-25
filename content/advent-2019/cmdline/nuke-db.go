package main

import (
	"flag"
	"fmt"
	"os"
)

var usage = `usage: %s [DATABASE]

Delete all data and tables from DATABASE.
`

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	fmt.Println("database nuked")
}
