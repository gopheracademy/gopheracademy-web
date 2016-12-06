+++
author = ["NATS.io team"]
date = "2016-12-06T0:00:00-08:00"
title = "Using NATS Messaging with some of your favorite Golang tools"
series = ["Advent 2016"]
+++

# Quick Intro to NATS, and Why We Love Go!

For those of you who have been reading GopherAcademy for a while, you may already be familiar with [NATS](https://nats.io/) via last year’s [post](https://blog.gopheracademy.com/advent-2015/nats-high-performance-cloud-native-messaging-written-in-go/), or you may have known about NATS for a while before that - NATS was one of the earliest production applications written in Golang. 
NATS is a very, very simple messaging system (just like Go is a simple to use development language), and shares many of the same characteristics developers like about Go. For anyone building cloud native or microservices based applications, some (if not all) of your stack these days is likely Golang, and NATS should make your life quite a bit easier.

In this blog post, we want to take a look at two things: 1) What do we love about Go on the NATS team? 2) How are Go developers using NATS with other tools in the Go ecosystem?

### What are some things we love about Go, and how are we using it in NATS?
* We like the performance Go provides across the major platforms (Linux, OSX, Windows).  While other technologies support multiple platforms like Go, they do not compile down to native binaries, sacrificing performance and requiring a larger footprint with additional runtime components to be installed.
* [Goroutines](https://tour.golang.org/concurrency/1) eliminate the need to manage threads and thread pools - they are very simple to use, extremely lightweight, and performant.
* Built-in facilities like [Channels](https://tour.golang.org/concurrency/2) and WaitGroups make it really easy to use a variety of concurrency patterns. 
* The consistent formatting style enforced in Go facilitates development in a team environment.
* The rich and portable “net” package lets us focus on the important things, rather than low-level and traditionally error prone socket code.

In addition to being a great development language (obviously) that comes with all sorts of great primitives, Golang also comes with a community and ecosystem second to none. That is the focus the remainder of this blog post - we’ll delve a bit further into some of the interesting things happening in the NATS community this year, and how developers are implementing NATS alongside a variety of popular Go-based projects in their infrastructure

### What’s new in 2016 with NATS?

From a product standpoint, there have been quite a few important updates. The most important was the launch of [NATS Streaming](http://nats.io/documentation/streaming/nats-streaming-intro/) this past summer. NATS Streaming adds big data and IoT semantics such as message replay, persistence, and durable subscriptions, if this is something you require. This was implemented as a separate library rather than baked into the core NATS Server to maintain the simplicity we all enjoy with NATS. Subject based authorization was also added to NATS this year, and the team is also in the process of delivering clustering for NATS Streaming - you can look forward to that in the first half of 2017.

Quite a lot has also been going on with the growth of the NATS community, as any of you present at GopherCon this year may remember. Our team got to meet dozens of you at Hackday thanks to a packed room and [excellent workshop](https://github.com/ardanlabs/gotraining/tree/master/topics/nats) put on by the one and only [Bill Kennedy](https://twitter.com/goinggodotnet).

<blockquote class="twitter-tweet" data-lang="en"><p lang="en" dir="ltr"><a href="https://twitter.com/goinggodotnet">@goinggodotnet</a> kicking off <a href="https://twitter.com/nats_io">@nats_io</a> workshop on <a href="https://twitter.com/hashtag/gophercon?src=hash">#gophercon</a> Hackday..fun few hours ahead! <a href="https://t.co/e7ayCxLUaz">pic.twitter.com/e7ayCxLUaz</a></p>&mdash; Brian Flannery (@brianflannery) <a href="https://twitter.com/brianflannery/status/753260354696728576">July 13, 2016</a></blockquote>
<script async src="//platform.twitter.com/widgets.js" charset="utf-8"></script>

A few doors down, while the NATS Workshop was unfolding, the team at [Gobot.io](http://www.gobot.io) were holding their own Hackday session, where [Cale Hoopes](https://twitter.com/calehoopes) submitted a winning entry using NATS, GoBot, and ReactNative:

<blockquote class="twitter-tweet" data-lang="en"><p lang="en" dir="ltr">Fun mobile-based distributed led on/off hack at <a href="https://twitter.com/GopherCon">@GopherCon</a> with <a href="https://twitter.com/calehoopes">@calehoopes</a> using <a href="https://twitter.com/nats_io">@nats_io</a> + <a href="https://twitter.com/gobotio">@gobotio</a> + <a href="https://twitter.com/reactnative">@reactnative</a> <a href="https://t.co/qQyXbtzuu3">pic.twitter.com/qQyXbtzuu3</a></p>&mdash; Bret Marzolf (@marzolfb) <a href="https://twitter.com/marzolfb/status/753308179052638208">July 13, 2016</a></blockquote>
<script async src="//platform.twitter.com/widgets.js" charset="utf-8"></script>

Since GopherCon, there have been some excellent talks at Meetups about using NATS with Go tooling. 

Wally Quevedo, who maintains the [Python Asyncio](http://nats.io/download/nats-io/asyncio-nats/) NATS, and [Ruby NATS](http://nats.io/download/nats-io/ruby-nats/) client libraries (as well as the monitoring utility [nats-top](https://github.com/nats-io/nats-top)) gave an overview on how to use NATS with Docker’s 1.12 Release at the Docker meetup in October:

<iframe width="560" height="315" src="https://www.youtube.com/embed/X4m-voD3zjU" frameborder="0" allowfullscreen></iframe>

You may find the slides on creating a [NATS Cluster in Swarm mode](http://www.slideshare.net/wallyqs/nats-docker-meetup-talk-oct-2016#80) particularly interesting.

In November at the [Phoenix Golang Meetup](http://www.meetup.com/Golang-Phoenix/), [Cesar Gonzalez](https://twitter.com/codaheck) of Bolste gave an overview on using NATS for Event Handling:

<iframe width="560" height="315" src="https://www.youtube.com/embed/fCp7DwGfmo4" frameborder="0" allowfullscreen></iframe>

If you’re hosting a Golang Meetup in your area and want to include NATS in a talk let [Brian](https://twitter.com/brianflannery) on our team know; we’d be happy to get you some swag and support you how we can!

### NATS as the glue for the new Go-based Microservices Stack

#### Kubernetes
Using NATS with Kubernetes is a breeze, and as our development community like to remind us, almost ‘too easy’:


<blockquote class="twitter-tweet" data-lang="en"><p lang="en" dir="ltr">In a few hours we set this up &amp; configured our architecture for messaging in <a href="https://twitter.com/hashtag/k8?src=hash">#k8</a>. Almost too easy... <a href="https://twitter.com/hashtag/natsio?src=hash">#natsio</a> <a href="https://twitter.com/hashtag/tbe?src=hash">#tbe</a> <a href="https://t.co/wAo4fOJZxL">https://t.co/wAo4fOJZxL</a></p>&mdash; Effectively Bo (@theBoEffect) <a href="https://twitter.com/theBoEffect/status/799415095164026880">November 18, 2016</a></blockquote>
<script async src="//platform.twitter.com/widgets.js" charset="utf-8"></script>


[Paulo Pires](https://github.com/pires) has been a member of the Kubernetes and NATS development communities for quite a while. He’s a very active contributor to both projects. He’s done a variety of ‘clustering on Kubernetes made easy’ tutorials and repos: Elasticsearch, Hazelcast, and of course NATS: <https://github.com/pires/kubernetes-nats-cluster>

Next up we hear Pires plans to implement an [operator-based model](https://github.com/pires/kubernetes-nats-cluster/issues/5) (Controllers + TPR i.e. Third Party Resources) to allow you to manage NATS clusters from within Kubernetes in programmable manner vs the current recipe-driven method so we’re looking forward to that, as well!

#### Docker
NATS Server has been an [Official Docker Image](https://hub.docker.com/_/nats/) available on DockerHub for approximately a year and a half now. The simplicity, performance, and scalability of NATS make it a natural fit for anyone developing a container-based architecture. The Docker Image is just 6MB and a few layers, making it one of the smallest Official Images around. The image has now been pulled over a million times and is a popular Golang developer tool for anyone working with Docker infrastructure.

NATS Streaming has now also joined NATS Server as an Official Image. You can pull NATS Streaming via [DockerHub](https://hub.docker.com/_/nats-streaming/).

The talk we’ve already mentioned above from Wally gives some very practical examples of how to get started with NATS and Docker, and you can also try this [example](https://github.com/docker/docker/pull/27841).

We’ve also recently contributed a logging driver for Docker. If you’d like to take a look, you can see some examples in the [pull request](https://github.com/docker/docker/pull/27841) and we would be interested in your feedback or opinion on if this is useful.

#### Minio
[Minio](http://www.minio.io) is a Go based Amazon S3-Compatible Object Storage Server that many of you will be familiar with. Like NATS, they also sponsored GopherCon and several other Golang community events this year. Much like NATS, Minio emphasizes simplicity and is a common choice for Golang developers. Minio recently added NATS as an event notification target, joining AMQP, Elasticseach, Redis, and PostgreSQL. You can make use of this using the events command in Minio, and more flags for this and how to use it are available via their [documentation](https://docs.minio.io/docs/minio-client-complete-guide).

#### Prometheus
[Prometheus](https://prometheus.io/) has become a very popular monitoring solution for cloud native applications, and chances are many of you reading this have tried it or are actively using it. There are several community developed integrations for exporting metrics from NATS to Prometheus available you may want to have a look at:

<https://github.com/lovoo/nats_exporter> 
<https://github.com/markuslindenberg/nats_exporter>
<https://github.com/SLASH2NL/nats-prometheus>


#### Micro
If you read the NATS.io blog regularly, you may have already seen a guest post by Asim Aslam on [Micro](https://micro.mu/) - a Go-based microservices framwork. It does an excellent job explaining what Micro is, and how Micro uses NATS - definitely worth checking out [here](https://blog.micro.mu/2016/04/11/micro-on-nats.html). 

If you were at GopherCon UK this year, you may also have seen the talk about Micro:

<blockquote class="twitter-tweet" data-lang="en"><p lang="en" dir="ltr">Here&#39;s me speaking about simplifying <a href="https://twitter.com/hashtag/microservices?src=hash">#microservices</a> with Micro at <a href="https://twitter.com/GolangUKconf">@GolangUKconf</a> last month. I just want to pin this! <a href="https://t.co/woInK3hhMC">https://t.co/woInK3hhMC</a></p>&mdash; Asim Aslam (@chuhnk) <a href="https://twitter.com/chuhnk/status/781886832564924416">September 30, 2016</a></blockquote>
<script async src="//platform.twitter.com/widgets.js" charset="utf-8"></script>

There is plenty more we could share about the good things happening in the Go/NATS Community - more than fit into a blog post so we had to wrap it up somewhere for this article... 

We’d like to sign off by saying *thank you* to all of you in the Go Community for your ongoing feedback, trying things out, and the rapid pace of innovation. We are excited about 2017 and where you all take Golang next - it’s an exciting time to be learning and working in Go!

If you’d like to get take a look at NATS or get involved in the NATS Community, you can:

Find us on [GitHub](https://github.com/nats-io)
Follow us on [Twitter](https://twitter.com/nats_io)
Join the [Google Group](https://groups.google.com/forum/#!forum/natsio)
Join our [Slack Community](https://docs.google.com/a/apcera.com/forms/d/104yA7oqq7SPoMDG_J9MnVE74gVwBnTmVHKP5ABHoM5k/viewform?embedded=true)
Check us out on [Reddit](https://www.reddit.com/r/nats_io)
