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

	huni := hbook.NewH1D(100, 0, 1.0)
	hgauss := hbook.NewH1D(100, -5, 5)

	for i := 0; i < N; i++ {
		r := rnd.Float64() // r is in [0.0, 1.0)
		huni.Fill(r, 1)

		g := rnd.NormFloat64()
		hgauss.Fill(g, 1)
	}

	plot(huni, "uniform.png")
	plot(hgauss, "norm.png")
}

func plot(h *hbook.H1D, fname string) {
	p := hplot.New()
	hh := hplot.NewH1D(h)
	hh.Color = color.NRGBA{0, 0, 255, 255}
	p.Add(hh, hplot.NewGrid())

	const (
		width  = 10 * vg.Centimeter
		height = -1 // automatically chose height
	)
	err := p.Save(width, height, fname)
	if err != nil {
		log.Fatal(err)
	}
}
