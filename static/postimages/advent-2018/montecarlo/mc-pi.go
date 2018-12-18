package main

import (
	"flag"
	"fmt"
	"math/rand"
)

func main() {
	n := flag.Int("n", 1e7, "MC sample size")
	flag.Parse()
	fmt.Printf("pi(%d) = %1.16f\n", *n, pi(*n))
}

func pi(n int) float64 {
	inside := 0
	for i := 0; i < n; i++ {
		x := rand.Float64()
		y := rand.Float64()
		if (x*x + y*y) < 1 {
			inside++
		}
	}
	return 4 * float64(inside) / float64(n)
}
