+++
author = ["Simon HEGE"]
date = "2015-12-23T08:00:00+00:00"
title = "Geographical data manipulation using go"
series = ["Advent 2015"]

+++

GIS open source world is dominated by C/C++, Java and Python code. 
Libraries like PROJ4, JTS, GEOS or GDAL are at the core of most of the 
open source geospatial projects. Through this article we will have a 
look at the ecosystem of geospatial related packages. We will create a 
GIF generator of an animated earth. In case you want to know more about 
the image generation package, I recommend reading two articles on the 
Go blog: [The Go image package](
http://blog.golang.org/go-image-package) and [the Thanksgiving 2011 
doodle](http://blog.golang.org/from-zero-to-go-launching-on-google). 

![Rotating Earth](/postimages/advent-2015/earth.gif)

The basic concepts to generate an image like this is to use a dataset
containing the world countries (as polygons or multipolygons) and to 
use an orthographic projection. By varying the longitude, we will mimic 
the rotation of the Earth. The projected coordinates will then be 
scaled to fit the required GIF size.

# Parsing input data
Multiple libraries are available to parse geographical data. If you 
have the possibility to use CGO, the easiest way is to use a wrapper 
around [GDAL/OGR](https://godoc.org/?q=gdal). They are mainly based on 
the [github.com/lukeroth/gdal](https://github.com/lukeroth/gdal) 
package. It requires to have the GDAL shared library available at 
runtime.

Otherwise you may use pure Go libraries which are available for some 
common formats:

- Shapefile: https://github.com/jonas-p/go-shp and its forks
- [GeoJSON](https://godoc.org/?q=geojson)

A GeoJSON parsing package is as simple as defining the 3 struct as  
bellow and using the ``encoding\json`` standard package.

```go
type FeatureCollection struct {
	Type     string     `json:"type"`
	Features []*Feature `json:"features"`
}

type Feature struct {
	Type       string                 `json:"type"`
	Id         string                 `json:"id"`
	Geometry   Geometry               `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
}

type Geometry struct {
	Type        string      `json:"type"`
	Coordinates interface{} `json:"coordinates,omitempty"`
	Geometries  []*Geometry `json:"geometries,omitempty"`
}
```

On the other side, no pure Go GML decoding libraries are available. The
format is heavily based on inheritance and substitutions concepts. Thus 
it is an open challenge for everyone who wants to translate it to the 
Go philosophy.

	
# Reprojecting data
The easiest way to manage reprojection of data is by using the PROJ4 C
library. It can be done either using a shared library 
([github.com/pebbe/go-proj-4/proj](https://github.com/pebbe/go-proj-4)) 
or by including the C code of the PROJ4 library inside a Go package
([github.com/xeonx/proj4](https://github.com/xeonx/proj4)).

The C code is made to operate on an array of double. Using the 
''unsafe'' package, it is possible to get pointers to members of Go 
structs and size of Go structs, making possible to operate directly on 
a Go slice of struct having X and Y members.

```go
type Point struct {
	float64 X
	float64 Y
}

var pointSize = (C.int)(unsafe.Sizeof(Point{}) / unsafe.Sizeof(float64(0.0)))

var points []Point

...

errno := C.pj_transform(src, dst,
	C.long(len(points)),
	pointSize,
	(*C.double)(unsafe.Pointer(&points[0].X)),
	(*C.double)(unsafe.Pointer(&points[0].Y)),
	nil)
```

# Creating images
[github.com/llgcode/draw2d](https://github.com/llgcode/draw2d) is a 
pure go 2D vector graphics library. After choosing a backend (image, 
pdf, ...) you can setup a transformation matrix and start drawing in your
own coordinates system. In our case we want to have the center of our 
coordinates system at the center of the image and a scale such that the 
earth will use as much space as possible.
```go

imgSize := 256.
earthRadius := 6378137.

r := image.Rect(0,0,int(imgSize),int(imgSize))

img := image.NewRGBA(r)
gc := draw2dimg.NewGraphicContext(img)

gc.Translate(float64(r.Dx())/2.,float64(r.Dy())/2.)
gc.Scale(imgSize/(2.*earthRadius),-imgSize/(2.*earthRadius))

```

# Conclusion

A few geospatial projects (for example [imposm](http://imposm.org/)) have 
started transitionning from C/C++ or Python to Go. The ecosystem is still 
young but, thanks to cgo, interoperability with existing libraries is 
easy.

Some libraries are being developed aiming at pure Go manipulation of 
geographical data. I may cite:

- [github.com/paulmach/go.geo](https://github.com/paulmach/go.geo) focused
on server side manipulation. It includes algorithm such as clustering or 
GeoHash.
- [github.com/ctessum/geom](https://github.com/ctessum/geom): geometry
objects that can be encoded, decoded, reprojected, ...
- [github.com/twpayne/go-geom](https://github.com/twpayne/go-geom): fast 
and GC-efficient Open Geo Consortium-style geometries, including encoding 
and decoding from WKB.

Fo now there is no pure Go implementation of the OGC Simple Feature 
specification, and still less of the GML specification heavily based on 
inheritance concept. Go packages tends to solve specific problems faced 
by developpers and let the generic solution to other languages.

# Source code
The whole source code used to generate the GIF is available at 
https://github.com/xeonx/earth