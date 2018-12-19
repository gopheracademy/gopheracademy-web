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
personal photo collections. Go itself is a great example for the power of simplicity.

This article explains our choice of technology and highlights interesting challenges we are determined to solve.
More information and a [demo](https://demo.photoprism.org/) can be found on [photoprism.org](https://photoprism.org/).

![](https://dl.photoprism.org/assets/img/usa.jpg)

### Broad Adoption Requires a Single Binary ###

Go was the natural choice for our endeavour: It is available for all major operating systems,
comes with a built-in Web server, is easy to learn and open-source, has an amazing community,
plus there is an API for [Google TensorFlow](https://www.tensorflow.org/).

User [feedback](https://github.com/photoprism/photoprism/wiki/Concerns) we received while PhotoPrism was [trending on Reddit](https://www.reddit.com/r/golang/comments/9nwjpf/photoprism_personal_photo_management_powered_by/)
also made clear that we have to provide a single binary including all dependencies to reach broad adoption.
Other than developers, most users are not comfortable using Docker.

### Go-native Replacement for MySQL ###

Finding a native replacement for MySQL was one of the challenges we had to solve for this.
The two obvious alternatives were using a key/value store like [LevelDB](https://github.com/google/leveldb)
or going for [SQLite3](https://github.com/mattn/go-sqlite3) - a popular
embedded SQL database. It requires linking to a C library, the Go driver is just a wrapper.

A key/value store would have added major overhead as we wouldn't be able to build upon everything
SQL has to provide plus PhotoPrism wouldn't work with external SQL databases anymore.
SQLite3 might work well in this regard, the differences to MySQL are minimal. We would still
have to find a way to properly manage concurrency, for example when indexing photos in goroutines.
Plus building gets ugly because of the C library.

In a commercial project, this might have been the end of the story, but we saw this as a unique
opportunity to experiment: [TiDB](https://github.com/pingcap/tidb) is a New SQL database implemented in pure Go.
Why not embed TiDB and run our own MySQL-compatible database server? Problem [solved](https://github.com/photoprism/photoprism/issues/60#issuecomment-448470212), at least for now.

### Image Classification using TensorFlow ###

The [TensorFlow API for Go](https://www.tensorflow.org/install/lang_go) is well suited to loading [existing models](https://github.com/tensorflow/models/blob/master/research/slim/README.md) and executing them within a Go application.
It is not designed to train models - you'll have to learn Python for this and it obviously requires a very large labeled set of images.

Getting a list of tags for an image is [pretty simple](https://outcrawl.com/image-recognition-api-go-tensorflow) and requires less than 200 lines of code.
All you need to do is load one of the free models (thank you Google!), resize the image to whatever the model uses as input,
run inference and then filter the best labels by probability:

```go
// GetImageTags returns the tags for a given image.
func (t *TensorFlow) GetImageTags(image string) (result []TensorFlowLabel, err error) {
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
		return nil, errors.New("could not run inference")
	}

	// Return best labels
	return t.findBestLabels(output[0].Value().([][]float32)[0]), nil
}
```

The true challenges come after: To build a single binary, you need a statically linkable
version of the [TensorFlow C library](https://www.tensorflow.org/install/lang_c) - which is [not available](https://github.com/tensorflow/tensorflow/issues/15563) yet,
unless you compile it yourself. Plus a good user experience requires [natural language processing](https://github.com/photoprism/photoprism/wiki/Image-Classification#natural-language-processing) to match search terms with similar tags/labels.

If this sounds like something you enjoy, you're welcome to [join our team](https://docs.photoprism.org/en/latest/contribute/)!

### Face Recognition ###

Using dlib.

### Expectation-oriented Photo Selection ###

Andrea Ceroni.

### Outlook ###

PhotoPrism will be released when it's done in good quality. You can expect a release in the first half of 2019.

Our long-term goal is to become an open platform for machine learning research based on real-world photo collections.
