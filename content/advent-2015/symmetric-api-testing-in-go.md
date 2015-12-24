+++
author = ["Aditya Mukerjee"]
date = "2015-12-26T00:00:00-05:00"
title = "Symmetric API Testing"
series = ["Advent 2015"]
draft = false
+++

I maintain [Anaconda](https://github.com/ChimeraCoder/anaconda), the Twitter client library for Go. There are a lot of interesting things I could write about Anaconda - for example, automatic rate-limiting and throttling using the [tokenbucket](https://github.com/ChimeraCoder/tokenbucket) library. Today, I'd like to demonstrate symmetric API testing in Go, which Anaconda highlights quite well. 

The asymmetric approach to testing the client library would be to test each function by querying the Twitter API and testing the response values returned. This is the easiest way to start, and especially when developing locally, it's the most logical place to begin.

However, this has a few downsides. First, the Twitter API is rate-limited, which means that running the same test suite multiple times in a short period  of time will cause the later tests to take a long time to complete. If the client library did not automatically handle throttling and rate-limiting, these tests would fail entirely, resulting in a flaky test suite. Additionally, these tests require a set of API credentials with full read/write permissions in order to perform the test. Managing these credentials securely on public testing infrastructure is somewhere between cumbersome and impossible.

It's also quite slow. Even without accounting for normal rate-limiting and throttling, it can take a full minute to run the tests (depending on how many tests there are, the quality of the network connection, and the current state of the Twitter API servers). Accounting for the rate limits imposed by Twitter, it can actually take several minutes to run the full test suite, depending on how recently the API credentials were last used (ie, how recently the test suite was last run).


Finally, this approach makes the test suite tightly coupled to the Twitter API. The tests should not rely on the Twitter API being available.  Twitter itself could be down, or the tests could be run in a sandbox that does not permit external network connections. We still want the tests to function properly.


Our ideal test suite would be capable of running in a hermetic environment with no external network connectivity, while also capable of detecting changes to the responses returned by the Twitter API. If we were also responsible for implementing the API server as well, our ideal test suite would also test that the responses returned by the server of the server match the behavior that the client library expects.

We can accomplish all of this - without any duplication of code - using *symmetric testing*. Symmetric testing means that we clearly define the relationship between data and code, store this information in a single place, and use our tests to validate that this relationship is correct.





Serving responses: Local HTTP server
-----------

Instead of making an HTTP request to Twitter every time the tests run, we can set up a test server that runs locally and responds to each request with the response body we expect.


```go

    mux := http.NewServeMux()
    server := httptest.NewServer(mux)

    parsed, _ := url.Parse(server.URL)
    api.SetBaseUrl(parsed.String() + "/")
    mux.HandleFunc("/"+endpoint, func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprint(w, `<insert sample response here>`)
    })
```


The sample responses for each endpoint can be copied straight from the [Twitter API documentation](https://dev.twitter.com/rest/reference/get/statuses/show/%3Aid). This server will respond to the requests made by the client library that we are testing with the actual responses that we know Twitter *should* return. Our client library receives a valid response, even if we are running the tests offline. 

Furthermore, because we are running this server locally and not imposing any rate limits on our test requests, we cut our testing time down from several minutes to just a few seconds - compiling *and* running the test suite with the recorded responses takes less than three seconds on my laptop. 



Recording Responses: io.Reader
-----------------------------------

However, copying and pasting responses from the documentation is tedious. Worse, there's no guarantee that the examples included in the documentation are complete or up-to-date, which [has been a problem in practice](https://github.com/ChimeraCoder/anaconda/pull/63). Instead of trusting the documentation to provide us with accurate, complete, and up-to-date responses, we can test our client against the API *as it actually is*, as opposed to testing it against what the documentation claims it should be.


Go's `io.Reader` and `io.Writer` interfaces make it incredibly easy to record the real HTTP response bodies.


The original client code makes an OAuth-authenticated HTTP request and decodes the JSON response body:

```go
    resp, err := oauthClient.Get(c.HttpClient, c.Credentials, urlStr, form)
    if err != nil {
        return err
    }       
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        // handle error
    }   
    return json.NewDecoder(resp.Body).Decode(data)
```

We only have to change one line in the client code in order to make this function copy the response body to a file, while leaving the rest of the behavior unmodified.

```go
    resp, err := oauthClient.Get(c.HttpClient, c.Credentials, urlStr, form)
    if err != nil {
        return err
    }       
    defer resp.Body.Close()

    resp.Body = ioutil.NopCloser(io.TeeReader(resp.Body, f))
    if resp.StatusCode != 200 {
        // handle error
    }   
    return json.NewDecoder(resp.Body).Decode(data)
```

In this case, `f` is an `*os.File` that we opened earlier in the function (`os.Open`). Because the HTTP response struct stores the body as an `io.Reader` and the JSON decoder accepts any `io.Reader`, it doesn't matter that we've teed the original response body to a file. We could have instead used `ioutil.ReadAll` to obtain a `[]byte` of JSON data that we unmarshal and write to a file, but this approach is both cleaner and more concise. 

If we want to inspect the output as we record it, we simply tee again, as `os.Stdout` and `os.Stderr` also support the `io.Writer` interface.

```go
    resp.Body = ioutil.NopCloser(io.TeeReader(io.TeeReader(resp.Body, f), os.Stdout))
```


The hardest part of programming is naming things, but fortunately, the API gives us a convenient naming scheme for the file. The `/statuses/show.json` endpoint can be stored in the project directory under `json/statuses/show.json`. Because this information is already part of the HTTP GET/POST request, we can express this in a one-liner as well:

```go
    filename := filepath.Join(append([]string{"json"}, strings.Split(strings.TrimPrefix(urlStr, c.baseUrl), "/")...)...)
```

We'll take this line out before committing the code so that the client library doesn't require filesystem access when running in production. Since it's such a small change, it's easy to add this back in as needed.


There is one disadavantage to recording the responses in this way: not all endpoints will return identical responses over time. It's not important if `GET  /posts/latest` returns different data tomorrow from what it returned today, as long as the structure of the response remains predictable. Unfortunately, some APIs allow the structure of the responses to vary. For example, one of the fields on a tweet, `scopes`, is usually a map of strings to boolean values (eg. `"scopes":{"followers":false}`), but [on some tweets](https://github.com/ChimeraCoder/anaconda/issues/82) it is actually a map of strings to arrays of strings (e.g. `"scopes":{"place_ids":["c799e2d3a79f810e"]}`). To avoid missing edge cases when using recorded responses, you will want to identify these exceptions and incorporate them into your testing suite. 


(On that note, a related plea: if you are designing an API, please keep request/response structures uniform. Not only are variable structures inconvenient in statically-typed languages like Go, but they are *also* inconvenient in weakly, dynamically-typed languages like Javascript.)

Generating Structs with Gojson
--------------

Since we already have an entire folder full of example JSON data, it seems a bit redundant to have to type out all the struct definitions by hand. Worse, if the Twitter API response ever changes structure, we have to manually update the relevant fields.

Fortunately, [gojson](https://github.com/ChimeraCoder/gojson) lets us bypass this process entirely, generating the struct definitions automatically from the example JSON responses.


```go
//go:generate gojson -o tweet.go -name "Tweet" -pkg "anaconda" -input json/statuses/show.json
```

We can place this comment in any `.go` file in the repository, and when we run `go generate`, the struct definition will be written to `tweet.go`.


Tying it all together
------------------------

Now, the entire client can be tested against the most recent responses returned by the Twitter API, without requiring any external network connections. The tests can be updated whenever the API itself changes by updating the recorded responses. Updating the recorded responses *also* updates the client to work with the new API. Our tests and our client code are kept in sync, and updating them both can be done in a single move.


For this example, we've shown symmetric testing for an API client. The same process can also be used to test the API server - the `net/http/httptest` package provides a `ResponseRecorder` that allows us to make requests directly on handler functions and inspect the results. Just as we used a mock server to intercept requests from the client to test the client, we can use a mock client to fake requests to the server and test the server.

When testing the client, we recorded real JSON response bodies and replayed them when testing using a mock server. We can use the same strategy here as well - this time, we record the *request* bodies and replay them using a mock *client*. The big question remains: what naming scheme do we use for the files that store the recorded responses? For anaconda, we are only interested in testing the client functionality, but it is straightforward to modify the naming scheme to include the HTTP method (for example, `json/statuses/show.json.REQ`, which has the advantage of keeping the request and response in the same directory). 

However, a [well-designed RESTful API](https://codewords.recurse.com/issues/five/what-restful-actually-means) will sometimes structure the requests and responses identically. (Or, commonly, the structure of the request will be isomorphic to the structure of one of the fields in the response object). In both of these cases, we can actually reuse the JSON data we stored when testing the client.


The next time you are writing an API server or client library, use symmetric testing to save yourself work and reduce the number of moving parts that you have to keep in sync with each other. The particular details will depend on the API you are testing against (in the case of a client), the API your are writing (in the case of a server), or the API that you are both writing and testing against (in the case you are implementing both halves). Regardless of which case applies to your situation, there is no need to copy around large chunks of data just so you can write boilerplate code around it. When we clearly define the relationship between our data and code, we end up writing less code, and the code that we write is more robust.
