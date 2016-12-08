+++
linktitle = "Pachyderm"
date = "2016-12-13T00:00:00"
author = [ "Daniel Whitenack" ]
title = "Data Pipelines and Versioning with the Pachyderm Go Client"
series = ["Advent 2016"]
+++

**What is Pachyderm?**

[Pachyderm](http://pachyderm.io/) is an open source framework, written in Go, for reproducible data processing.  With Pachyderm, you can create [language agnostic data pipelines](http://pachyderm.io/pps.html) where the data input and output of each stage of your pipeline are versioned controlled in [Pachyderm's File System (PFS)](http://pachyderm.io/pfs.html).  Think "git for data."  You can view diffs of your data and collaborate with teammates using Pachyderm commits and branches. Moreover, if your data pipeline generates a surprising result, you can debug or validate it by understanding its historical processing steps (or even reproducing them exactly).

Pachyderm leverages the container ecosystem ([Kubernetes](http://kubernetes.io/) and [Docker](https://www.docker.com/)) to enable this functionality and to distribute your data processing.  It can parallize your computation by only showing a subset of your data to each container within a Pachyderm cluster. A single node either sees a slice of each file (a map job) or a whole single file (a reduce job). The data itself lives in any object store of your choice (usually S3 or GCS), and Pachyderm smartly assigns different pieces of data to be processed by different containers.

As mentioned, you can build your Pachyderm data pipelines using any languages or frameworks (python, Tensorflow, Spark, Rust, etc.), but, because it is written in Go, Pachyderm has a nice [Go client](https://godoc.org/github.com/pachyderm/pachyderm/src/client) that will let you launch pipelines, put data into data versioning, pull data out of data versioning, etc. directly from your Go applications.  For example, you could commit metrics from your Go backend service directly to Pachyderm and, on every commit, have Pachyderm automatically update predictive analysis (written using Go, Tensorflow, python, or whatever you might prefer) that is detecting fraudulent activity based on those metrics.

For more information visit [Pachyderm's website](http://pachyderm.io/) and look through [the docs](http://docs.pachyderm.io/en/latest/). 

**Some Simple Data Processing for This Post**

In this post, we are going to illustrate some distributed data processing and data versioning with a few simple Go programs and some Pachyderm configuration. This data processing will gather some statistics about Go projects posted to [Github](github.com).  We will:

1. Create a Pachyderm pipeline that takes Go repository names (e.g., `github.com/docker/docker`) as input and outputs as couple of stats/metrics about those repositories.  

2. Write a Go program that commits a series of repository names one at a time into Pachyderm's data versioning.  For each commit, we will automatically trigger the pipeline created in step 1 to update our stats.

To keep things simple, we will just calculate two statistics or metrics for each repository, number of lines of Go code and number of dependencies.  So our input data (supplied by the Go program written in step 2) will look something like this:

```
github.com/myusername/projectname
```

and our output data will look like this:

```
github.com/myusername/projectname, 4, 350
```

where we calculated that `github.com/myusername/projectname` contained 350 lines of Go code and imported 4 dependencies.  As we commit more and more input data, we will update our statistics.  For example, if we commit additional input data in the form of:

```
github.com/myusername/anotherprojectname
```

we will have Pachyderm automatically update our results:

```
github.com/myusername/projectname, 4, 350
github.com/myusername/anotherprojectname, 8, 427
```

**Step 1: Creating a Pachyderm Pipeline**

blah


**Resources**

The official docs for the Go assembler at https://golang.org/doc/asm.  They're
useful to read, but remember that PeachPy will be taking care of the many of
the details for you regarding the syntax and calling convention.

The [PeachPy sources](https://github.com/Maratyszcza/PeachPy)

Finally, at GolangUK 2016, Michael Munday gave a talk on [Dropping Down: Go Functions in
Assembly](https://www.youtube.com/watch?v=9jpnFmJr2PE).
