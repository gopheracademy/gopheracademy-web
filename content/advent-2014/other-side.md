+++
author = ["Anthony Starks"]
date = "2014-12-25T17:10:00+05:00"
title = "The Other Side of Go: Programming Pictures, the Read, Parse, Draw Pattern"
series = ["Advent 2014"]
+++

# The other side of Go: Programming Pictures, the Read, Parse, Draw Pattern

Go has proven to be extremely versatile and well suited to back-end
tasks, but sometimes you need a picture, and I've found that Go works
well for generating visuals as well. This post will explore one method
for generating pictures (specifically vector graphics) from data using
the [SVGo package](https://github.com/ajstarks/svgo).

The [SVGo package API](http://godoc.org/github.com/ajstarks/svgo)
performs a single function: generate standard
[SVG](http://www.w3.org/TR/SVG11/) to an `io.Writer.` Because of Go's
flexible I/O package, your pictures can go anywhere you need to write:
standard output, files, network connections, and in web servers.

SVGo is designed so that programmers can think in terms of high-level
objects like circles, rectangles, lines, polygons, and curves, using the
program's logic to manage layout and relationships, while applying
styles and other attributes to the objects as needed.

![SVGo API](https://farm9.staticflickr.com/8613/16056841885_9f13689cf6_b.jpg "SVGo API")

## The read/parse/draw pattern

One pattern for generating pictures from your own or Internet sources is
the read/parse/draw pattern. The pattern has these steps:

* Define the input data structures and destination
* Read the input
* Parse and load the data structures
* Draw the picture, walking through the structures

[Here is a simple example](https://github.com/ajstarks/svgo/blob/master/rpd/rpd.go)
that takes data from XML (using JSON is very
similar), and creates a simple visualization in SVG to standard output.
Note that for your own data you are free to define the input structure
as you see fit, but other sources like Internet service APIs will define
their own structure.

Given the XML input (thing.xml).

```xml
	<thing top="100" left="100" sep="100">
    	<item width="50"  height="50"  name="Little" color="blue">This is small</item>
    	<item width="75"  height="100" name="Med"    color="green">This is medium</item>
    	<item width="100" height="200" name="Big"    color="red">This is large</item>
	</thing>
```
	
First we define the data structures that match the structure of our
input.  You can see the correspondence between the elements and
attributes of the data with the Go struct: A "thing" has a top and left
location that defines the drawing's origin, along with an attribute that
defines the separation between elements. Within the thing is a list of
items, each one having a width, height, name, color, and text.

```go
	type Thing struct {
		Top  int `xml:"top,attr"`
		Left int `xml:"left,attr"`
		Sep  int `xml:"sep,attr"`
		Item []item `xml:"item"`
	}
	
	type item struct {
		Width  int    `xml:"width,attr"`
		Height int    `xml:"height,attr"`
		Name   string `xml:"name,attr"`
		Color  string `xml:"color,attr"`
		Text   string `xml:",chardata"`
	}
```

Specify the destination for the generated SVG, standard output, and
flags for specifying the dimensions of the canvas.
	
```go
	var (
		canvas = svg.New(os.Stdout)
		width = flag.Int("w", 1024, "width") 
		height = flag.Int("h", 768, "height") 
	)
```

Next, define a function for reading the input:

```go
	func dothing(location string) {
		f, err := os.Open(location)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}
		defer f.Close()
		readthing(f)
	}
```

An important function is to parse the and load the structs ---this is
straightforward using the [XML package](http://golang.org/pkg/encoding/xml/) from Go's standard
library: Pass the `io.Reader` to `NewDecoder`, and `Decode` into the
thing.

```go
	func readthing(r io.Reader) {
		var t Thing
		if err := xml.NewDecoder(r).Decode(&t); err != nil {
			fmt.Fprintf(os.Stderr, "Unable to parse components (%v)\n", err)
			return
		}
		drawthing(t)
	}
```

Finally, once you have the data loaded, walk the data, making the
picture. This is where you use the higher-level functions of the SVGo
library to make your visualization. In this case, set the origin (x, y),
and for each item, make a circle that corresponds to the specified size
and color. Next add the text, with the desired attributes.  Finally,
apply vertical spacing between each item.

```go
	func drawthing(t Thing) {
		x := t.Left
		y := t.Top
		for _, v := range t.Item {
			style := fmt.Sprintf("font-size:%dpx;fill:%s", v.Width/2, v.Color)
			canvas.Circle(x, y, v.Height/4, "fill:"+v.Color)
			canvas.Text(x+t.Sep, y, v.Name+":"+v.Text+"/"+v.Color, style)
			y += v.Height
		} 
	}
```

The main program kicks things off, reading the input file from the command line:

```go
	func main() {
		flag.Parse()
		for _, f := range flag.Args() {
			canvas.Start(*width, *height)
			dothing(f)
			canvas.End()
		}
	}
```

running the program sends the SVG to standard output

	$ go run rpd.go thing.xml
	<?xml version="1.0"?>
	<!-- Generated by SVGo -->
	<svg width="1024" height="768"
		 xmlns="http://www.w3.org/2000/svg" 
		 xmlns:xlink="http://www.w3.org/1999/xlink">
	<circle cx="100" cy="100" r="12" style="fill:blue"/>
	<text x="200" y="100" style="font-size:25px;fill:blue">Little:This is small/blue</text>
	<circle cx="100" cy="150" r="25" style="fill:green"/>
	<text x="200" y="150" style="font-size:37px;fill:green">Med:This is medium/green</text>
	<circle cx="100" cy="250" r="50" style="fill:red"/>
	<text x="200" y="250" style="font-size:50px;fill:red">Big:This is large/red</text>
	</svg>

When viewed in a browser, it looks like this:

![SVGo Thing](https://farm8.staticflickr.com/7548/16031051506_81f407ba05_b.jpg "Thing Output")

Using this pattern, you can build many kinds of visualization tools; for
example in my work I have tools that build conventional things like
[barcharts](https://github.com/ajstarks/svgo/tree/master/barchart) and
[bulletgraphs](https://github.com/ajstarks/svgo/tree/master/bulletgraph)
, but also [alternatives to
pie-charts](https://github.com/ajstarks/svgo/tree/master/pmap),
roadmaps, [component
diagrams](https://github.com/ajstarks/svgo/tree/master/compx),
timelines, heatmaps, and scoring grids.

You can also generate data from Internet APIs as well.  For example, the
["f50" (Flickr50)](https://github.com/ajstarks/svgo/tree/master/f50)
program takes a keyword and generates a clickable grid of pictures
chosen by Flickr's "interestingness" algorithm. f50 uses the same
pattern as above, but instead of reading from files, it makes a HTTPS
request, parses the XML response, and makes the picture.

	$ f50 sunset
	
Generates this response:

```xml
	<?xml version="1.0" encoding="utf-8" ?> 
	<rsp stat="ok">
		<photo id="15871035007" ... secret="84d59df678" server="7546" farm="8" title="flickr-gopher" ... />
		<photo id="15433662714" ... secret="3b9358c61d" server="7559" farm="8" title="Laurence Maroney 2006..." ... />
		...
	</rsp>
```
	
The f50 program uses the id, secret, farm, server and title attributes to build this picture.

```go
	// makeURI converts the elements of a photo into a Flickr photo URI
	func makeURI(p Photo, imsize string) string {
		im := p.Id + "_" + p.Secret

		if len(imsize) > 0 {
			im += "_" + imsize
		}
		return fmt.Sprintf(urifmt, p.Farm, p.Server, im)
	}

	// imageGrid reads the response from Flickr, and creates a grid of images
	func imageGrid(f FlickrResp, x, y, cols, gutter int, imgsize string) {
		if f.Stat != "ok" {
			fmt.Fprintf(os.Stderr, "Status: %v\n", f.Stat)
			return
		}
		xpos := x
		for i, p := range f.Photos.Photo {
			if i%cols == 0 && i > 0 {
				xpos = x
				y += (imageHeight + gutter)
			}
			canvas.Link(makeURI(p, ""), p.Title)
			canvas.Image(xpos, y, imageWidth, imageHeight, makeURI(p, "s"))
			canvas.LinkEnd()
			xpos += (imageWidth + gutter)
		}
	}
```
	
![Flickr 50 output](https://farm8.staticflickr.com/7546/15871035007_84d59df678_z.jpg "Flickr 50: gopher")

If you view the resulting SVG in a browser, and hover over a picture, you can see the title, and click on it to get the larger image.





