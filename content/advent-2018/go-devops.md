+++
author = ["Natalie Pistunovich"]
title = "Using Go in Devops"
linktitle = "Using Go in Devops"
date = 2018-12-24T00:00:00Z
series = ["Advent 2018"]
+++


== Using Go for DevOps ==

This post is aiming to provide a new angle on using Go. Don't expect code snippets or learning a new thing, but rather be open for a new perspective, and share this with your favorite SysOps/DevOps/Observability Engineers who are considering new solutions.

--

Not too long ago SysOps was a common job title, and that included updating softwares, setting up networks and adding glue bash scripts.
Then came DevOps - kind of a meeting point between the developer who learns to deploy and monitor their code, and the ops person whose code is now more sophisticated.
And now, with systems becoming distributed and more complex, spread over a group of services (or microservices) - Observability becoming a trade that helps keep you on track with the system's health.


### Define Observability

Great people had, and still have, this discussion going on. I am learning a lot from following it and excited to take part in a tech branches that is developing so rapidly. Some of you might not agree with this definition, and maybe looking at this from the future I'll think differently.

Monitoring refers to repeatedly checking a system and its outputs to make sure they are within known-good ranges. It’s the operational version of integration tests and is heavily biased toward outages and problems with tools like alerts and logs.

In control theory, observability is a measure of how well internal states of a system can be inferred from knowledge of its external outputs. The observability and controllability of a system are mathematical duals. 

But Observability is about being able to understand the inner workings of your software and systems, asking questions, and observing the answers by looking on the outside. ANY question — no particular bias toward actionable alerts or problems. As the conversation is moving to observability from monitoring, we're also disucssung more about unknown-unknowns rather than the known-unknowns.

Many of the modern day tooling, for ops and for observability, are written in Go.


#### How Will This Shift Translate in Code?

It all started with just adding log lines into your code. And this is definitely the mother of all debug tools out there, and it is here to stay.

Another thing we all do is One liners, for example in bash. They are also here to stay.

But sometimes you find yourself saying "I will just write a small throw away script for that”. But those end up staying too.
And as systems become more complex, our work becomes more complex, and so do these script. And they multiply. And become a cluster of legacy scripts.
These - they can be made better, by using a coding language rather than scripting language, to write a piece of code rather than a set of scripts.

Just like you can’t imagine a restaurant chef using a regular knife, maintaining multiple serves in multiple environments in a complex architecture with a high uptime and low response time - require a proper solution, that is easy to maintain. Go provides many of these benefits, and we will go over that in detail.

## Tooling

All the tools listed here are open source too:

### Ops Tools (Re)Written in Go

#### [Kubernetes](https://kubernetes.io/)
Borg was created in Google in 2003 as a cluster manager with the purpose to efficiently schedule runs hundreds of thousands of jobs and makes computing much more efficient. It was written in C++. 

In 2014 Kubernetes was released as the OS equivalent: a container orchestrator used in production by large companies to run distributed services. 

Kubernetes is a system for automating deployment, scaling, and management of containerized applications. It groups containers, that make up an application into logical units, for easy management and discovery. Kubernetes can scale and add infinite amount of resources without increasing the ops team. It runs on on-premises, hybrid, or public cloud infrastructure. 

#### [Prometheus](https://prometheus.io/)
Borg Mon was also develop in 2003 as an internal monitoring system complimentary for Borg, and it was also written in C++.

In 2012 Prometheus was incepted as an open source equivalent: a time-series data as a data source, for collecting metrics and generating alerts.

The time series are identified by a metric name and a set of key-value pairs, which makes powerful queries and efficient storage, built-in alerting, it's supporting 10 languages and bridges data import from sources like Docker and StatsD.

#### [Etcd](https://coreos.com/etcd/)
Chubby was created in 2006 as a distributed lock manager. 
Fun fact: the SLO was 99.99 (4 nines) and to make sure it is met - the team often had to carry out a [planned outage](https://landing.google.com/sre/sre-book/chapters/service-level-objectives/#xref_risk-management_global-chubby-planned-outage) for up to 13 min per quarter!

In 2013 etcd was developed: a strongly consistent distributed key value store, into which you can read and write data, that provides a reliable way to store data across a cluster of machines. 

Etcd is a central component of Kubernetes and it’s used in mission critical distributed systems, because it gracefully handles leader elections during network partitions and tolerates machine failure, including the leader. A simple use-case is to store db connection details in etcd as key value pairs. These values can be watched, allowing your app to reconfigure itself when they change. Now also a CNCF project!

#### [Docker](https://www.docker.com/)
In 2006 Google introduced cgroups, allowing you take a process, isolate it, and limit its resources.
In 2008 came lxc, with the additional functionality of kernel module. This allowed name spacing and limiting what a process can see.

In 2013 Docker was released: it can do all the above, and additionally pipeline and tooling to build the images that spawn to the containers, and control map ports.

Docker is performing containerization - operating-system-level virtualization. It is the most popular container runtime on the market right now and a central component of Kubernetes.


### Ops Tools Written in Go

#### [Helm](https://helm.sh/)
Helm is the package manager for Kubernetes. It is the best way to find, share, and use software built for Kubernetes. Helm Charts help define, install, and upgrade complex Kubernetes application; version control, share, and publish config files instead of getting lost in copy-pasta.

#### [Grafana](https://opentracing.io/)
Grafana is a time series analytics and monitoring and visulatization platform.
Some of the officially supported datasources are: Prometheus, InfluxDB and Elasticsearch. Each Data Source has a query editor customized for its features and capabilities. Each Data Source is tied to a panel, and all the panels together make a dashboard where the time period can be controlled, and the annotations display event data across the panels for correlating the time series data with other events.

#### [The Open Tracing Project](https://opentracing.io/)
Distributed tracing anyone?
Used for profiling and monitoring applications on distributed software architectures, such as microservices, distributed tracing helps debugging complex systems by pinpointing where failures occurs and optimizing by spotting what causes poor performance.

The OpenTracing API Project is working towards creating more standardized APIs and instrumentation, and it's comprised of an API specification, frameworks and libraries that have implemented the specification, and documentation for the project. It allows developers to add instrumentation to the code using APIs that do not lock them into any one particular product or vendor.

#### [Jaeger](https://www.jaegertracing.io/)
Open Tracing compatible, Jaeger is an end-to-end distributed tracing for monitoring and troubleshooting transactions in complex distributed systems that many time ground in networking or observability. 

Jaeger is a tool for distributed transaction monitoring and context propagation, performance and latency optimization, root cause analysis and service dependency analysis.

#### [Istio](https://istio.io/)
A service mesh that is a configurable infrastructure layer, for a microservices architecture. Istio makes communication between service instances flexible, reliable, fast and secure. It’s added to the application by deploying a sidecar (a proxy) next to each service, which means no code changes are needed!

This allows automatic discovery of devices and services, tracing, monitoring, logging, observability into the platform, security by means like access control and authentication, access policy control and traffic control.

#### [CNCF](https://www.cncf.io/)
The Cloud Native Computing Foundation is a home to many of the above projects. It is an open source software foundation dedicated to making cloud native computing universal and sustainable. The foundation builds sustainable ecosystems and fosters a community around a constellation of high-quality open source projects that orchestrate containers as part of a microservices architecture.    

Cloud native computing uses an open source software stack to deploy applications as microservices packaging each part into its own container, and dynamically orchestrating those containers to optimize resource utilization.


## Go Benefits for SRE

#### Simple, Reliable, Fast
With Go - whatever you’re building, you focus on building it, rather than the tools you need to run it.

### Simple

#### Readability
Remember we discussed earlier the need for a solid piece of software to replace the group of bash scripts? That software is easily readable and looks the same everywhere, even if several ops teams share the codebase. This readability is easy to achieve with `gofmt`, the built-in linter.

#### Built-in Testing, Profiling and Benchmarking
Following the best practice of Go, you probably will be doing TDD (Test-Driven Development). There's no need for assertion, but it’s there if you want it. 
Testing, profiling (CPU and memory) and benchmarking are all built-in, so there's no need for learning a micro language, new commands or use new tools, and there are some nice tools for results visualisations.

#### One Binary to Rule Them All
Go is statically linked, meaning there's no need for external libraries, copy dependencies or worry for imports. All the code and its dependencies are in the binary, so that's all you need to distribute. And as a purely homogenous environment it's not dependent on language versions and releases.

#### Cross Compilation
Having everything in the binary makes things simple, being able to cross compile it makes things simple even in organizations where everyone have their own setup: in order for a binary to be supported on the different operating systems all it takes is setting the 2 environment variables: $GOOS, $GOARCH. No need for a virtual environment, a package manager or managing dependencies. This is a great feature for CLIs, and indeed some of the most popular ones are using it: etcdctl, kubectl and docker.

Here's a partial list:

|  $GOOS  | $GOARCH |       OS       |
|---------|---------|----------------|
| darwin  | 386     | 32 bit MacOSX  |
| darwin  | amd64   | 64 bit MacOSX  |
| linux   | 386     | 32 bit Linux   |
| linux   | amd64   | 64 bit Linux   |
| linux   | arm     | RISC Linux     |
| windows | 386     | 32 bit Windows |
| windows | amd64   | 64 bit Windows |

#### Composition, Not Inheritance
Staying out of inheritance confusion.

#### The Standard Librart
Many of the packages in the standard library are the building blocks of the Ops toolbox like handling web services with different protocols like HTTP/HTTP2,
and file processing: path, download, open, process, time, json, regex, etc. No need for keeping track what is the currently standard package or external software for the bsaic operations, or switching if it gets deprecated. As these are all in the standard library - they are all lofficially maintained, well documented and in use across all the developers. 
While the std lib is not big but is wisely composed of packages that proof useful for an OPS/SRE person

### Reliable

#### Pointers Exist, Pointer Arithmetic Doesn’t
Stayin safe!

#### Error Handling
Planning clearly for errors and acting upon them as values, rather than having error exceptions, makes the execution smoother.as values.

#### Open Source
Go is backed by an amazing community, companies who are using the language for years now, and by industry giants like Google, Microsoft, Apple and more (did you know that SpaceX is using Go?). So by now, the language is here to stay.

#### Data Types
Go is Type safe and strongly typed, which means that string operations on an int cannot happen, because it will be caight by the compiler.
The added bonus of slice being a memory efficient abstraction built on top of the array type is making some of the operations significantly faster.

### Fast

#### Fast Compilation and Execution 
As the compiler will fail the run if there are unused imports, the compilation time is short and binary size is small. As the code is compiled to machine code - it will also execute fast. Think of running millions of inputs through sed or a bash loop, how much faster will it be in Go?

#### Garbage Collection
As in many languages, the garbace collector is  a controversial topic.
In short - there are default values, and they can be changed for optimal performance.
In more detail - there are some great articles

#### Import-Defined Dependencies 
All the dependencies are included in the binary, which saves any extra steps for carrying the dependencies alongside the binary.

#### Fast Performing
Many benchmarks were done. [Here's](https://stackimpact.com/blog/practical-golang-benchmarks/) a rather comprehensive list. It's just really fast!


## To Summarize
Go is a great language for building fast and reliable web services. Many times it also makes a great fit for work needed to make sure these services are working as they should. It's of course important to choose the right tool for the task, so apply critical thinking and consider using Go for more than web development, and use more of the (few but growing) resource pool of using Go for DevOps. 

Happy holidays!
@NataliePis
