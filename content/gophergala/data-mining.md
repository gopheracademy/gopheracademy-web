+++
author = ["Marcel Hauf"]
date = "2015-02-02T17:10:37+01:00"
linktitle = "datamining"
title = "Data mining with Go"
+++

## What is the idea behind Gopher Gala project heatingeffect?

Heating Effect provides a visual representation of the raw notice data from (chillingeffects.org).

> The Chilling Effects database collects and analyzes legal complaints and requests for removal of online materials, helping Internet users to know their rights and understand the law. 
> This data enables us to study the prevalence of legal threats and let Internet users see the source of content removals.

This is achieved by getting the data from chillingeffects.org via a worker called harvester.
The harvested data gets stored directly in a MongoDB database.
A second worker aggregates the harvested data and stores it in the MongoDB database.
The aggregated data is displayed through a simple Go http server.

This post will focus on how data gets from chillingeffects.org into a MongoDB database via the harvester.



## The chillingeffects.org API

The [chillingeffects.org API](https://github.com/berkmancenter/chillingeffects/blob/master/doc/api_documentation.mkd) is a simple http JSON API.
To harvest the Notices the harvester only requires one function of the API which is [request a notice](https://github.com/berkmancenter/chillingeffects/blob/master/doc/api_documentation.mkd#request-a-notice).
The request is a GET call to the endpoint: `https://chillingeffects.org/notices/<notice id>.json`
On success the response body contains a JSON object.
The chillingeffects Go package has one function RequestNotice, which returns a Notice struct.


## Simple sequential harvester

Since bulk requests for notices are not possible with the API, each notice needs to be fetched on its own.
The simplest solution is to take an ID range and call the function RequestNotice for each ID.

``` Go
func harvestNotices(low, high int, session *mgo.Session) {
	for id := low; id <= high; id++ {
		notice, _ := chillingeffects.RequestNotice(id)
		session.DB("").C("notices").Insert(notice)
	}
}
```

The problem is, it simply takes too long to fetch thousands of notices.
Most of the time is spent waiting between a request and a response from chillingeffects.org and the database.
If you use a worker service like [Iron.io](http://www.iron.io/) and you are metered by the second your quota is exceeded very fast.


## Infusing goroutines

To reduce the time spent on each task, the harvester has to be optimized.
One of Go's advertised features are coroutines called goroutines.
A goroutine runs code concurrently to other goroutines.
Since they have little overhead, a simple solution to the time problem would be to start each request in its own goroutine.
Which would look like this:

``` Go
func harvestNotices(low, high int, session *mgo.Session) {
	for id := low; id <= high; id++ {
		go function(id int, session *mgo.Session) {
			notice, _ := chillingeffects.RequestNotice(id)
			session.DB("").C("notices").Insert(notice)
		}(id, session)
	}
}
```

The above code runs but most likely ends in an undesired result.
The program ends before the goroutines have a chance to request and store the notices.
The result is probably nothing but spend processing time.

## sync.WaitGroup

To avoid an application exit before the request goroutines are done, the main goroutine needs to wait.
Waiting/blocking can be done with mutexes or channels.
The Go package sync provides helpful functions for blocking.
The following code uses sync.WaitGroup to group all request goroutines together and blocks on the main goroutine until all request goroutines have called Done().


``` Go
func harvestNotices(low, high int, session *mgo.Session) {
	var wg sync.WaitGroup 
	for id := low; id <= high; id++ {
		go harvestNotice(id, &wg, session)
	}
	wg.Wait()
}

func harvestNotice(id int, wg *sync.WaitGroup, session *mgo.Session) {
	defer wg.Done()
	notice, _ := chillingeffects.RequestNotice(id)
	session.DB("").C("notices").Insert(notice)
}

```

## Further improvements also known as over engineering

The code can be further "improved" by using a goroutine pool to avoid creating too many goroutines.
You never known if somebody decides to pull millions of notices in one go.

Another improvement is to bulk insert the notices into MongoDB instead of each notice on its own.
Bulk insert is currently an experimental feature of the Go package mgo.

To achieve this, a limited amount of goroutines are created.
Each goroutine requests multiple notices sequentially. After a certain amount of responses, the notices are being bulk inserted into a MongoDB database.
This is repeated until the ids channel is closed.

``` Go
func harvest(low, high int, session *mgo.Session) {
	ids := make(chan int)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go work(ids, &wg, session)
	}
	for id := low; id <= high; id++ {
		ids <- id
	}
	close(ids)
	wg.Wait()
}

func work(ids <-chan int, wg *sync.WaitGroup, session *mgo.Session) {
	defer wg.Done()
	n := 0
	b := session.DB("").C("notices").Bulk()
	for id := range ids {
		notice, _ := chillingeffects.RequestNotice(id)
		b.Insert(notice)
		n++
		if n == 99 {
			b.Run()
			b = session.DB("").C("notices").Bulk()
			n = 0
		}
	}
	if n > 0 {
		b.Run()
	}
}
```

## A note on error handling

I omitted error handling in the above code examples. You should never ignore a returned error value.


## Links

 + [Orignal Gopher Gala submission](https://github.com/gophergala/heatingeffect)
 + [Continued version](https://github.com/marshauf/heatingeffect)
 + Twitter [@marshauf](https://twitter.com/marshauf)
