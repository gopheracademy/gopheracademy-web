+++
linktitle = "Pachyderm"
date = "2016-12-13T00:00:00"
author = [ "Daniel Whitenack (@dwhitena)" ]
title = "Data Pipelines and Versioning with the Pachyderm Go Client"
series = ["Advent 2016"]
+++

# I know about Gophers, but what is a Pachyderm?

[Pachyderm](http://pachyderm.io/) is an open source framework, written in Go, for reproducible data processing.  With Pachyderm, you can create [language agnostic data pipelines](http://pachyderm.io/pps.html) where the data input and output of each stage of your pipeline are versioned controlled in [Pachyderm's File System (PFS)](http://pachyderm.io/pfs.html).  Think "git for data."  You can view diffs of your data and collaborate with teammates using Pachyderm commits and branches. Also, if your data pipeline generates an unexpected result, you can debug or validate it by understanding the historical processing steps leading to the result (or even reproducing them exactly).

Pachyderm leverages the container ecosystem ([Kubernetes](http://kubernetes.io/) and [Docker](https://www.docker.com/)) to enable this functionality and to distribute your data processing.  It can parallelize your computation by only showing a subset of your data to each container within a Pachyderm cluster. A single node either sees a slice of each file (a map job) or a whole single file (a reduce job). The data itself can live in an object store of your choice (usually S3 or GCS), and Pachyderm smartly assigns different pieces of data to be processed by different containers.

As mentioned, you can build your Pachyderm data pipelines using any languages or frameworks (Go, Python, Tensorflow, Spark, Rust, etc.), but Pachyderm does have a nice [Go client](https://godoc.org/github.com/pachyderm/pachyderm/src/client) that will let you launch pipelines, put data into data versioning, pull data out of data versioning, etc. directly from your Go applications.  For example, you could commit metrics from your Go backend service directly to Pachyderm and, on every commit, have Pachyderm automatically update predictive analysis (written in Go, Tensorflow, python, or whatever you might prefer) to detect fraudulent activity based on those metrics.

To read more about the Pachyderm project, visit [Pachyderm's website](http://pachyderm.io/) and look through [the docs](http://docs.pachyderm.io/en/latest/). 

# A data processing example for this post

In this post, we are going to illustrate some distributed data processing and data versioning with a few simple Go programs and some Pachyderm configuration. The goal of the data processing will be to generate statistics about Go projects posted to [GitHub](github.com).  We will:

1. Deploy Pachyderm.

2. Create a Pachyderm pipeline that takes Go repository names (e.g., `github.com/docker/docker`) as input and outputs a couple of stats/metrics about those repositories.  

3. Write a Go program that commits a series of repository names one at a time into Pachyderm's data versioning.  For each commit, we will automatically trigger the pipeline created in step 2 to update our stats.

To keep things simple, we will just calculate two statistics/metrics for each repository, number of lines of Go code and number of dependencies.  Our input data (supplied by the Go program written in step 3) will look something like this:

```
myusername/projectname
```

where the `github.com/` prefix is implied, and our output data will look like this:

```
myusername/projectname, 4, 350
```

where 4 is the number of dependencies in `github.com/myusername/projectname` and 350 is the number of lines of Go code.  As we commit more and more input data, we will update our statistics.  For example, if we commit additional input data:

```
myusername/anotherprojectname
```

we will have Pachyderm automatically update our results:

```
myusername/projectname, 4, 350
myusername/anotherprojectname, 8, 427
```

The output of the pipeline will be version controlled by Pachyderm as well.  This is pretty cool, because, if/when we wanted, we could look back in time to what our stats were at any commit.  Further we could deduce the provenance of the results (i.e., what data and calculations led to those results) and reproduce them exactly.  To learn more about this "data provenance" feature, check out [this article](https://medium.com/pachyderm-data/provenance-the-missing-feature-for-good-data-science-now-in-pachyderm-1-1-2bd9d376a7eb#.1jyvi2ek2).

## Step 1: Deploying Pachyderm

Our Pachyderm pipeline will run in, where else, but a Pachyderm cluster.  Thus, let's get a Pachyderm cluster up and running.  Thankfully, Pachyderm can be deployed in [just a few commands](http://docs.pachyderm.io/en/latest/getting_started/local_installation.html) locally, or via a [deploy command](http://docs.pachyderm.io/en/latest/deployment/deploying_on_the_cloud.html) for Google, Amazon, or Azure cloud platforms.


After completing one of these deploys, you can verify that your Pachyderm cluster is running with the Pachyderm CLI tool `pachctl`:

```
$ pachctl version
COMPONENT           VERSION
pachctl             1.3.0
pachd               1.3.0
```

where `pachd` is the Pachyderm server daemon.

## Step 2a: Creating a Pachyderm Pipeline

To create a pachyderm pipeline we need:

- One or more Docker images that will be used to process/manipulate input data (in this case to calculate number of lines of Go code and number of dependencies), and
- A [JSON pipeline specification](http://docs.pachyderm.io/en/latest/deployment/pipeline_spec.html) that tells Pachyderm which images to use, how to parallelize, what to use as input data, etc.

Regarding the first piece, let's just run `wc -l` to get the number of lines of go codes and `go list` to get the number of dependencies for each Go project.  We will put these commands in a shell script that can be run in a Docker image built `FROM golang`.  

_Aside:_ Note, even though our "processing" is simple in this example, one of the beauties of Pachyderm is that you can use any Docker images for the data processing. This includes any language or framework and any logic from simple unix commands to recurrent neural networks implemented in [Tensorflow](https://www.tensorflow.org/).

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

This includes the `wc -l` and `go list` commands along with some clean up and things to support [Godep](https://github.com/tools/godep).  The script will output our metrics given a GitHub Go project name input to the variable `REPONAME`.  Note, the metrics are output to `/pfs/out`, which is a way of telling Pachyderm to output something to the output data repository of a pipeline stage (which will be explained further below).  Our Docker image is simply the `golang` image plus this script:

```
FROM golang
ADD stats.sh /
```

We can then create a JSON Pachyderm pipeline specification, `pipeline.json`, that uses this image (uploaded to Docker Hub as `dwhitena/stats`) to process our input data:

```json
{
  "pipeline": {
    "name": "stats"
  },
  "transform": {
    "image": "dwhitena/stats",
    "cmd": [ "/bin/bash" ],
    "stdin": [
        "for filename in /pfs/projects/*; do",
		"REPONAME=`cat $filename`",
		"source /stats.sh >> /pfs/out/results",	
	"done"
    ]
  },
  "parallelism_spec": {
    "strategy": "CONSTANT",
    "constant": "1"
  },
  "inputs": [
    {
      "repo": {
        "name": "projects"
      },
      "method": "reduce"
    }
  ]
}
```

In this specification, we are telling Pachyderm the following:

- Use the `dwhitena/stats` image.
- Run the command `/bin/bash` in the container with the specified `stdin`.
- We are not parallelizing this processing yet (although we could specify a specific parallelization by simply changing the `constant` field under `parallelism_spec`).
- The data input for this pipeline is a "repo" named `projects`.  Remember Pachyderm's data versioning is similar to "git for data."  In this case, we are telling the pipeline to look for input in the versioned data repository called `projects`.  We could specify multiple repositories if we wish.  When the container runs, it will have access to the specified repos at `/pfs/<reponame>` (`/pfs/projects` here).
- We are accessing the input via a `reduce` method.  This means that as data is committed, the pipeline will only see the new data, and, if we introduced parallelism, Pachyderm could distribute the data over the containers at a block level.  For more information on this method and other see the docs for [Combining Parition Unit and Incrementality](http://docs.pachyderm.io/en/latest/deployment/pipeline_spec.html#combining-partition-unit-and-incrementality).

In essense, when new data is committed to a data repository call `projects`, this pipeline will be triggered and process the data using the specified image, cmd, and stdin.  The pipeline only has one "stage."  That is, we are only specifying one data processing step that takes data from the `projects` repository and outputs our metrics.  If we wanted to have more stages in our pipeline (e.g., a stage that compiles the project and measures compile time), we would just need to add more JSON blobs to `pipeline.json`.  See Pachyderm's [fruit stand example](http://pachyderm.readthedocs.io/en/stable/getting_started/beginner_tutorial.html) to see a multistage pipeline in action.

## Step 2b: Run and Test the Pachyderm Pipeline

At this point, we have the following:

- A script called `stats.sh` that calculates metrics for a Go project given a GitHub repository name.
- A docker image `dwhitena/stats` containing the script.
- A pipeline spec called `pipeline.json` that runs the script on data committed to a `projects` data repository.

Before we run this pipeline using the Go client, let's run it manually to ensure that it works and to gain some intuition about Pachyderm's pipelining and data versioning. To run the pipeline manually, we first need to create the `projects` data repository with the `pachctl` CLI tool:

```
$ pachctl create-repo projects
```

Next, let's create the pipeline with our JSON specification:

```
$ pachctl create-pipeline -f pipeline.json
```

At this point, we haven't committed any data into Pachyderm's data versioning, and, thus, our pipeline doesn't have any input data to process.  However, we can verify that our pipeline and repository exist:

```
$ pachctl list-repo
NAME                CREATED             SIZE                
projects            18 seconds ago      0 B                 
stats               5 seconds ago       0 B                 
$ pachctl list-pipeline
NAME                INPUT               OUTPUT              STATE               
stats               projects            stats               running 
```

Notice that an output repository (with the name of our pipeline) has also been created.  The output of our `stats` pipeline will be versioned there.  

Now, let's commit some data into the input data repository called `projects`.  Specifically we will commit a first file `one.txt` into the `projects` repository on the `master` branch, where `one.txt` includes the GitHub repository name that we want to analyze (`docker/docker`):

```
echo "docker/docker" | pachctl put-file projects master one.txt -c
```

This will trigger the first run of our `stats` pipeline, and we can confirm that the pipeline ran via:

```
$ pachctl list-job
ID                                 OUTPUT                                     STARTED              DURATION            STATE               
4c4f53668e46c20fdeb1286ca971ea1f   stats/9c8c1ad2667d44d586818306ec19f1ec/0   About a minute ago   57 seconds          success 
```

The `OUTPUT` column above shows the location of the output of our `stats` pipeline within Pachyderm's data versioning in the pattern `<repo name>/<branch>/<commit>`.  To see what our output looks like we can list the files in the output and get the `results` file:

```
$ pachctl list-file stats 9c8c1ad2667d44d586818306ec19f1ec
NAME                TYPE                MODIFIED            LAST_COMMIT_MODIFIED                 SIZE                
/results            file                About an hour ago   9c8c1ad2667d44d586818306ec19f1ec/0   27 B  
$ pachctl get-file stats 9c8c1ad2667d44d586818306ec19f1ec /results
docker/docker, 559, 798271   
```

The `results` file shows that our pipeline determined that the `github.com/docker/docker` project has 559 dependencies and 798,271 lines of Go code.  Pretty cool. Let's commit another repository and see what happens:

```
echo "kubernetes/kubernetes" | pachctl put-file projects master two.txt -c
```

We can now see that two commits have been made to the `projects` data repository:

```
$ pachctl list-commit projects
BRANCH              REPO/ID             PARENT              STARTED             FINISHED            SIZE                
master              projects/master/0   <none>              About an hour ago   About an hour ago   14 B                
master              projects/master/1   master/0            50 seconds ago      49 seconds ago      22 B
```

where the second commit added the second file (and thus has a parent commit with the first file).  When we added this second commit, Pachyderm automatically ran our `stats` pipeline based on the newly committed data.  We can confirm this with `list-job` again, and we can also see that there are two commits in our output `stats` repo:

```
$ pachctl list-commit stats
BRANCH                             REPO/ID                                    PARENT              STARTED             FINISHED            SIZE                
9c8c1ad2667d44d586818306ec19f1ec   stats/9c8c1ad2667d44d586818306ec19f1ec/0   <none>              About an hour ago   About an hour ago   27 B                
e1a8a83729d64ebeb9d8549ae9581e3f   stats/e1a8a83729d64ebeb9d8549ae9581e3f/0   <none>              3 minutes ago                           0 B
$ pachctl list-commit stats
BRANCH                             REPO/ID                                    PARENT              STARTED             FINISHED            SIZE                
9c8c1ad2667d44d586818306ec19f1ec   stats/9c8c1ad2667d44d586818306ec19f1ec/0   <none>              2 hours ago         2 hours ago         27 B                
e1a8a83729d64ebeb9d8549ae9581e3f   stats/e1a8a83729d64ebeb9d8549ae9581e3f/0   <none>              10 minutes ago      3 minutes ago       64 B
```

Now if we get the `results` file, we will see that the new commit was processed:

```
dwhitena@dirac:pachyderm-go-stats$ pachctl get-file stats e1a8a83729d64ebeb9d8549ae9581e3f /results
docker/docker, 559, 798232
kubernetes/kubernetes, 1459, 2800450
```

Sweet! We can keep committing new files and the results will keep getting updated.  Not only that, because everything is versioned, we can still access the state the results at any point in history.  For example, we can still access the `results` file at the previous commit:

```
$ pachctl get-file stats 9c8c1ad2667d44d586818306ec19f1ec /results
docker/docker, 559, 798271
```

## Step 3a: Write a Go program that uses the Pachyderm client

The `pachctl` CLI is useful, but now let's use Pachyderm's Go client to stream a series of Go project names into the `projects` repo and, in turn, calculate stats for each of the committed projects.  

First, create a program `feed.go` that imports Pachyderm's Go client, connects to our Pachyderm cluster, and creates the `projects` repo:

```go
package main

import (
	"log"

	"github.com/pachyderm/pachyderm/src/client"
)

func main() {

	// Connect to Pachyderm.
	c, err := client.NewFromAddress("0.0.0.0:30650")
	if err != nil {
		log.Fatal(err)
	}

	// Create a repo called "projects."
	if err := c.CreateRepo("projects"); err != nil {
		log.Fatal(err)
	}
}
```

Then add a loop that commits files to `projects`, where each successive file includes the name of a different Go project on GitHub:

```go
// ...

// projects includes a list of example Go projects on Github.
var projects = []string{
	"docker/docker",
	"kubernetes/kubernetes",
	"hashicorp/consul",
	"spf13/hugo",
	"prometheus/prometheus",
	"influxdata/influxdb",
	"coreos/etcd",
}

func main() {

	// ...

	// Loop over the projects.
	for idx, project := range projects {

		// Start a commit in our "projects" data repo on the "master" branch.
		commit, err := c.StartCommit("projects", "master")
		if err != nil {
			log.Fatal(err)
		}

		// Put a file containing the respective project name.
		if _, err := c.PutFile("projects", commit.ID, strconv.Itoa(idx), strings.NewReader(project)); err != nil {
			log.Fatal(err)
		}

		// Finish the commit.
		if err := c.FinishCommit("projects", commit.ID); err != nil {
			log.Fatal(err)
		}
	}
}
```

Finally, let's add the functionality to create the `stats` pipeline:

```go
// ...

func main() {

	// ...

	// Define the stdin for our pipeline.
	stdin := []string{
		"for filename in /pfs/projects/*; do",
		"REPONAME=`cat $filename`",
		"source /stats.sh >> /pfs/out/results",
		"done",
	}

	// Create our stats pipeline.
	if err := c.CreatePipeline(
		"stats",               // the name of the pipeline
		"dwhitena/stats",      // your docker image
		[]string{"/bin/bash"}, // the command run in your docker image
		stdin,
		nil, // let pachyderm decide the parallelism
		[]*pps.PipelineInput{
			// reduce over "projects"
			client.NewPipelineInput("projects", client.ReduceMethod),
		},
		false, // not an update
	); err != nil {
		log.Fatal(err)
	}
}
```

The entire `feed.go` program with all of these pieces can be found [here](https://github.com/dwhitena/pachyderm-go-stats/blob/master/feed.go).

## Step 3b: Runnning our Go program and examining the results

The above `feed.go` program will do everything we did in steps 2a and 2b, but directly from Go itself.  Moreover, it allows us to automate the commits of input data.  

After deleting our previous testing data and pipeline with `pachctl delete-all`, we can compile and run this program.  We will then observe a few things.  First, all of our commits to the input repo `projects` show up:

```
$ pachctl list-commit projects
BRANCH              REPO/ID             PARENT              STARTED             FINISHED            SIZE                
master              projects/master/0   <none>              6 minutes ago       6 minutes ago       13 B                
master              projects/master/1   master/0            6 minutes ago       6 minutes ago       21 B                
master              projects/master/2   master/1            6 minutes ago       6 minutes ago       16 B                
master              projects/master/3   master/2            6 minutes ago       6 minutes ago       10 B                
master              projects/master/4   master/3            6 minutes ago       6 minutes ago       21 B                
master              projects/master/5   master/4            6 minutes ago       6 minutes ago       19 B                
master              projects/master/6   master/5            6 minutes ago       6 minutes ago       11 B  
``` 

Also, we will see that Pachyderm is running our pipeline for each of the commits of data into `projects`:

```
$ pachctl list-job
ID                                 OUTPUT                                     STARTED             DURATION            STATE               
d9255dbe5df83e25ba3d3c890e022598   stats/cd309fd23a3745b68dafe56b581d63bf/0   2 minutes ago       -                   running    
d15c3e962e2ed2412be460db26a21c85   stats/46b8270e1e2c4fdd990d89eaa4b4351b/0   2 minutes ago       -                   running    
7d7a88b806fefcf8f2166492eb7e32a6   stats/65e0d4edaca7483fa06c7c2443133774/0   2 minutes ago       -                   running    
35f6c95d843a12f506e3d18c94dfffab   stats/ac98664623a74f029506ec10701387c1/0   2 minutes ago       -                   running    
fd5b8623ca2556e0da4b71fc205c6f52   stats/f7ebb735f2fa46b894d3ce81e8cb8855/0   2 minutes ago       -                   running    
16f1108c71e6a0da6dca4712e54d4ddb   stats/ff7a262a03ea4f279e92c6d54c7e8129/0   2 minutes ago       -                   running    
4c4f53668e46c20fdeb1286ca971ea1f   stats/930e29b046a948649176b04225a547d9/0   2 minutes ago       -                   running   
```

Eventually these will finish, and we will see each corresponding commit to our output data repository `stats`:

```
$ pachctl list-commit stats
BRANCH                             REPO/ID                                    PARENT              STARTED             FINISHED            SIZE                
46b8270e1e2c4fdd990d89eaa4b4351b   stats/46b8270e1e2c4fdd990d89eaa4b4351b/0   <none>              27 minutes ago      59 seconds ago      183 B               
65e0d4edaca7483fa06c7c2443133774   stats/65e0d4edaca7483fa06c7c2443133774/0   <none>              27 minutes ago      3 minutes ago       151 B               
930e29b046a948649176b04225a547d9   stats/930e29b046a948649176b04225a547d9/0   <none>              27 minutes ago      23 minutes ago      27 B                
ac98664623a74f029506ec10701387c1   stats/ac98664623a74f029506ec10701387c1/0   <none>              27 minutes ago      3 minutes ago       116 B               
cd309fd23a3745b68dafe56b581d63bf   stats/cd309fd23a3745b68dafe56b581d63bf/0   <none>              27 minutes ago      22 seconds ago      208 B               
f7ebb735f2fa46b894d3ce81e8cb8855   stats/f7ebb735f2fa46b894d3ce81e8cb8855/0   <none>              27 minutes ago      3 minutes ago       94 B                
ff7a262a03ea4f279e92c6d54c7e8129   stats/ff7a262a03ea4f279e92c6d54c7e8129/0   <none>              27 minutes ago      5 minutes ago       64 B
```

Finally, we can examine our output file, `results`, to see all of the nice metrics for the Go projects we supplied:

```
$ pachctl get-file stats cd309fd23a3745b68dafe56b581d63bf /results
docker/docker, 559, 798232
kubernetes/kubernetes, 1459, 2801166
hashicorp/consul, 127, 357396
spf13/hugo, 15, 39430
prometheus/prometheus, 298, 889665
influxdata/influxdb, 68, 126163
coreos/etcd, 157, 407630
```

where, as a reminder, the first number in each row is the number of dependencies in the project and the second number is the number of lines of Go code in the project (at least at the time of writing this post).  Yay for data versioning, pipelining, and analysis in Go!  Be sure to try this out on your own and replace the projects above with the ones that are interesting to you.

# Resources

All the code and files mentioned in this post can be found [here](https://github.com/dwhitena/pachyderm-go-stats).

[Get started with Pachyderm](http://docs.pachyderm.io/en/latest/getting_started/getting_started.html) now by installing it in [just a few commands](http://docs.pachyderm.io/en/latest/getting_started/local_installation.html).  Also be sure to:

- [Join the Pachyderm Slack community](http://slack.pachyderm.io/) for questions, discussions, deployment help, etc.
- Read [the Pachyderm docs](http://docs.pachyderm.io/en/latest/).
- Read [the godocs](https://godoc.org/github.com/pachyderm/pachyderm/src/client) for the Pachyderm Go client.
- Check out [example Pachyderm pipelines](http://docs.pachyderm.io/en/latest/examples/readme.html).
- Connect with Pachyderm [on Twitter](https://twitter.com/pachydermIO).

Also, more generally, make sure to get involved in the Go data science community:

- Join the `#data-science` channel on [Gophers Slack](https://invite.slack.golangbridge.org/).
- Check out the [gopherds org on GitHub](https://github.com/gopherds).  Specifically, visit the [resources repo](https://github.com/gopherds/resources) for [a list of Go data science packages](https://github.com/gopherds/resources/tree/master/tooling) and [community updates](https://github.com/gopherds/resources/tree/master/community).
- Join the [GopherDS mailing list](https://groups.google.com/forum/#!forum/gopherds).
- Attend community events and workshops, such as data-related talks at Gopher conferences and meetups or public [Ardan Labs](https://www.ardanlabs.com/) [Ultimate Data Science](https://www.ardanlabs.com/ultimate-data-science) courses.
