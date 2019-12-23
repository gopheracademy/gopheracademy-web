+++
title = "Go and Reliability"
date = "2019-12-26T00:00:00+00:00"
series = ["Advent 2019"]
author = ["Natalie Pistunovich"]
+++

[Last year](https://blog.gopheracademy.com/advent-2018/go-devops/) we discussed why Go is great in DevOps. 
We covered DevOps tools written in Go like Kubernetes, Docker, Prometheus and many others.
We also discsused properties of the language that are especially relevant to DevOps, like cross-compilation, dependencies being part of the binary and speed.

This year we'll zoom in on reliability of distributed systems.


### What Exactly is Reliability?
Reliability describes the ability of a system or component to function as expected for a specified period of time.

You probably heard about up time of nines. This describes the percent of the time that the system is functioning, e.g. [Five Nines](https://en.wikipedia.org/wiki/High_availability#Percentage_calculation) means that 99.999% of the time the system is up, or that the downtime is less than 5.26 minutes per year.
You probably also heard about SLx:

- Service-Level Objective (SLO): A precise numerical availability target for system

- Service-Level Agreement (SLA): An availability promise to a user of the system 

- Service-Level Indicator (SLI): A direct measurement of a service’s behavior: the frequency of successful probes of our system

Reliability vs. availability:
Reliability is the measure of how long a machine performs its intended function; availability is the measure of the time the system is up.


### Tooling Ecosystem
We've previously listed DevOps tools that are open source (OS) and written in Go, for example: Docker, Kubernetes, Helm, Prometheus, Grafana, Jaeger and Istio.


Here are some additional OS tools written in Go you can consider for SRE:

- [Loki](https://github.com/grafana/loki): a horizontally-scalable, highly-available, multi-tenant log aggregation system inspired by Prometheus. Or in short: Prometheus for logs

- [Terraform](https://github.com/hashicorp/terraform): codifies APIs into declarative configuration files that can be shared amongst team members, treated as code, edited, reviewed, and versioned

- [Nomad](https://github.com/hashicorp/nomad): workload orchestrator that deploys microservices, batch, containerized, and non-containerized applications

- [Logstash](https://github.com/elastic/logstash): transport and process your logs, events, etc

- [GolangCI-Lint](https://github.com/golangci/golangci-lint): a linter aggregator built in the CI process

- [Go Report Card](https://goreportcard.com/): will help you evaluate the OS library you are considering to add into the system

- [Still in alpha] [Gaia](https://github.com/gaia-pipeline/gaia): a plugin-based automation platform that uses gRPC to communicate over HTTP/2

And for additional recommendations try [this list of SRE tools](https://github.com/squadcastHQ/awesome-sre-tools).


### What Makes a System Distributed?

A distributed system is a group of machines working together to appear as a single device to the end user. 
In this system, the components are located on different networked computers, which communicate and coordinate their actions by passing messages to one another.

[Awesome Go](https://awesome-go.com/#distributed-systems) has a list of libraries written in Go that are distributed systems-related, from examples to toolkits like [go-kit](https://github.com/go-kit/kit).


### Testing Distributed Systems

Before suggesting my answer here, I'd like to point out that this question is a field of research on its own. One article will not be enough to cover this, but it can make a good start.

Your goal is to confirm the system behaves as expected during down time, getting unusual requests, and also confirm the resource allocation makes sense. The main difference between testing any system to testing a distributed system, is the additional focus on the topology of the components.



There are 2 ways to go about it: testing in production, or testing in a simulated environment.

#### Testing in Production

Game days is one great way to test in production. 
This method simulates failures or events to test systems, processes, and the response of the team to those.

In short, a red team is preparing an adversary plan. The red team can include different members that will bring different perspectives, from newly joined employees who are not familiar yet, to architects who know most of the systems in place.
On the side, the blue team is trying to recover as fast as possible, using the company's tools and processes that are in place.
Then there's a post-mortem from which a lot of conclusions and action items rise.
And this can take place periodically, there's always something new to break!


#### Testing in a Simulated Environment:

Testing simulated reliability of distributed systems requires:

1. A Replication of the Distributed System

The first step is to make sure the entire setup is exactly the same. As we discussed, this is a system made of multiple nodes. The system should have a setup similar to the production, both in size and in ratio.
This leaves the difference between staging and production to only be the traffic, mainly type and frequency.

2. Real Traffic

It should be similar to the one expected in production, that means:
- Unique requests/transactions. Each should be different in content and size, don't duplicate the same few requests multiple time. This would allow catching potential bottlenecks.
- Real-life frequency. Monotonous traffic is very uncommon in real life systems. Usually there are periods of high traffic, low traffic, spikes, and sometimes no traffic.

3. Tools to Break Things

Some good libraries to help you break things:

- [ChaosMonkey](https://github.com/Netflix/chaosmonkey): randomly terminates virtual machine instances and containers that run inside of your production environment

- [Rate limiting and throttrling](https://github.com/golang/go/wiki/RateLimiting) (also try searching [GoDoc](https://godoc.org/?q=rate+limit))

- More on [throttling (and benchmarking) in Go](https://www.youtube.com/watch?v=oE_vm7KeV_E) by [Daniel Martí](https://twitter.com/mvdan_?lang=en) 

- [Rate limiting access to HTTP Endpoints](https://godoc.org/github.com/throttled/throttled)

- [go-fuzz](https://github.com/dvyukov/go-fuzz): a coverage-guided fuzzing solution for testing of Go packages

- [failpoint](https://github.com/pingcap/failpoint): injecting failpoints (errors or abnormal behavior) into the code

- [Vegeta](https://github.com/tsenart/vegeta): HTTP load testing tool and library that drills HTTP services with a constant request rate

- [GoReplay](https://github.com/buger/goreplay): a network monitoring tool which can record your live traffic, and use it for shadowing, load testing, monitoring and detailed analysis

- Create race conditions by learning [all about the concurreny memoery access](https://www.ardanlabs.com/blog/2018/12/scheduling-in-go-part3.html)


There are many great guides out there that guide how to test new systems, and I would recommend reading several writeups for a wider perspective. This [curated list](https://github.com/asatarin/testing-distributed-systems) of has tools and articles, and it makes a fair summary for engineers.



To finish this section, a great tip, relevant to keeping systems reliable and beyond: [Always use the latest released version of Go](https://dave.cheney.net/high-performance-go-workshop/dotgo-paris.html#always_use_the_latest_released_version_of_go).


About the author:

[Natalie Pistunovich](https://twitter.com/nataliepis) is leading the [Berlin Go User Group](https://twitter.com/gdgberlingolang) since 2015; organizing [GopherCon Europe](https://gophercon.berlin), [Cloud Nein](https://cloudne.in) and Berlin [B-Sides](https://www.papercall.io/bsides-2020-berlin); and is advocating at scale for developers at [Aerospike](https://www.aerospike.com).