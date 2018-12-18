package main

import (
	"image/color"
	"log"
	"math/rand"

	"go-hep.org/x/hep/hbook"
	"go-hep.org/x/hep/hplot"
	"gonum.org/v1/plot/vg"
)

func main() {
	const seed = 12345
	src := rand.NewSource(seed)
	rnd := rand.New(src)

	const N = 10000

	// create a 1-dim histogram of float64s, with 100 bins, from 0 to 1.
	huni := hbook.NewH1D(100, 0, 1.0)

	for i := 0; i < N; i++ {
		r := rnd.Float64() // r is in [0.0, 1.0)
		huni.Fill(r, 1)
	}

	plot(huni, "uniform.png")
}

func plot(h *hbook.H1D, fname string) {
	p := hplot.New()      // create a new hplot.Plot
	hh := hplot.NewH1D(h) // create a plotter for the histogram
	hh.Color = color.NRGBA{0, 0, 255, 255}
	p.Add(hh, hplot.NewGrid())

	const (
		width  = 10 * vg.Centimeter
		height = -1 // choose height automatically
	)
	err := p.Save(width, height, fname)
	if err != nil {
		log.Fatal(err)
	}
}
