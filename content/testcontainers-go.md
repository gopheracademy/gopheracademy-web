+++
author = ["Gianluca Arbezzano"]
date = "2019-01-19T15:07:34+09:00"
linktitle = "Integration test made easy with testcontainers"
title = "Integration test made easy with testcontainers"
+++

There are a lot of information in the title I know, but I am not good enough to
make it simple.

Back in the days, I tried to make some contribution to [OpenZipkin](https://github.com/openzipkin/zipkin) an open
source tracing infrastructure in Java. I never really worked in that language, and apparently, I failed, but it wasn't all a waste of time.

OpenZipkin has an excellent integration test suite, and I liked the approach it took to write
integration tests for all the backends it supports MySql, ElasticSeach,
Cassandra.

Provision the integration test environment is complicated even when you do it
wrong:

1. Without a per test isolation.
2. Without a cleanup process.
3. Without putting the right effort to have isolated tests.

If you try to make integration tests in the right way, you will have a very hard
time, but Zipkin uses a project called
[testcontainers-java](https://github.com/testcontainers/testcontainers-java). It is a library
that wraps the Docker SDK to offer a friendly API to write integration
tests using containers.

## Why containers

In 2019 everyone knows the answer, containers are great for integration testings
because they are a lightweight and flexible technology. Docker provides the
architecture that simplifies how you can turn them on and off.

You can spin up a bunch of containers for every integration tests, they will be
fresh new, and you can terminate them at the end of the tests. This increases
isolation a lot, and it makes your tests more stable and easy to reproduce.

## Golang

I develop in Go every day I loved the approach, so I decided to port that
library to Golang and it eventually get moved to the
[testcontainers](https://github.com/testcontainers) GitHub
organization under the repository [testcontainers/testcontainers-go](https://github.com/testcontainers/testcontainers-go).

There is a lot to do but I think at this point the API is stable and we have
everything we need to use it. All the rest will be driven by yourself asking for
new features or from contributors that will port more things from the java
project.


This is our "Hello World."

```golang
package main

import (
    "context"
    "fmt"
    "net/http"
    "testing"

    testcontainers "github.com/testcontainers/testcontainers-go"
)

// TestNginxLatestReturn verifies that a requesto to root returns 200 as status
// code
func TestNginxLatestReturn(t *testing.T) {
    ctx := context.Background()
    // Request an nginx container that exposes port 80
    req := testcontainers.ContainerRequest{
        Image:        "nginx",
        ExposedPorts: []string{"80/tcp"},
    }
    nginxC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    if err != nil {
        t.Error(err)
    }
    // At the end of the test remove the container
    defer nginxC.Terminate(ctx)
    // Retrieve the container IP
    ip, err := nginxC.Host(ctx)
    if err != nil {
        t.Error(err)
    }
    // Retrieve the port mapped to port 80
    port, err := nginxC.MappedPort(ctx, "80")
    if err != nil {
        t.Error(err)
    }
    resp, err := http.Get(fmt.Sprintf("http://%s:%s", ip, port.Port()))
    if resp.StatusCode != http.StatusOK {
        t.Errorf("Expected status code %d. Got %d.", http.StatusOK, resp.StatusCode)
    }
}
```

This is a straightforward test, but you can imagine a lot of other use cases. Let's say that you need to test how your `application A` interact with an `application B` that
depends on Redis. You can programmatically build the environment you need in the tests:

```golang
// You spin up the Redis container
req := testcontainers.ContainerRequest{
    Image:        "redis",
    ExposedPorts: []string{"6379/tcp"},
}
redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
    ContainerRequest: req,
    Started:          true,
})
if err != nil {
    t.Error(err)
}
defer redisC.Terminate(ctx)
ip, err := redisC.Host(ctx)
if err != nil {
    t.Error(err)
}
redisPort, err := redisC.MappedPort(ctx, "6479/tcp")
if err != nil {
    t.Error(err)
}

// Spin up Application B
appB, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
    ContainerRequest: req,
    Started:          true,
    Env: map[string]string{
        "REDIS_HOST": fmt.Sprintf("http://%s:%s", ip, redisPort.Port()),
    },
})
if err != nil {
    t.Error(err)
}
ipB, err := redisC.Host(ctx)
if err != nil {
    t.Error(err)
}
portB, err := redisC.MappedPort(ctx, "8081/tcp")
if err != nil {
    t.Error(err)
}

defer appB.Terminate(ctx)
defer redis.Terminate(ctx)

// Now you can use the go function from your application A that interact with
// application B
bclient := appA.NewServiceBClient(ipB, portB)
content, err := bclient.GetKey("my-key")

// Check what you need to check
```

## Programmable environment is the key

I wrote about my relationship with [infrastructure as
code](/blog/infrastructure-as-real-code) in a previous article but once again
the fact that you can programmatically build your infrastructure
using real code is the key for all this flexibility.

As plus for integration tests, you can build the environment you need from inside the test case itself, this ability provides significant control over it.

If you need to worm up etcd with some data, you spin up the etcd container and
you push your data using the traditional Go [etcd client](https://github.com/etcd-io/etcd/tree/master/client):

```go
// Spin up Etcd
req := testcontainers.ContainerRequest{
    Image:        "quay.io/coreos/etcd:latest",
    ExposedPorts: []string{"2379/tcp"},
}
etcdC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
    ContainerRequest: req,
    Started:          true,
})
if err != nil {
    t.Error(err)
}
defer etcdC.Terminate(ctx)
ip, err := etcdC.Host(ctx)
if err != nil {
    t.Error(err)
}
etcdPort, err := redisC.MappedPort(ctx, "2379/tcp")
if err != nil {
    t.Error(err)
}

// Configure the etcd client
cfg := client.Config{
    Endpoints:               []string{"http://" + ip + ":" + etcdPort},
    Transport:               client.DefaultTransport,
    // set timeout per request to fail fast when the target endpoint is unavailable
    HeaderTimeoutPerRequest: time.Second,
}
c, err := client.New(cfg)
if err != nil {
    log.Fatal(err)
}
kapi := client.NewKeysAPI(c)

// Set the key foo
resp, err := kapi.Set(context.Background(), "/foo", "bar", nil)
```

I wrote this article because after a few weeks of coding and revisions I have
finally tagged
[`v0.0.1`](https://github.com/testcontainers/testcontainers-go/releases/tag/0.0.1)
and the library is ready to be tried we need feedback and feature requests to
Prioritize the work to do. So feel free to try it and to open GitHub
[issues](https://github.com/testcontainers/testcontainer-go/issues).
