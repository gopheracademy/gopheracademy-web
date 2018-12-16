+++
author = ["Andrew Brampton"]
date = "2018-12-17"
title = "Apache Beam and Google Dataflow in Go"
series = ["Advent 2018"]
+++

# Overview
[Apache Beam](https://beam.apache.org/) (**b**atch and str**eam**) is a powerful tool for handling [embarrassingly parallel](https://en.wikipedia.org/wiki/Embarrassingly_parallel) workloads. It is a evolution of [Google’s Flume](https://ai.google/research/pubs/pub35650), which provides batch and streaming data processing based on the [MapReduce](https://en.wikipedia.org/wiki/MapReduce) concepts. One of the novel features of Beam is that it’s agnostic to the platform that runs the code. For example, a pipeline can be written once, and run locally, across [Flink](https://flink.apache.org/) or [Spark](https://spark.apache.org/) clusters, or on [Google Cloud Dataflow](https://cloud.google.com/dataflow/).

An experimental [Go SDK](https://beam.apache.org/documentation/sdks/go/) was created for Beam, and while it is still immature compared to Beam for [Python](https://beam.apache.org/documentation/sdks/python/) and [Java](https://beam.apache.org/documentation/sdks/java/), it is able to do some impressive things. The remainder of this article will briefly recap a simple example from the Apache Beam site, and then work through a more complex example running on Dataflow. Consider this a more advanced version of the [official getted started guide](https://beam.apache.org/get-started/) on the Apache Beam site.

Before we begin, it’s worth pointing out, that if you can do your analysis on a single machine, it is more likely faster, and more cost effective. Beam is more suitable when your data processing needs are large enough they must run in a distributed fashion.

# Concepts
Beam already has good documentation, that explains all the [main concepts](https://beam.apache.org/documentation/programming-guide/). We will cover some of the basics.

<div style="text-align: center;">
	<img 
src="/postimages/advent-2018/apache-beam/design-your-pipeline-linear.png">
</div>

A pipeline is made up of multiple steps, that takes some input, operates on that data, and finally produces output. The steps that operates on the data are called PTransforms (parallel transforms), and the data is always stored in PCollections (parallel collections). The PTransform takes one item at a time from the PCollection and operates on it. The PTransform are assumed to be hermetic, using no global state, thus ensuring it will always produce the same output for the given input. These properties allow the data to be sharded into multiple smaller dataset and processed in any order. The code you write ends up being very simple, but when the program runs it is able to seamlessly split across 10s or 100s of machines.

# Shakespeare (simple example)

<div style="float: right; width: 200px">
	<img 
src="/postimages/advent-2018/apache-beam/word-count.png">
</div>

A classic example is counting the words in Shakespeare. In brief, the pipeline counts the number of times each word appears across Shakespeare’s works, and outputs a simple key-value list of word to word-count. There is an [example](https://github.com/apache/beam/blob/master/sdks/go/examples/minimal_wordcount/minimal_wordcount.go) provided with the Beam SDK, and along with a great [walk through](https://beam.apache.org/get-started/wordcount-example/). I suggest you read that before continuing. I will however dive into some of the Go specifics, and add additional context.

The example begins with [`textio.Read`](https://godoc.org/github.com/apache/beam/sdks/go/pkg/beam/io/textio#Read), which reads all the files under the shakespeare directory stored on [Google Cloud Storage](https://cloud.google.com/storage/) (GCS). The files are stored on GCS, so when this pipeline runs across a cluster of machines, they will all have access. [`textio.Read`](https://godoc.org/github.com/apache/beam/sdks/go/pkg/beam/io/textio#Read) always returns a `PCollection<string>` which contains one element for every line in the given files.

```go
lines := textio.Read(s, "gs://apache-beam-samples/shakespeare/*")
```

The `lines` PCollection is then processed by a ParDo (**Par**allel **Do**) a type of PTransform. Most transforms are built with a [`beam.ParDo`](https://godoc.org/github.com/apache/beam/sdks/go/pkg/beam#ParDo). It will execute a supplied function in parallel on the source PCollection. In this example, the function is defined inline and very simply splits the input lines into words with a regexp. Each word is then emitted to another `PCollection<string>` named `words`. Note how for every line, zero or more words may be emitted, making this new collection a different size to the original.

```go
splitFunc := func(line string, emit func(string)) {
    for _, word := range wordRE.FindAllString(line, -1) {
        emit(word)
    }
}
words := beam.ParDo(s, splitFunc, lines)
```

An interesting trick used by the Apache Beam Go API is passing functions as an `interface{}`, and using reflection to infer the types. Specifically, since `lines` is a `PCollection<string>` it is expected that the first argument of `splitFunc` is a string type. The second argument to `splitFunc` will allow Beam to infer the type of the `words` output PCollection. In this example it is a function with a single string argument. Thus the output type will be `PCollection<string>`. If `emit` was defined as `func(int)` then the return type would be a `PCollection<int>`, and the next PTransform would be expected to handle ints.

The next step uses one of the library’s higher level constructs.

```go
counted := stats.Count(s, words)
```

[`stats.Count`](https://godoc.org/github.com/apache/beam/sdks/go/pkg/beam/transforms/stats#Count) takes a `PCollection<X>`, counts each unique element, and outputs a key-value pair of (X, int) as a `PCollection<KV<X, int>>`. In this specific example, the input is a PCollection<string>, thus the output is PCollection<KV<string, int>>

Internally [`stats.Count`](https://godoc.org/github.com/apache/beam/sdks/go/pkg/beam/transforms/stats#Count) it’s made up of multiple ParDos, and a [`beam.GroupByKey`](https://godoc.org/github.com/apache/beam/sdks/go/pkg/beam#GroupByKey), but it hides that to make it easier to use.

At this point, the counts of each word has been calculated, and the results are stored to a simple text file. To do this the `PCollection<KV<string, int>>` is converted to a `PCollection<string>`, containing one element for each line to be written out.

```go
formatFunc := func(w string, c int) string {
    return fmt.Sprintf("%s: %v", w, c)
}
formatted := beam.ParDo(s, formatFunc, counted)
```

Again a [`beam.ParDo`](https://godoc.org/github.com/apache/beam/sdks/go/pkg/beam#ParDo) is used, but you’ll notice the `formatFunc` is slightly different to the `splitFunc` above. The `formatFunc` takes two arguments, a string (the key), and a int (the value). These are the pairs in the `PCollection<KV<string, int>>`. However, the `formatFunc` does not take a `emit func(...)` instead it simply returns a type string.

Since the PTransform outputs a single line for each input element, a simpler form of the function can be specified. One where the output element is just returned from the function. The `emit func(...)` is useful when the number of output elements differ to the number of input elements. If its a 1:1 mapping a return makes the function easier to read.

Multiple return arguments can also be used. For example, if the output was expected to be `PCollection<KV<float64, bool>>`, the return type could be `func(...) (float64, bool)`. This is all inferred at runtime with reflection when the pipeline is being constructed.

```go
textio.Write(s, "wordcounts.txt", formatted)
```

Finally the [`textio.Write`](https://godoc.org/github.com/apache/beam/sdks/go/pkg/beam/io/textio#Write) takes the `PCollection<string>` and writes it to a file named “wordcounts.txt" with one line per element.

## Running the pipeline

To test the pipeline it can easily be run locally like so:

```shell
go get github.com/apache/beam/sdks/go/examples/wordcount
cd $GOPATH/src/github.com/apache/beam/sdks/go/examples/wordcount
go run wordcount.go --runner=direct
```

To run in a more realistic way, it can be run on [GCP Dataflow](https://cloud.google.com/dataflow/). Before you do so, you need to create a GCP project, create a GCS bucket, enable the Cloud Dataflow APIs, and create a service account. This is documented on the [Python quickstart guide](https://cloud.google.com/dataflow/docs/quickstarts/quickstart-python), under “Before you begin”.

```shell
export GOOGLE_APPLICATION_CREDENTIALS=$PWD/your-gcp-project.json
export BUCKET=your-gcs-bucket
export PROJECT=your-gcp-project

cd $GOPATH/src/github.com/apache/beam/sdks/go/examples/wordcount
go run wordcount.go \
           --runner dataflow \
           --input gs://dataflow-samples/shakespeare/kinglear.txt \
           --output gs://${BUCKET?}/counts \
           --project ${PROJECT?} \
           --temp_location gs://${BUCKET?}/tmp/ \
           --staging_location gs://${BUCKET?}/binaries/ \
         --worker_harness_container_image=apache-docker-beam-snapshots-docker.bintray.io/beam/go:20180515
```

If this works correctly you’ll see something similar to the following printed:

```
Cross-compiling .../wordcount.go as .../worker-1-1544590905654809000
Staging worker binary:  .../worker-1-1544590905654809000
Submitted job: 2018-12-11_21_02_29
Console: https://console.cloud.google.com/dataflow/job/2018-12-11...
Logs: https://console.cloud.google.com/logs/viewer?job_id%2F2018-12-11...
Job state: JOB_STATE_PENDING …
Job still running …
Job still running …
...
Job succeeded!
```

Let's take a moment to explain what’s going on, starting with the various flags. The `--runner dataflow` flag tells the Apache Beam SDK to run this on GCP Dataflow, including executing all the steps required to make that happen. This includes, compiling the code and uploading it to the `--staging_location`. Later the staged binary will be run by Dataflow under the `--project` project. As this will be running “in the cloud”, the pipeline will not be able to access local files. Thus for both the `--input` and ` --output` flags, a bucket on GCS is used, as this is a convenient place to store files. Finally the `--worker_harness_container_image` flag specifies the docker image that Dataflow will use to host the workcount.go binary that was uploaded to the `--staging_location`.

Once wordcount.go is running, it prints out helpful information, such as links to the the Dataflow console. The console displays current progress as well as a visualization of the pipeline as a directed graph. Eventually the locally running wordcount.go will end when the pipeline successes or fails. Once this occurs, the logs link can provide useful information about what occurred. The local wordcount.go invocation can be interrupted at any time, but the pipeline will continue to run on Dataflow until it either succeeds or fails.

# Art history (more complex example)

<div style="float: right; width: 300px">
	<img src="/postimages/advent-2018/apache-beam/palette.png">
</div>

Now we’ll construct a more complex pipeline, that demonstrates some other features of Beam and Dataflow. In this pipeline we will be taking 100,000 paintings from the last 600 years and processing them to extract information about their color palettes. Specifically the question we aim to answer is, “Has the color palettes of paintings change over the decades?”. This may not be a pipeline we run repeatedly, but it was a fun example, and demonstrates many advance topics.

We will skip over the details of the color extraction algorithm, and provide that in a later article. Here we’ll focus on how to create a pipeline to accomplish this task.

We start by reading a csv file that contains metadata for each painting, such as the artist, year it was painted, and a GCS path to a jpg of the painting. The paintings will then be grouped by the decade they were painted, and then the color palette for each group will be determined. Each palette will saved to a png file, as well as all the palette saved to a single large json file. To finish it off, the pipeline will be productionised, so it easier to debug, and re-run. The full source code will be available here. TODO

To start with, the main function for the pipeline looks like this:

```go
import (
...
	"github.com/apache/beam/sdks/go/pkg/beam"
...
)

func main() {
	// If beamx or Go flags are used, flags must be parsed first.
	flag.Parse()

	// beam.Init() is an initialization hook that must called on startup. On
	// distributed runners, it is used to intercept control.
	beam.Init()

	p := beam.NewPipeline()
	s := p.Root()

	buildPipeline(s)

	ctx := context.Background()
	if err := beamx.Run(ctx, p); err != nil {
		log.Fatalf(ctx, "Failed to execute job: %v", err)
	}
}
```

That is the standard boilerplate for a Beam pipeline, it parses the flags, initialises Beam, delegates the pipeline construction to `buildPipeline` function, and finally runs the pipeline.

The interesting code begins in the `buildPipeline` function, which constructs the pipeline, by passing PCollections from one function to the next. To build up the tree we see in the above diagram.

```go
func buildPipeline(s beam.Scope) {
	// nothing -> PCollection<Painting>
	paintings := csvio.Read(s, *index, reflect.TypeOf(Painting{}))

	// PCollection<Painting> -> PCollection<CoGBK<string, Painting>>
	paintingsByGroup := GroupByDecade(s, paintings)

	// PCollection<CoGBK<string, Painting>> ->
	//   (PCollection<KV<string, Histogram>>, PCollection<KV<string, string>>)
	histograms, errors1 := ExtractHistogram(s, paintingsByGroup)

	// Calculate the color palette for the combined histograms.
	// PCollection<KV<string, Histogram>> ->
	//   (PCollection<KV<string, []color.RGBA>>, PCollection<KV<string, string>>)
	palettes, errors2 := CalculateColorPalette(s, histograms)

	// PCollection<KV<string, []color.RGBA>> -> PCollection<KV<string, string>>
	errors3 := DrawColorPalette(s, *outputPrefix, palettes)

	// PCollection<KV<string, []color.RGBA>> -> nothing
	WriteIndex(s, morebeam.Join(*outputPrefix, "index.json"), palettes)

	// PCollection<KV<string, string>> -> nothing
	WriteErrorLog(s, "errors.log", errors1, errors2, errors3)
}
```

To make it easy to follow, each function describes the step, and is annotated with a comment that explains what kind of PCollection is accepted and returned. Let's highlight some interesting steps.

```go
var (
	index = flag.String("index", "art.csv", "Index of the art.")
)

// Painting represents a single painting in the dataset.
type Painting struct {
	Artist string `csv:"artist"`
	Title  string `csv:"title"`
	Date   string `csv:"date"`
	Genre  string `csv:"genre"`
	Style  string `csv:"style"`

	Filename string `csv:"new_filename"`
...
}

...
func buildPipeline(s beam.Scope) {
	// nothing -> PCollection<Painting>
	paintings := csvio.Read(s, *index, reflect.TypeOf(Painting{}))
...
```

The very first step uses [`csvio.Read`](TODO) to read the CSV file specified by the `--index` flag, and returns a PCollection of Painting structs. In all the examples we’ve seen before we only pass basic types, strings, ints, etc in PCollections. But more complex types, such as a slices and structs are allowed (but not maps and interfaces). This makes it easier to pass rich information between the steps. The only caveat is the type must be JSON-serialisable. This is because in a distributed pipeline, the steps could be processed on separate machines, and the PCollection needs to be passed between them.

For Beam to successfully unmarshal your data, the types must be registered. This is done within the init() function, by called `beam.RegisterType`.

```go
func init() {
	beam.RegisterType(reflect.TypeOf(Painting{}))
}
```

If you forget to register the type, a error will occur at Runtime, such as:

```
java.util.concurrent.ExecutionException: java.lang.RuntimeException: Error received from SDK harness for instruction -224: execute failed: panic: reflect: Call using main.Painting as type struct { Artist string; Title string; ... } goroutine 70 [running]:
```

This can be a little frustrating, as when running the pipeline locally with the `direct` runner, it does not marshal your data, so errors like this aren’t exposed until running on Dataflow.

Now we have a collection of Paintings, we group them:
 
```go
// GroupByDecade takes a PCollection<Painting> and returns a 
// PCollection<CoGBK<string, Painting>> of the paintings group by decade.
func GroupByDecade(s beam.Scope, paintings beam.PCollection) beam.PCollection {
	s = s.Scope("GroupBy Decade")

	// PCollection<Painting> -> PCollection<KV<string, Painting>>
	paintingsWithKey := morebeam.AddKey(s, func(art Painting) string {
		return art.Decade()
	}, paintings)

	// PCollection<string, Painting> -> PCollection<CoGBK<string, Painting>>
	return beam.GroupByKey(s, paintingsWithKey)
}
```

The first line in this function, `s.Scope("GroupBy Decade")` allows us to name this step, and group multiple sub-steps. For example, in the above diagram “GroupBy Decade” is a single step, which can be expanded to show a `AddKey` and `GroupByKey` step.

`GroupByDecade` returns a PCollection<CoGBK<string, Painting>>. The CoGBK, is short for **Co**mmon **G**roup **B**y **K**ey. It is a special collection, where (as you’ll see later) each element is a tuple of a key, and an iterable collection of items. The key in this case is the Decade the painting was painted. The PCollection<Painting> is transformed into a PCollection<KV<String,Painting>> by the `morebeam.AddKey` step, adding a key to each value. Then the `GroupByKey` will use that key to produce the final PCollection.

Next up is the `ExtractHistogram`. This takes the PCollection<CoGBK<string, Painting>>, and this time returns two PCollections. The first is a PCollection<KV<string, Histogram>>, which is for every group of paintings in a decade, a [Histogram](https://en.wikipedia.org/wiki/Color_histogram) which represents the colors used in those paintings.

ExtractHistogram demonstrates three new concerns, `Stateful functions`, `Data enrichment`, and `Dead Letter error handling`.

## Stateful functions

```go
var (
	artPrefix = flag.String("art", "gs://mybucket/art", "Path to where the art is kept.")
)

func init() {
	beam.RegisterType(reflect.TypeOf((*extractHistogramFn)(nil)).Elem())
}

type extractHistogramFn struct {
	ArtPrefix string `json:"art_prefix"`

	fs filesystem.Interface
}

// ExtractHistogram calculates the color histograms for all the Paintings in
// the CoGBK.
func ExtractHistogram(s beam.Scope, files beam.PCollection)
		(beam.PCollection, beam.PCollection) {
	s = s.Scope("ExtractHistogram")
	return beam.ParDo2(s, &extractHistogramFn{
		ArtPrefix: *artPrefix,
	}, files)
}
```

Instead of passing a simple function to `beam.ParDo`, instead a struct is passed containing two fields. The exported field, `ArtPrefix` is the path to where the painting jpgs are stored, and the unexported field, `fs`, is a filesystem client for reading externally. 

When the pipeline runs, no global state is allowed, and the value of the command line flags are lost. For example, when running this pipeline we may start it like so:

```shell
go run image2palette.go \
  --art gs://${BUCKET?}/art/ \
  --runner dataflow \
  ...
```

However, when the code actually runs on the Dataflow instances, the `--art` flag is not specified. Thus the `*artPrefix` value will use the default value. To pass state like this, it must be part of the DoFn struct that is passed to `beam.ParDo`. So in this example, we create a `extractHistogramFn` struct, with the exported `ArtPrefix` field set to the value of the `--art` flag. Since this extractHistogramFn is then marshalled and passed to the workers, it must also be registered with beam, during the init.

When the pipeline wants to execute this step, it calls the `extractHistogramFn`’s `ProcessElement` method. This method works in a similar way to a simple function, with the arguments and return value being mapped to the PCollections being processed and returned.

## Iterating over a CoGBK

```go
func (fn *extractHistogramFn) ProcessElement(
ctx context.Context,
key string, values func(*Painting) bool,
errors func(string, string)) HistogramResult {

	log.Infof(ctx, "%q: ExtractHistogram started", key)
	var art Painting
	for values(&art) {
		filename := morebeam.Join(fn.ArtPrefix, art.Filename)
		h, err := fn.extractHistogram(ctx, key, filename)
		if err != nil {
			…
		}
		
		result.Histogram = result.Histogram.Combine(h)
	}

	return result
}
```

ProcessElement is called once for every unique group in the `PCollection<CoGBK<string, Painting>`. The `key string` argument will be the key for that group, and a `values func(*Painting) bool` is used to iterate all values within the group. The contact, is `values` is passed a pointer to a `Painting` struct, and returns true as long as there are more paintings to process in the group. As soon as it returns false, the group has been processed fully. This iterator pattern is unique to the `CoGBK` and make it convient to apply a operation to every element.

In this case, for each Painting, extractHistogram is called with fetches a jpg of the artwork, and extract a histogram of colors. The histograms from each painting is combined, and finally one result is returned for that group.

## Data enrichment

Reading the paintings from an external service (such as GCS) demonstrates a data enrichment step. This is where an external service is used to “enrich” the dataset the pipeline is process. You could imagine a user service be called when processing log entries, or a stock taking service when processing purchases. It should be noted, that any external action should be [idempotent](https://en.wikipedia.org/wiki/Idempotence). The pipeline may process the same element multiple times, if for example during processing the worker fails, the work is rescheduled on another worker and reprocessed. 

When calling an external service, typically some kind of client is constructed to initiate the connection. In this pipeline we read the images from GCS, thus setting up GCS client is useful. Since we are using a struct based DoFn, there are some additional methods that can be defined.

```go
func (fn *extractHistogramFn) Setup(ctx context.Context) error {
	var err error
	fn.fs, err = filesystem.New(ctx, fn.ArtPrefix)
	if err != nil {
		return fmt.Errorf("filesystem.New(%q) failed: %s", fn.ArtPrefix, err)
	}
	return nil
}

func (fn *extractHistogramFn) Teardown() error {
	return fn.fs.Close()
}
```

When the DoFn is initialized on the worker, the `Setup` method is called. Here we construct a new [Filesystem client](https://godoc.org/github.com/apache/beam/sdks/go/pkg/beam/io/filesystem) and store it in the struct’s fs field. Later, when the DoFn is no longer needed, the `Teardown` method is called, giving us opportunity to cleanup the client. With all things distributed, don’t expect the `Teardown` to ever be called.

There are some simple best practices, that should be following when calling an external services around catching errors.

```go
func (fn *extractHistogramFn) extractHistogram(ctx context.Context,
key, filename string) (palette.Histogram, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	fd, err := fn.fs.OpenRead(ctx, filename)
	if err != nil {
		return nil, fmt.Errorf("fs.OpenRead(%q) failed: %s", filename, err)
	}
	defer fd.Close()

	img, _, err := image.Decode(fd)
	if err != nil {
		return nil, fmt.Errorf("image.Decode(%q) failed: %s", filename, err)
	}

	return palette.NewColorHistogram(img), nil
}
```

The function begins by using a `context.WithTimeout`. This ensures that if the external service does not respond in a timely manner the context will be cancelled and a error returned. If this timeout wasn’t set, the external call may never end, and the pipeline never terminate.

The external service, may also return errors. Thus a new pattern is used called dead letter.

## Error handling and dead letters.

When Beam processes a PCollection, is bundles up multiple items into a bundle and processes one bundle at a time. If the PTransform return an error, panics, or otherwise fails (such as running out of memory), the bundle is retried. With Dataflow, Bundles are [retried up to four times](https://cloud.google.com/dataflow/docs/resources/faq#how-are-java-exceptions-handled-in-cloud-dataflow), after which the entire pipeline is aborted. This can be inconvenient, so instead we use a [dead letter pattern](https://en.wikipedia.org/wiki/Dead_letter_queue). This is a new PCollection that collects processing errors. These errors can then be stored at the end of the pipeline, manually inspected, and processed again later.

```go
return beam.ParDo2(s, &extractHistogramFn{
	ArtPrefix: *artPrefix,
}, files)
```

A keen observer would have noticed that `beam.ParDo2` was used by ExtractHistogram, instead of `beam.ParDo`. This function works the same, but returns two PCollections. In our case, the first is the normal output, and the second is a PCollection<KV<string, string>>. This second collection is keyed on the unique id of the painting have an issue, and the value the error message.

Since returning a error is optional, the errors PCollection was passed to `extractHistogramFn`’s `ProcessElement` as a `errors func(string, string)`.

Throughout we use this kind of errors PCollection from every stage, and at the end of the pipeline they are collected together and output to a errors log file:

```go
// WriteErrorLog takes multiple PCollection<KV<string,string>>s combines them
// and writes them to the given filename.
func WriteErrorLog(s beam.Scope, filename string, errors ...beam.PCollection) {
	s = s.Scope(fmt.Sprintf("Write %q", filename))

	c := beam.Flatten(s, errors...)
	c = beam.ParDo(s, func(key, value string) string {
		return fmt.Sprintf("%s,%s", key, value)
	}, c)

	textio.Write(s, morebeam.Join(*outputPrefix, filename), c)
}
```

Since the output is key, comma, value, the file can easily be re-read to try just the failed keys.

The rest of the pipeline is much of the same. `CalculateColorPalette` takes the color histograms and runs a K-Means clustering algorithm to extract the color palettes for those paintings. Those palettes are written out to png files with the `DrawColorPalette`, and finally all the palettes are written out to a JSON file in `WriteIndex`. 

## Gotchas

### Marshing
Always remember to register the types that will be transmitted between workers. This is anything that’s inside a PCollection, as well as any DoFn. Not all types are allowed, but slices, structs, and primitives are. For other types, custom JSON marshalling can be used.

It should also be reminded that state is not allowed. Flags, and other global variables will not always be populated when running on a remote worker. Also, examples like this may catch you out:

```go
prefix := “X”
s = s.Scope(“Prefix ” + prefix)
c = beam.ParDo(s, func(value string) string {
	return prefix + value
}, c)
```

This simple example, may appear to add “X” to the beginning of each element, however, it will prefix nothing. This is because, the simple anonymous function is marshalled, and unmarshalled on the worker. When it is then called on the worker, it does not capture the value of prefix. Instead prefix is the zero value. For this example to work, prefix must be defined inside the anonymous function, or a DoFn struct used which contains the prefix as a marshalled field.

### Errors
Since the pipeline could be running across 100s of workers, errors are to be expected. Thus making aggressive use of `log.Infof`, `log.Debugf`, etc is useful. This can make it very useful to debug why the pipeline gets stuck, or fails mysteriously.

While debugging this pipeline, it would fail occasionally due to exceeding the memory limits of the Dataflow VMs. To help debug this you can use [Go’s pprof](https://golang.org/pkg/net/http/pprof/) infrastructure.

```go
import (
	"net/http"
	_ "net/http/pprof"
)

func main() {
	...
	go func() {
		// HTTP Server for pprof (and other debugging)
		log.Info(ctx, http.ListenAndServe("localhost:8080", nil))
	}()
	…
}
```

This configures a webserver which can be used to export useful stats, and for grabbing profiling data.

### Difference between direct and dataflow runners

Running the pipeline locally is a quick way to validate the pipeline is setup, and that is runs as expected. However, running locally won’t run the pipeline in parallel, and it is obviously constrained to a single machine. There are some other difference, mostly around marshalling data. It’s always a good idea to test on Dataflow, perhaps smaller or sampled dataset as input, that can be used as a smoke test.

# Conclusion

This article has covered the basics of creating an Apache Beam pipeline with the Go SDK, while also covering some more advanced topics. The results of the specific pipeline example will be revealed in a later article. Until then the code is available here.

While the Beam Go SDK is still experimental, there are many great tutorials and example using the more mature Java and Python Beam SDKs [[1](https://medium.com/google-cloud/popular-java-projects-on-github-that-could-use-some-help-analyzed-using-bigquery-and-dataflow-dbd5753827f4), [2](https://medium.com/@vallerylancey/error-handling-elements-in-apache-beam-pipelines-fffdea91af2a)]. Google themselves even published a series of generic articles [[part 1](https://cloud.google.com/blog/products/gcp/guide-to-common-cloud-dataflow-use-case-patterns-part-1), [part 2](https://cloud.google.com/blog/products/gcp/guide-to-common-cloud-dataflow-use-case-patterns-part-2)].

