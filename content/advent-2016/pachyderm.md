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

1. Deploy Pachyderm.

2. Create a Pachyderm pipeline that takes Go repository names (e.g., `github.com/docker/docker`) as input and outputs as couple of stats/metrics about those repositories.  

3. Write a Go program that commits a series of repository names one at a time into Pachyderm's data versioning.  For each commit, we will automatically trigger the pipeline created in step 2 to update our stats.

To keep things simple, we will just calculate two statistics or metrics for each repository, number of lines of Go code and number of dependencies.  So our input data (supplied by the Go program written in step 3) will look something like this:

```
myusername/projectname
```

where the `github.com/` prefix is assumed to be there.  Our output data will look like this:

```
myusername/projectname, 4, 350
```

where we calculated that `github.com/myusername/projectname` contained 350 lines of Go code and imported 4 dependencies.  As we commit more and more input data, we will update our statistics.  For example, if we commit additional input data in the form of:

```
myusername/anotherprojectname
```

we will have Pachyderm automatically update our results:

```
myusername/projectname, 4, 350
myusername/anotherprojectname, 8, 427
```

**Step 1: Deploying Pachyderm**

Our Pachyderm pipeline will run in, where else, but a Pachyderm cluster.  Thus, let's get our Pachyderm cluster running.  Thankfully, this can be done in [just a few commands](http://docs.pachyderm.io/en/latest/getting_started/local_installation.html) locally, or via one of a number of [deploy commands](http://docs.pachyderm.io/en/latest/deployment/deploying_on_the_cloud.html) for Google, Amazon, or Azure cloud platforms.


After going through one of these simple deploys, you can verify that your Pachyderm cluster is running with the Pachyderm CLI tool `pachctl`:

```
$ pachctl version
COMPONENT           VERSION
pachctl             1.3.0
pachd               1.3.0
```

**Step 2a: Creating a Pachyderm Pipeline**

To create a pachyderm pipeline we need:

- One or more Docker images that will be used for our data processing (in this case to calculate number of lines of Go code and number of dependencies).
- A [JSON pipeline specification](http://docs.pachyderm.io/en/latest/deployment/pipeline_spec.html).

To get our metrics for each input project, let's just use `wc -l` to get the number of lines of go codes and `go list` to get the number of dependencies.  We will put these commands in a shell script that can be run in a Docker image built `FROM golang`.  

_Aside:_ Note, even though our "processing" is simple in this example, one of the beauties of Pachyderm is that we can use any Docker images for our processing. We can use any language or framework and any logic from simple unix commands to recurrent neural networks implemented in [Tensorflow](https://www.tensorflow.org/).

Here is the shell script, `stats.sh` that we will use:

```sh
#!/bin/bash

# Grab the source code
go get -d github.com/$REPONAME/...

# Grab Go package name
pkgName=github.com/$REPONAME

# Grab just first path listed in GOPATH
goPath="${GOPATH%%:*}"

# Construct Go package path
pkgPath="$goPath/src/$pkgName"

if [ -e "$pkgPath/Godeps/_workspace" ];
then
  # Add local godeps dir to GOPATH
  GOPATH=$pkgPath/Godeps/_workspace:$GOPATH
fi

# get the number of dependencies in the repo
go list $pkgName/... > dep.log || true
deps=`wc -l dep.log | cut -d' ' -f1`;
rm dep.log

# get number of lines of go code
golines=`( find $pkgPath -name '*.go' -print0 | xargs -0 cat ) | wc -l`

# output the stats
echo $REPONAME, $deps, $golines
```

This includes the `wc -l` and `go list` functionality along with some clean up and things to support [Godep](https://github.com/tools/godep).  Then our Docker image is simple the `golang` image plus this script:

```
FROM golang
ADD stats.sh /
```

**Resources**

The official docs for the Go assembler at https://golang.org/doc/asm.  They're
useful to read, but remember that PeachPy will be taking care of the many of
the details for you regarding the syntax and calling convention.

The [PeachPy sources](https://github.com/Maratyszcza/PeachPy)

Finally, at GolangUK 2016, Michael Munday gave a talk on [Dropping Down: Go Functions in
Assembly](https://www.youtube.com/watch?v=9jpnFmJr2PE).
