+++
author = ["Michael Mayer"]
title = "Personal Photo Management using Go and TensorFlow"
linktitle = "Personal Photo Management using Go and TensorFlow"
date = 2018-12-26
series = ["Advent 2018"]
+++

We love taking photos. Privacy concerns - and the wish to properly archive them for the next generation -
brought us to the conclusion that existing cloud solutions are not the right tool to keep them organized.
That's why we started working on an easy-to-use application that can be hosted at home or on a private server.

### About PhotoPrism.org ###

Our first proof-of-concept was a simple demo app that could [find cat pictures](https://github.com/photoprism/photoprism/wiki/Screenshots) in a directory.
The progress since then is remarkable.
While me might not reach that goal with our first release, we strive to build the most user-friendly software for browsing, organizing and sharing
personal photo collections.
Go itself is a great example for the power of [simplicity](https://talks.golang.org/2015/simplicity-is-complicated.slide).

This article explains our choice of technology and highlights interesting challenges we are solving.
More information and a [demo](https://demo.photoprism.org/) can be found on [photoprism.org](https://photoprism.org/).

![User Interface](/postimages/advent-2018/photoprism/preview.jpg)

### Broad Adoption Requires a Single Binary ###

Go was the natural choice for our endeavor: It is available for all major operating systems,
comes with a built-in Web server, is easy to learn and open-source, has an amazing community,
plus there is an API for [Google TensorFlow](https://www.tensorflow.org/).

User [feedback](https://github.com/photoprism/photoprism/wiki/Concerns) we received while PhotoPrism was [trending](https://www.reddit.com/r/selfhosted/comments/9op2kn/photoprism_new_selfhosted_free_software_photo/) on [Reddit](https://www.reddit.com/r/golang/comments/9nwjpf/photoprism_personal_photo_management_powered_by/)
also made clear that we have to provide a single binary including all dependencies to reach broad adoption.
Other than developers, most users are not comfortable using Docker.

### Go-native Replacement for MySQL ###

Finding a native replacement for MySQL was one of the challenges we had to solve for this.
The two obvious alternatives were using a key/value store like [LevelDB](https://github.com/google/leveldb)
or going for [SQLite3](https://github.com/mattn/go-sqlite3) - a popular
embedded SQL database. It requires linking to a C library, the Go driver is just a wrapper.

A key/value store would have added major overhead as we wouldn't be able to build upon everything
SQL has to provide plus our app wouldn't work with external SQL databases anymore.
SQLite3 might work well in this regard, the differences to MySQL are minimal. We would still
have to find a way to properly manage concurrency, for example when indexing photos in goroutines.
On top, building gets ugly because the C library prevents cross compiling, although
some people seem to [work on this](https://github.com/karalabe/xgo).

In a commercial project, this might have been the end of the story, but we saw this as a unique
opportunity to experiment: [TiDB](https://github.com/pingcap/tidb) is a New SQL database implemented in pure Go.
Why not embed it and run our own MySQL-compatible database server?

All we had to do is take TiDB's main function and [slightly modify it](https://github.com/photoprism/photoprism/blob/develop/internal/tidb/server.go)
to work with our configuration:

```go
go tidb.Start(storagePath, serverPort, serverHost, debug)
```

Problem solved and the TiDB developers even think [it's cool](https://github.com/photoprism/photoprism/issues/60#issuecomment-448470212).

### Image Classification Using TensorFlow ###

The [TensorFlow API for Go](https://www.tensorflow.org/install/lang_go) is well suited for loading [existing models](https://github.com/tensorflow/models/blob/master/research/slim/README.md)
and running them within a Go application.
It is not designed to train models - you'll have to learn Python for this and it obviously requires a large set of labeled images.

Getting a list of tags for an image is [pretty simple](https://outcrawl.com/image-recognition-api-go-tensorflow) and requires less than 200 lines of code.
All you need to do is load one of the free models (thank you Google!), resize the image to whatever the model uses as input,
run inference and then filter the best labels by probability:

```go
// GetImageTags returns tags for a jpeg image string.
func (t *TensorFlow) GetImageTags(image string) (
                                  result []TensorFlowLabel, err error) {
	if err := t.loadModel(); err != nil {
		return nil, err
	}

	// Make tensor
	tensor, err := t.makeTensorFromImage(image, "jpeg")
	if err != nil {
		return nil, errors.New("invalid image")
	}

	// Run inference
	session, err := tf.NewSession(t.graph, nil)
	if err != nil {
		log.Fatal(err)
	}

	defer session.Close()

	output, err := session.Run(
		map[tf.Output]*tf.Tensor{
			t.graph.Operation("input").Output(0): tensor,
		},
		[]tf.Output{
			t.graph.Operation("output").Output(0),
		},
		nil)

	if err != nil {
		return nil, err
	}

	// Return best labels
	return t.findBestLabels(output[0].Value().([][]float32)[0]), nil
}

// GetImageTagsFromFile returns tags for a jpeg file.
func (t *TensorFlow) GetImageTagsFromFile(filename string) (
                                  result []TensorFlowLabel, err error) {
	imageBuffer, err := ioutil.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	return t.GetImageTags(string(imageBuffer))
}
```

Application code can now easily find matching tags for an image:

```go
tf := NewTensorFlow(conf.TensorFlowModelPath())
tags, err := tf.GetImageTagsFromFile("IMG_6788.JPG")
```

The true challenges come after: To build a single binary, we need a statically linkable
version of the [TensorFlow C library](https://github.com/photoprism/photoprism/issues/83) for
every operating system we want to support. It is [not available](https://github.com/tensorflow/tensorflow/issues/15563) yet,
unless you compile it yourself. A good user experience also requires [natural language processing](https://github.com/photoprism/photoprism/wiki/Image-Classification#natural-language-processing) to match search terms with similar tags/labels.

If this sounds like something you enjoy, you're welcome to [join our team](https://docs.photoprism.org/en/latest/contribute/)!

We collected links to related articles and other useful information in our
[Developer Guide](https://github.com/photoprism/photoprism/wiki/Image-Classification).

### Face Recognition ###

You would expect that face recognition could be implemented similar to finding tags,
at least using the same framework. This is not the case.
While there are a number of interesting machine learning projects written in
pure Go, using [dlib](http://dlib.net/) - another external library - seems to be
the only way to go for proper face recognition. If you just want to detect faces
without identifying them, check out https://github.com/esimov/pigo.

We are looking for a contributor who likes to [implement a simple poof-of-concept](https://github.com/photoprism/photoprism/issues/22)
using [go-face](https://github.com/Kagami/go-face).
You can even earn $36 as this issue is [funded](https://github.com/photoprism/photoprism/issues?q=is%3Aissue+is%3Aopen+label%3AIssueHunt) by IssueHunt.
Think of open-source development as free training with a visible outcome. Of course we are there to help, if needed.

### Resampling JPEG Images ###

Thumbnails are essential for every photo app. We want the best possible
quality for all images, without compromise. This is a photo of a cat's whiskers
downsampled to 240x160 from 6000x4000, first Flickr then PhotoPrism
(Facebook's quality is so bad, you don't even need to compare it):

![Cat's whiskers downsampled by Flickr](/postimages/advent-2018/photoprism/flickr.png)

![Cat's whiskers downsampled by PhotoPrism](/postimages/advent-2018/photoprism/photoprism.png)

Notice the subtle, but significant differences. This fantastic quality
is possible thanks to the [disintegration/imaging](https://github.com/disintegration/imaging)
image processing package. The resampling filter we're using is Lanczos:

```go
import "github.com/disintegration/imaging"

img, err := imaging.Open("original.jpg", imaging.AutoOrientation(true))
if err != nil {
    return nil, err
}

img = imaging.Fit(img, 240, 160, imaging.Lanczos)
err = imaging.Save(img, "thumbnail.jpg")
```

While it may be slower than low quality filters, it still seems fast
enough to display tiles in search results on demand without pre-rendering.
Other people play 3D games at the highest possible resolution.

An [issue we discovered](https://github.com/photoprism/photoprism/issues/36) is
that Go [seems to be picky](https://github.com/golang/go/issues/10447) when it comes to proper encoding of JPEG files.
In practice, you'll find quite a lot of images that are slightly truncated
or have other, mostly invisible defects. Even the best SD card loses information over time.

### Working Towards Our First Release and Beyond ###

PhotoPrism will be released when it's done in good quality, hopefully in the first half of 2019.
We know we can do it because we've [done it before](https://github.com/photoprism/photoprism/wiki/Mediencenter).
Please [reach out to us](mailto:hello@photoprism.org) if you work for an organization that can support our project
as we are looking for a way to continue doing this full-time.

We've recently started organizing small [meetups in Berlin](https://github.com/photoprism/photoprism/wiki/Meetups).
You're welcome to join us, even if you're new to Go or software development in general.
We would like to establish regular learning sessions for beginners.

Our long-term goal is to become an open platform for machine learning research
based on real-world photo collections. Andrea Ceroni
recently joined [our team](https://docs.photoprism.org/en/latest/team/) as scientific adviser. He has
published [numerous papers](https://github.com/photoprism/photoprism/wiki/Research) related to
[Personal Photo Management and Preservation](https://dl.photoprism.org/slides/Personal%20Photo%20Management%20and%20Preservation.pdf).