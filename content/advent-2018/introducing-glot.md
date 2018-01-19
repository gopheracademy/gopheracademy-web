+++
author = ["Arafat Dad Khan"]
title = "Introducing Glot the plotting library for Golang"
linktitle = "Introducing Glot the plotting library for Golang"
date = "2018-1-19T00:00:00"
series = ["Advent 2018"]
+++

![System diagram](/static/postimages/advent-2018/introducing-glot/glot-1.jpeg)

Go is an open source programming language that makes it easy to build simple, reliable, and efficient software. It provides an expressive syntax with its lightweight type system and comes with concurrency as a built-in feature at the language level. With all these features its not a surprise that Golang is really hot these days and tons of developers are shifting towards it.

While playing with Golang Packages. I was surprised to find out that it needs a simple plotting library for scientific computation purposes and so I decided to hack it together with Gnuplot and built a rather simple yet powerful plotting library that can easily be used by any average Joe.

Let’s start with something simple shall we?

Let’s look at a simple A 2-d plot that draws over a plane to mark points.


```go
package main
import "github.com/Arafatk/glot"
func main() {
	dimensions := 2
	// The dimensions supported by the plot
	persist := false
	debug := false
	plot, _ := glot.NewPlot(dimensions, persist, debug)
	pointGroupName := "Simple Circles"
	style := "circle"
	points := [][]float64{{7, 3, 13, 5.6, 11.1}, {12, 13, 11, 1,  7}}
        // Adding a point group
	plot.AddPointGroup(pointGroupName, style, points)
	// A plot type used to make points/ curves and customize and save them as an image.
	plot.SetTitle("Example Plot")
	// Optional: Setting the title of the plot
	plot.SetXLabel("X-Axis")
	plot.SetYLabel("Y-Axis")
	// Optional: Setting label for X and Y axis
	plot.SetXrange(-2, 18)
	plot.SetYrange(-2, 18)
	// Optional: Setting axis ranges
	plot.SavePlot("2.png")
}
```
The commented code above is self-explanatory and plain. Notice how, the many customization options available make it easier to work with your plots.

![System diagram](/static/postimages/advent-2018/introducing-glot/glot-2.png)

That’s just an intro. The real takeaway is that the plot type is very dynamic and supports easy adding and removing of different types of point groups to the same plot. So now I am gonna add a simple line curve to this plot.

```go
package main

import "github.com/Arafatk/glot"

func main() {
	dimensions := 2
	// The dimensions supported by the plot
	persist := false
	debug := false
	plot, _ := glot.NewPlot(dimensions, persist, debug)
	pointGroupName := "Simple Circles"
	style := "circle"
	points := [][]float64{{7, 3, 13, 5.6, 11.1}, {12, 13, 11, 1,  7}}
        // Adding a point group
	plot.AddPointGroup(pointGroupName, style, points)
	pointGroupName = "Simple Lines"
	style = "lines"
	points = [][]float64{{7, 3, 3, 5.6, 5.6, 7, 7, 9, 13, 13, 9, 9}, {10, 10, 4, 4, 5.4, 5.4, 4, 4, 4, 10, 10, 4}}
	plot.AddPointGroup(pointGroupName, style, points)
	// A plot type used to make points/ curves and customize and save them as an image.
	plot.SetTitle("Example Plot")
	// Optional: Setting the title of the plot
	plot.SetXLabel("X-Axis")
	plot.SetYLabel("Y-Axis")
	// Optional: Setting label for X and Y axis
	plot.SetXrange(-2, 18)
	plot.SetYrange(-2, 18)
	// Optional: Setting axis ranges
	plot.SavePlot("2.png")
}
```
Just by adding 4 lines to the previous code, I have added another line curve in this plot.

```go
pointGroupName = “Simple Lines”
style = “lines”
points = [][]float64{{7, 3, 3, 5.6, 5.6, 7, 7, 9, 13, 13, 9, 9}, {10, 10, 4, 4, 5.4, 5.4, 4, 4, 4, 10, 10, 4}} plot.AddPointGroup(pointGroupName, style, points)

```

![System diagram](/static/postimages/advent-2018/introducing-glot/glot-3.png)

*See what I did there ^^.*

You can also easily remove curves too and save different variants of the same plot with different styles. Currently Glot supports many styles like lines, points, linepoints, impulses, dots, bar, steps, histogram, circle, errorbars, boxerrorbars and I plan on adding more.

## Is that all?

No Way… The package supports all of 1,2 and 3 dimensional curves. And even supports functions of the form

Y = Function(X) or Z = Function(X,Y)

Lets take a look

![System diagram](/static/postimages/advent-2018/introducing-glot/glot-4.png)

Glot also supports 3-d plots

![System diagram](/static/postimages/advent-2018/introducing-glot/glot-5.png)
For more information checkout my [medium blog](https://medium.com/@Arafat./).

Acknowledgements

Thanks to my friends who helped with the drafts. I am especially thankful to [Sebastian Binet](https://github.com/sbinet) for his contribution to Go plotting libraries and other gonum libraries.
Feature requests

My ultimate goal is to make [Glot](https://github.com/Arafatk/glot) similar to a [matplotlib](https://matplotlib.org/) equivalent for Golang with tons of really amazing customisation features. I hope you find this interesting and useful. Feel free to try glot from [github](https://github.com/Arafatk/glot). Any suggestions and recommendations are welcome.

Have a great day!!!
