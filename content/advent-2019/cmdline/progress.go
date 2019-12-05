package main

import (
	"flag"
	"time"

	"github.com/cheggaaa/pb/v3"
)

func main() {
	flag.Parse()
	count := 100
	bar := pb.StartNew(count)
	for i := 0; i < count; i++ {
		time.Sleep(100 * time.Millisecond)
		bar.Increment()
	}
	bar.Finish()

}
