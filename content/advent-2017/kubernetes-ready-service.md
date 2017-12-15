+++
author = [ "Elena Grahovac" ]
date = "2017-12-14T00:00:00"
linktitle = "Write a Kubernetes-ready service from zero step-by-step"
title = "Write a Kubernetes-ready service from zero step-by-step"
series = ["Advent 2017"]

+++

If you have ever tried Go, you probably know that writing services with Go is an easy thing. Yes, we really need [only few lines](https://github.com/rumyantseva/advent-2017/commit/76864ab0587dd9a599752ed090f618749b6bfe0c) to be able to run http service. But what do we need to add if we want to prepare our service for production? Let’s discuss it by an example of a service which is ready to be run in [Kubernetes](http://kubernetes.io).

You can find all examples from this article [in the single tag](https://github.com/rumyantseva/advent-2017/tree/all-steps) and you can follow this article [commit-by-commit](https://github.com/rumyantseva/advent-2017/commits/master).

### Step 1. The simplest service

So, we have a very simple application here:

`main.go`
```go
package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/home", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "Hello! Your request was processed.")
	},
	)
	http.ListenAndServe(":8000", nil)
}
```

If we want to try it for the first time, `go run main.go` might be enough. If we want to see how it works, we may use a simple curl command: `curl -i http://127.0.0.1:8000/home`. But when we run this application, we see that there is not any information about its state in the terminal.

### Step 2. Add a logger

First of all, let's add a logger to be able to understand what is going on and to be able to log errors and other important situations. In this example we will use the simplest logger from the standard Go library, but for a production-ready service you might be intersted in more complicated solutions such as [glog](https://github.com/golang/glog) or [logrus](https://github.com/sirupsen/logrus).

For example, we might want to log 3 situations: when the service is starting, when the service is ready to handle requests and if `http.ListenAndServe` returns an error. As the result we will have something like [this](https://github.com/rumyantseva/advent-2017/commit/65689d282be7e6a7ec38ec254605811fd3bef784):

`main.go`
```go
func main() {
	log.Print("Starting the service...")

	http.HandleFunc("/home", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "Hello! Your request was processed.")
	},
	)

	log.Print("The service is ready to listen and serve.")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
```

Looks better!

### Step 3. Add a router

Now, if we write a real application, we might want to add a router to be able to handle different URIs and HTTP methods and match other rules in an easy way. There is not any router in the standard Go library, so let's use [gorilla/mux](https://github.com/gorilla/mux) which is pretty compatible with the standard `net/http` library.

If your service needs some significant amount of different routing rules, it makes sense to move all routing-related things to separate functions or even a package. Let's move router initialization and rules to the package `handlers` (see the full change [here](https://github.com/rumyantseva/advent-2017/commit/1a61e7952e227e33eaab81404d7bff9278244080)).

Let's add `Router` function which returns a configured router and `home` function  which handles `/home` path. Personally, I prefer to use separated files for such things:

`handlers/handlers.go`
```go
package handlers

import (
	"github.com/gorilla/mux"
)

// Router register necessary routes and returns an instance of a router.
func Router() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/home", home).Methods("GET")
	return r
}
```

`handlers/home.go`
```go
package handlers

import (
	"fmt"
	"net/http"
)

// home is a simple HTTP handler function which writes a response.
func home(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprint(w, "Hello! Your request was processed.")
}
```

And then we need some small changes in `main.go` file:
```go
package main

import (
	"log"
	"net/http"

	"github.com/rumyantseva/advent-2017/handlers"
)

// How to try it: go run main.go
func main() {
	log.Print("Starting the service...")
	router := handlers.Router()
	log.Print("The service is ready to listen and serve.")
	log.Fatal(http.ListenAndServe(":8000", router))
}
```

### Step 4. Tests

It is time to [add some tests](https://github.com/rumyantseva/advent-2017/commit/a3e7f6356478095c41166ade41feba6917b37096). Let's use `httptest` package for it. For the `Router` function we might add something like this:

`handlers/handlers_test.go`:
```go
package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouter(t *testing.T) {
	r := Router()
	ts := httptest.NewServer(r)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/home")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("Status code for /home is wrong. Have: %d, want: %d.", res.StatusCode, http.StatusOK)
	}

	res, err = http.Post(ts.URL+"/home", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Status code for /home is wrong. Have: %d, want: %d.", res.StatusCode, http.StatusMethodNotAllowed)
	}

	res, err = http.Get(ts.URL + "/not-exists")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("Status code for /home is wrong. Have: %d, want: %d.", res.StatusCode, http.StatusNotFound)
	}
}
```

Here we check if `GET` method for `/home` returns code `200`. On the other hand, if we try to send `POST` we expect `405`. And, finally, for a route which does not exists we expect `404`. Actually, this example might be a bit "verbose" because the router is already well-tested as a part of `gorilla/mux` package, so you might want to check even less things.

For `home` it might make sense to check its response code and response body:

`handlers/home_test.go`:
```go
package handlers

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHome(t *testing.T) {
	w := httptest.NewRecorder()
	home(w, nil)

	resp := w.Result()
	if have, want := resp.StatusCode, http.StatusOK; have != want {
		t.Errorf("Status code is wrong. Have: %d, want: %d.", have, want)
	}

	greeting, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if have, want := string(greeting), "Hello! Your request was processed."; have != want {
		t.Errorf("The greeting is wrong. Have: %s, want: %s.", have, want)
	}
}
```

Let's run `go test` to check if our tests work:

```bash
$ go test -v ./...
?       github.com/rumyantseva/advent-2017      [no test files]
=== RUN   TestRouter
--- PASS: TestRouter (0.00s)
=== RUN   TestHome
--- PASS: TestHome (0.00s)
PASS
ok      github.com/rumyantseva/advent-2017/handlers     0.018s
```

### Step 5. Configuration

Next important question is ability to configure our service. Right now it always listens on the port `8000`, and probably it might be useful to be able to configure this value. [The Twelve-Factor App manifesto](https://12factor.net), which represents a really great approach for writing services, tells us that it is good to store configuration based on the environment. So, [let's use environment variables for it](https://github.com/rumyantseva/advent-2017/commit/a7446f5db919ed0ecd3b4f966ed9b4a399e68210):

`main.go`
```go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/rumyantseva/advent-2017/handlers"
)

// How to try it: PORT=8000 go run main.go
func main() {
	log.Print("Starting the service...")

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("Port is not set.")
	}

	r := handlers.Router()
	log.Print("The service is ready to listen and serve.")
	log.Fatal(http.ListenAndServe(":"+port, r))
}
```

In this example, if the port is not set, the application will simply exit with an error. There is no sense to try continue working, if the configuration is wrong.

### Step 6. Makefile

Few days ago there was [an article](https://blog.gopheracademy.com/advent-2017/make) about the `make` tool, which is very helpful if you want to automate some repeatable routines. Let's see how we can use it for our application. Currently, we have two actions: to run the tests, to compile and run the service. Let's [add these action to a Makefile](https://github.com/rumyantseva/advent-2017/commit/90966780ba6656f8dc0aebd166938c9adcbe0514). But instead of simple `go run` we will use `go build` and we will run a compiled binary, because this approach suits to our production-readiness goals better:

`Makefile`
```make
APP?=advent
PORT?=8000

clean:
	rm -f ${APP}

build: clean
	go build -o ${APP}

run: build
	PORT=${PORT} ./${APP}

test:
	go test -v -race ./...
```

In this example we moved a binary name to a separated variable `APP` to not to repeat the name few times.

Here, if we want to run an application, we need to delete an old binary (if it exists), to compile the code and to run a new binary with the right environment variable and to do all these things we can use `make run`.

### Step 7. Versioning

The next technique we will add to our service is versioning. Sometimes it might be very useful to know what are the exact build and commit we use in production and when the binary was built.

To be able to store this information let's add a new package - `version`:

`version/version.go`
```go
package version

var (
	// BuildTime is a time label of the moment when the binary was built
	BuildTime = "unset"
	// Commit is a last commit hash at the moment when the binary was built
	Commit = "unset"
	// Release is a semantic version of current build
	Release = "unset"
)
```

We can log these variables when the application starts:

`main.go`
```go
...
func main() {
	log.Printf(
		"Starting the service...\ncommit: %s, build time: %s, release: %s",
		version.Commit, version.BuildTime, version.Release,
	)
...
}
```

And we also may add them to the `home` handler (don't forget to change the test!):

`handlers/home.go`
```go
package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/rumyantseva/advent-2017/version"
)

// home is a simple HTTP handler function which writes a response.
func home(w http.ResponseWriter, _ *http.Request) {
	info := struct {
		BuildTime string `json:"buildTime"`
		Commit    string `json:"commit"`
		Release   string `json:"release"`
	}{
		version.BuildTime, version.Commit, version.Release,
	}

	body, err := json.Marshal(info)
	if err != nil {
		log.Printf("Could not encode info data: %v", err)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}
```

We will use the Go linker to set `BuildTime`, `Commit` and `Release` variables during compilation.

Let's add the new variables to the `Makefile`:

`Makefile`
```make
RELEASE?=0.0.1
COMMIT?=$(shell git rev-parse --short HEAD)
BUILD_TIME?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
```

Here `COMMIT` and `RELEASE` are defined by provided commands and we can use  [semantic versions](https://dave.cheney.net/2016/06/24/gophers-please-tag-your-releases) for `RELEASE`.

Now let's rewrite the `build` target to be able to use these variables:

`Makefile`
```make
build: clean
	go build \
		-ldflags "-s -w -X ${PROJECT}/version.Release=${RELEASE} \
		-X ${PROJECT}/version.Commit=${COMMIT} -X ${PROJECT}/version.BuildTime=${BUILD_TIME}" \
		-o ${APP}
```

I also defined the `PROJECT` variable in the beginning of `Makefile` to not to repeat the same things few times:

`Makefile`
```make
PROJECT?=github.com/rumyantseva/advent-2017
```

All changes we made during this step you can find [here](https://github.com/rumyantseva/advent-2017/commit/eaa4ff224b32fb343f5eac2a1204cc3806a22efd). Feel free to try `make run` and check how it works. 

### Step 8. Let's have less dependencies

There is one thing I do not like in our code: the `handler` package depends on the `version` package. It is easy to change it: we need to make the `home` handler configurable:

`handlers/home.go`
```go
// home returns a simple HTTP handler function which writes a response.
func home(buildTime, commit, release string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		...
	}
}
```

And again, do not forget to fix the tests and provide [all necessary changes](https://github.com/rumyantseva/advent-2017/commit/e73b996f8522b736c150e53db059cf041c7c3e64).

### Step 9. Health checks

In a case if we want to run a service in Kubernetes, we usually need to add the health checks: [liveness and readiness probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/). The purpose of a liveness probe is to understand that the application is running. If the liveness probe fails, the service will be restarted. The purpose of a readiness probe is to understand if the application is ready to serve traffic. If the readiness probe fails, the container will be removed from service load balancers.

To define the readiness probe we usually need to write a simple handler which always return response code `200`:

`handlers/healthz.go`
```go
// healthz is a liveness probe.
func healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
```

For the readiness probe it is often similar, but sometimes we might need to wait for some event (e.g. the database is ready) to be able to serve traffic:

`handlers/readyz.go`
```go
// readyz is a readiness probe.
func readyz(isReady *atomic.Value) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if isReady == nil || !isReady.Load().(bool) {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
```

In this example we return `200` only if the variable `isReady` is set and equals to `true`.

Let's see how we may use it:

`handlers.go`
```go
func Router(buildTime, commit, release string) *mux.Router {
	isReady := &atomic.Value{}
	isReady.Store(false)
	go func() {
		log.Printf("Readyz probe is negative by default...")
		time.Sleep(10 * time.Second)
		isReady.Store(true)
		log.Printf("Readyz probe is positive.")
	}()

	r := mux.NewRouter()
	r.HandleFunc("/home", home(buildTime, commit, release)).Methods("GET")
	r.HandleFunc("/healthz", healthz)
	r.HandleFunc("/readyz", readyz(isReady))
	return r
}
```

Here we want to mark that the application is ready to serve traffic after 10 seconds. Of course, in the real life there is not any sense to wait for 10 seconds, but you might want to add here cache warming (if your application uses cache) or something like this. 

As usual, the whole changes we made of this step you can find [on Github](https://github.com/rumyantseva/advent-2017/commit/e73b996f8522b736c150e53db059cf041c7c3e64).

**Note.** *If your application hits too much traffic, its endpoints will response unstable. E.g. liveness probe might be failed because of timeouts. This is why some engineers prefer to not to use liveness probe at all. Personally, I think that it would be better to scale resources if you find out that you have more and more requests. For example, you might want to [scale pods with HPA](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/).*

### Step 10. Graceful shutdown

When the service needs to be stoped, it is good to not to interrupt connections, requests and other operations immediately, but to handle all those things properly. Go supports graceful shutdown for `http.Server` since version 1.8. Let's see [how we may use it](https://github.com/rumyantseva/advent-2017/commit/93f8357d5f2a8fb0c978e5256d400dd00a393575):

`main.go`
```go
func main() {
    ...
	r := handlers.Router(version.BuildTime, version.Commit, version.Release)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}
	go func() {
		log.Fatal(srv.ListenAndServe())
	}()
	log.Print("The service is ready to listen and serve.")

	killSignal := <-interrupt
	switch killSignal {
	case os.Interrupt:
		log.Print("Got SIGINT...")
	case syscall.SIGTERM:
		log.Print("Got SIGTERM...")
	}

	log.Print("The service is shutting down...")
	srv.Shutdown(context.Background())
	log.Print("Done")
}
```

In this example we are able to catch operation system signals and if one of `SIGINT` or `SIGTERM` is catched, we will shut down the service gracefully.

**Note.** *When I was writing this code, I tried to catch `SIGKILL` here. I saw it few times in different libraries and I was sure that it worked. But, as Sandor Szücs [noted](https://twitter.com/sszuecs/status/941582509565005824), it is not possible to catch `SIGKILL`. In the case of `SIGKILL`, the application will be stoped immediately.*

### Step 11. Dockerfile

Our application is almost ready to be run in Kubernetes. Now we need to dockerize it.

The simplest `Dockerfile`, we need to define here, might look like this:

`Dockerfile`
```docker
FROM scratch

ENV PORT 8000
EXPOSE $PORT

COPY advent /
CMD ["/advent"]
```

We create the smallest container, copy the binary there and run it (we also do not forget about `PORT` configuration variable).

Let's change a bit the `Makefile` to be able to build an image and run a container. Here it might be useful to define new variables: `GOOS` and `GOARCH` which we will use for cross-compilation in the `build` goal. 

`Makefile`
```make
...

GOOS?=linux
GOARCH?=amd64

...

build: clean
	CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} go build \
		-ldflags "-s -w -X ${PROJECT}/version.Release=${RELEASE} \
		-X ${PROJECT}/version.Commit=${COMMIT} -X ${PROJECT}/version.BuildTime=${BUILD_TIME}" \
		-o ${APP}

container: build
	docker build -t $(APP):$(RELEASE) .

run: container
	docker stop $(APP):$(RELEASE) || true && docker rm $(APP):$(RELEASE) || true
	docker run --name ${APP} -p ${PORT}:${PORT} --rm \
		-e "PORT=${PORT}" \
		$(APP):$(RELEASE)

...
```

We also added the `container` goal to be able to build an image and the `run` goal to run our application from the container. All changes are available [here](https://github.com/rumyantseva/advent-2017/commit/909fef6d585c85c5e16b5b0e4fdbdf080893b679).

Now let's try `make run` to check the whole process.

### Step 12. Vendoring

We have an external dependency (`github.com/gorilla/mux`) in our project. And it means that for production readiness we definetely need to [add dependency management here](https://github.com/rumyantseva/advent-2017/commit/7ffa56a78400367e5d633521dee816b767d7d05d). If we use [dep](https://github.com/golang/dep) the only thing which we need for our service is `dep init`:

```bash
$ dep init
  Using ^1.6.0 as constraint for direct dep github.com/gorilla/mux
  Locking in v1.6.0 (7f08801) for direct dep github.com/gorilla/mux
  Locking in v1.1 (1ea2538) for transitive dep github.com/gorilla/context
```

It created `Gopkg.toml` and `Gopkg.lock` files and `vendor` directory. Personally, I prefer to push `vendor` to git, especially for important projects.

### Step 13. Kubernetes

[The last step](https://github.com/rumyantseva/advent-2017/commit/27b256191dc8d4530c895091c49b8a9293932e0f). Let's run our application in Kubernetes. The simplest way to run it locally is to install and configure [minikube](https://github.com/kubernetes/minikube).

Kubernetes pulls images from a Docker registry. In our case, we will work with the public Docker registry - [Docker Hub](https://hub.docker.com). We need to add one more variable and one more command to the `Makefile`:

`Makefile`
```make
CONTAINER_IMAGE?=docker.io/webdeva/${APP}

...

container: build
	docker build -t $(CONTAINER_IMAGE):$(RELEASE) .

...

push: container
	docker push $(CONTAINER_IMAGE):$(RELEASE)
```

The `CONTAINER_IMAGE` variable defines a Docker registry repo which we will use to push and pull our service images. As you can see, in this case it includes the username (`webdeva`). If you do not have an account at [hub.docker.com](https://hub.docker.com) yet, please create it and login with `docker login` command. After this, you will be able to push images.

Let's try `make push`:

```bash
$ make push
...
docker build -t docker.io/webdeva/advent:0.0.1 .
Sending build context to Docker daemon   5.25MB
...
Successfully built d3cc8f4121fe
Successfully tagged webdeva/advent:0.0.1
docker push docker.io/webdeva/advent:0.0.1
The push refers to a repository [docker.io/webdeva/advent]
ee1f0f98199f: Pushed 
0.0.1: digest: sha256:fb3a25b19946787e291f32f45931ffd95a933100c7e55ab975e523a02810b04c size: 528
```

It works! Now you can find [the image in the registry](https://hub.docker.com/r/webdeva/advent/tags/).

Let's define the necessary Kubernetes configuration (manifest). Usually, for the simplest service we need to set at least deployment, service and ingress configurations. By default the manifests are static, it means that you are not able to use any variables there. Hopefully, you can use [helm](https://github.com/kubernetes/helm) to be able to create flexible configuration. 

In this example we will not use `helm`, but it might be useful to define a couple of variables: `ServiceName` and `Release`, it gives us more flexibility. Later, we will use the `sed` command to be able to replace these "variable" with the real values.

Let's look at deployment configuration:

`deployment.yaml`
```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: {{ .ServiceName }}
  labels:
    app: {{ .ServiceName }}
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 50%
      maxSurge: 1
  template:
    metadata:
      labels:
        app: {{ .ServiceName }}
    spec:
      containers:
      - name: {{ .ServiceName }}
        image: docker.io/webdeva/{{ .ServiceName }}:{{ .Release }}
        imagePullPolicy: Always
        ports:
        - containerPort: 8000
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8000
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8000
        resources:
          limits:
            cpu: 10m
            memory: 30Mi
          requests:
            cpu: 10m
            memory: 30Mi
      terminationGracePeriodSeconds: 30
```

It is better to discuss Kubernetes configuration as a part a separate article, but, as you can see, here, among other things, we defined where it is possible to find a container image and how to reach liveness and readiness probes.

A typical service looks simpler:

`service.yaml`
```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ .ServiceName }}
  labels:
    app: {{ .ServiceName }}
spec:
  ports:
  - port: 80
    targetPort: 8000
    protocol: TCP
    name: http
  selector:
    app: {{ .ServiceName }}

```

And, finally, ingress. Here we define the rules to access a service from outside of Kubernetes. Assume, that we want to "attach" our service to the domain `advent.test` (which is actualy fake):

`ingress.yaml`
```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: nginx
    ingress.kubernetes.io/rewrite-target: /
  labels:
    app: {{ .ServiceName }}
  name: {{ .ServiceName }}
spec:
  backend:
    serviceName: {{ .ServiceName }}
    servicePort: 80
  rules:
  - host: advent.test
    http:
      paths:
      - path: /
        backend:
          serviceName: {{ .ServiceName }}
          servicePort: 80
```

Now to check how it works we need to install and run `minikube`, its official documentation is [here](https://github.com/kubernetes/minikube#installation). We also need the [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) tool to be able to apply the configuration and check the service.

To start `minikube`, enable ingress and prepare `kubectl` we need to run few commands:

```bash
minikube start
minikube addons enable ingress
kubectl config use-context minikube
```

Now let's add a new `Makefile` goal to be able to install the service on `minikube`:

`Makefile`
```make
minikube: push
	for t in $(shell find ./kubernetes/advent -type f -name "*.yaml"); do \
        cat $$t | \
        	gsed -E "s/\{\{(\s*)\.Release(\s*)\}\}/$(RELEASE)/g" | \
        	gsed -E "s/\{\{(\s*)\.ServiceName(\s*)\}\}/$(APP)/g"; \
        echo ---; \
    done > tmp.yaml
	kubectl apply -f tmp.yaml
```

These commands "compile" all `*.yaml` configurations to a single file, replace `Release` and `ServiceName` "variables" by the real values (please, note that here I use `gsed` instead of the standard `sed`) and run `kubectl apply` to install the application to Kubernetes.

Let's check if our configuration works:

```bash
$ kubectl get deployment
NAME      DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
advent    3         3         3            3           1d

$ kubectl get service
NAME         CLUSTER-IP       EXTERNAL-IP   PORT(S)   AGE
advent       10.109.133.147   <none>        80/TCP    1d

$ kubectl get ingress
NAME      HOSTS         ADDRESS        PORTS     AGE
advent    advent.test   192.168.64.2   80        1d
```

Now we can try to send requests to the service. But first of all, we need to add our fake domain `advent.test` to the `/etc/host` file:

```bash
echo "$(minikube ip) advent.test" | sudo tee -a /etc/hosts
```

And now finally we can check our service:

```bash
curl -i http://advent.test/home
HTTP/1.1 200 OK
Server: nginx/1.13.6
Date: Sun, 10 Dec 2017 20:40:37 GMT
Content-Type: application/json
Content-Length: 72
Connection: keep-alive
Vary: Accept-Encoding

{"buildTime":"2017-12-10_11:29:59","commit":"020a181","release":"0.0.5"}%
```

Yeah, it works!

---

You can find all steps [here](https://github.com/rumyantseva/advent-2017), there are two versions available: [commit-by-commit](https://github.com/rumyantseva/advent-2017/commits/master) and [all steps in one](https://github.com/rumyantseva/advent-2017/tree/all-steps). If you have any questions, please, [create an issue](https://github.com/rumyantseva/advent-2017/issues/new) or ping me via twitter: [@webdeva](https://twitter.com/webdeva) or just leave a comment here.

It might be interesting for you how a more flexible service, prepared for the real production, may look like. In this case, feel free to take a look at [takama/k8sapp](https://github.com/takama/k8sapp), a Go application template which meets the Kubernetes requirements.

P.S. Many thanks to [Natalie Pistunovich](https://twitter.com/NataliePis), [Paul Brousseau](https://twitter.com/object88), [Sandor Szücs](https://twitter.com/sszuecs) and others for their review and comments.
