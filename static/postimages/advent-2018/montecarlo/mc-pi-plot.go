package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"net/http"

	"go-hep.org/x/hep/hplot"
	"golang.org/x/net/websocket"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

func main() {

	log.SetPrefix("monte-carlo: ")
	log.SetFlags(0)

	n := flag.Int("n", 1e7, "number of samples")
	flag.Parse()

	srv := newServer()
	go srv.pi(*n)

	http.HandleFunc("/", plotHandle)
	http.Handle("/data", websocket.Handler(srv.dataHandler))

	log.Printf("listening on :8080...")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func (srv *server) pi(samples int) {
	for i := 0; i < samples; i++ {
		x := rand.Float64()
		y := rand.Float64()
		srv.datac <- [2]float64{x, y}
	}
}

func plotHandle(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, page)
}

func (srv *server) dataHandler(ws *websocket.Conn) {
	for data := range srv.plots {
		err := websocket.JSON.Send(ws, data)
		if err != nil {
			log.Printf("error sending data: %v\n", err)
		}
	}
}

type server struct {
	in  plotter.XYs // points inside the circle
	out plotter.XYs // points outside the circle
	n   int         // number of samples

	datac chan [2]float64 // channel of (x,y) points randomly drawn
	plots chan wplot      // channel of base64-encoded PNG plots
}

func newServer() *server {
	srv := &server{
		in:    make(plotter.XYs, 0, 1024),
		out:   make(plotter.XYs, 0, 1024),
		datac: make(chan [2]float64),
		plots: make(chan wplot),
	}

	go srv.run()

	return srv
}

func (srv *server) run() {
	for v := range srv.datac {
		srv.n++
		x := v[0]
		y := v[1]
		d2 := x*x + y*y
		pt := Point{x, y}
		switch {
		case d2 < 1:
			srv.in = append(srv.in, pt)
		default:
			srv.out = append(srv.out, pt)
		}
		switch {
		case srv.n < 1e1:
			if srv.n%1e0 == 0 {
				srv.plots <- plot(srv.n, srv.in, srv.out)
			}
		case srv.n < 1e2:
			if srv.n%1e1 == 0 {
				srv.plots <- plot(srv.n, srv.in, srv.out)
			}
		case srv.n < 1e3:
			if srv.n%1e2 == 0 {
				srv.plots <- plot(srv.n, srv.in, srv.out)
			}
		case srv.n < 1e4:
			if srv.n%1e3 == 0 {
				srv.plots <- plot(srv.n, srv.in, srv.out)
			}
		case srv.n < 1e5:
			if srv.n%1e4 == 0 {
				srv.plots <- plot(srv.n, srv.in, srv.out)
			}
		case srv.n < 1e6:
			if srv.n%1e5 == 0 {
				srv.plots <- plot(srv.n, srv.in, srv.out)
			}
		case srv.n < 1e7:
			if srv.n%1e6 == 0 {
				srv.plots <- plot(srv.n, srv.in, srv.out)
			}
		case srv.n > 1e7:
			if srv.n%1e7 == 0 {
				srv.plots <- plot(srv.n, srv.in, srv.out)
			}
		}
	}
}

func plot(n int, in, out plotter.XYs) wplot {
	radius := vg.Points(0.1)

	p := hplot.New()

	p.X.Label.Text = "x"
	p.X.Min = 0
	p.X.Max = 1
	p.Y.Label.Text = "y"
	p.Y.Min = 0
	p.Y.Max = 1

	pi := 4 * float64(len(in)) / float64(n)
	p.Title.Text = fmt.Sprintf("n = %d\nπ = %v", n, pi)

	sin, err := hplot.NewScatter(in)
	if err != nil {
		log.Fatal(err)
	}
	sin.Color = color.RGBA{255, 0, 0, 255} // red
	sin.Radius = radius

	sout, err := hplot.NewScatter(out)
	if err != nil {
		log.Fatal(err)
	}
	sout.Color = color.RGBA{0, 0, 255, 255} // blue
	sout.Radius = radius

	p.Add(sin, sout, hplot.NewGrid())

	return wplot{Plot: renderImg(p)}
}

func renderImg(p *hplot.Plot) string {
	size := 20 * vg.Centimeter
	canvas := vgimg.PngCanvas{vgimg.New(size, size)}
	p.Draw(draw.New(canvas))
	out := new(bytes.Buffer)
	_, err := canvas.WriteTo(out)
	if err != nil {
		log.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(out.Bytes())
}

type wplot struct {
	Plot string `json:"plot"`
}

type Point struct {
	X, Y float64
}

const page = `
<html>
	<head>
		<title>Monte Carlo</title>
		<script type="text/javascript">
		var sock = null;
		var plot = "";

		function update() {
			var p = document.getElementById("plot");
			p.src = "data:image/png;base64,"+plot;
		};

		window.onload = function() {
			sock = new WebSocket("ws://"+location.host+"/data");

			sock.onmessage = function(event) {
				var data = JSON.parse(event.data);
				plot = data.plot;
				update();
			};
		};

		</script>
	</head>

	<body>
		<div id="content">
			<p style="text-align:center;">
				<img id="plot" src="" alt="Not Available"></img>
			</p>
		</div>
	</body>
</html>
`
