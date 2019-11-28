// Convert time from one time zone to another
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

const layout = "2006-01-02T15:04"
const usage = `usage: %s TIME FROM_TZ TO_TZ

Convert time from one time zone to the other.
TIME format is YYYY-MM-DDTHH:MM
`

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage, os.Args[0])
	}
	flag.Parse()
	if flag.NArg() != 3 {
		log.Fatal("error: wrong number of arguments")
	}

	ts, from, to := flag.Arg(0), flag.Arg(1), flag.Arg(2)
	t, err := convertTime(ts, from, to)
	if err != nil {
		log.Fatalf("error: %s", err)
	}

	fmt.Println(t.Format(layout))
}

func convertTime(ts, from, to string) (time.Time, error) {
	src, err := time.LoadLocation(from)
	if err != nil {
		return time.Time{}, err
	}

	t, err := time.ParseInLocation(layout, ts, src)
	if err != nil {
		return time.Time{}, err
	}

	dest, err := time.LoadLocation(to)
	if err != nil {
		return time.Time{}, err
	}

	return t.In(dest), nil
}
