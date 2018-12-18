package main

import (
	"fmt"
	"math/rand"
)

func main() {
	const seed = 12345
	src := rand.NewSource(seed)
	rnd := rand.New(src)

	const N = 10
	for i := 0; i < N; i++ {
		r := rnd.Float64() // r is in [0.0, 1.0)
		fmt.Printf("%v\n", r)
	}
}
