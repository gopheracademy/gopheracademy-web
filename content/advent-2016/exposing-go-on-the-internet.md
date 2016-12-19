+++
date = "2016-12-19T01:30:27Z"
author = ["Filippo Valsorda"]
title = "So you want to expose Go on the Internet"
linktitle = "Exposing Go on the Internet"
series = ["Advent 2016"]
+++

Back when `crypto/tls` was slow and `net/http` young, the general wisdom was to always put Go servers behind a reverse proxy like NGINX. That's not necessary anymore!

However, the Internet is the deep end of the pool when it comes to networks, and there are a few things you have to do to teach your server to swim before throwing it in.

At [Cloudflare][cf] we recently experimented with exposing pure Go services to the hostile wide area network. Here are some hard-learned lessons.

[cf]: https://blog.cloudflare.com/tag/go/

## `crypto/tls`

You're not running an insecure HTTP server on the Internet in 2016. So you need `crypto/tls`. The good news is that it's [now really fast][gap] (as you've seen in a [previous article on this blog][bench]), and its security track record so far is excellent.

The default settings resemble the *Intermediate* recommended configuration of the [Mozilla guidelines][mozilla]. However, you should still set `PreferServerCipherSuites` to ensure safer and faster cipher suites are preferred, and `CurvePreferences` to avoid unoptimized curves: a client using `CurveP384` would cause up to a second of CPU to be consumed on our machines.

```go
&tls.Config{
	// Causes servers to use Go's default ciphersuite preferences,
	// which are tuned to avoid attacks. Does nothing on clients.
	PreferServerCipherSuites: true,
	// Only use curves which have assembly implementations
	CurvePreferences: []tls.CurveID{
		tls.CurveP256,
                tls.X25519, // Go 1.8 only
	},
}
```

If you can take the compatibility loss of the *Modern* configuration, you should then also set `MinVersion` and `CipherSuites`.

```go
	MinVersion: tls.VersionTLS12,
	CipherSuites: []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305, // Go 1.8 only
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305, // Go 1.8 only
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,

		// Best disabled, as they don't provide Forward Secrecy,
		// but might be necessary for some clients
		// tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		// tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
	},
```

Be aware that the Go implementation of the CBC cipher suites (the ones we disabled in *Modern* mode above) is vulnerable to the [Lucky13 attack][lucky13], even if [partial countermeasures were merged in 1.8][patch].

Final caveat, all these recommendations apply only to the amd64 architecture, for which [fast, constant time implementations][gap] of the crypto primitives (AES-GCM, ChaCha20-Poly1305, P256) are available. Other architectures are probably not fit for production use.

Since this server will be exposed to the Internet, it will need a publicly trusted certificate. You can get one easily and for free thanks to Let's Encrypt and the [`golang.org/x/crypto/acme/autocert`][autocert] package’s `GetCertificate` function.

Don't forget to redirect HTTP page loads to HTTPS, and consider [HSTS][hsts] if your clients are browsers.

```go
srv := &http.Server{
	ReadTimeout:  5 * time.Second,
	WriteTimeout: 5 * time.Second,
	Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Connection", "close")
		url := "https://" + req.Host + req.URL.String()
		http.Redirect(w, req, url, http.StatusMovedPermanently)
	}),
}
go func() { log.Fatal(srv.ListenAndServe()) }()
```

You can use the [SSL Labs test][ssllabs] to check that everything is configured correctly.

[bench]: https://blog.gopheracademy.com/advent-2016/tls-termination-bench/
[mozilla]: https://wiki.mozilla.org/Security/Server_Side_TLS
[ssllabs]: https://www.ssllabs.com/ssltest/
[lucky13]: https://www.imperialviolet.org/2013/02/04/luckythirteen.html
[patch]: https://github.com/golang/go/commit/f28cf8346c4ce7cb74bf97c7c69da21c43a78034
[autocert]: https://godoc.org/golang.org/x/crypto/acme/autocert
[gap]: https://blog.cloudflare.com/go-crypto-bridging-the-performance-gap/
[hsts]: https://www.owasp.org/index.php/HTTP_Strict_Transport_Security_Cheat_Sheet

## `net/http`

`net/http` is a mature HTTP/1.1 and HTTP/2 stack. You probably know how (and have opinions about how) to use the Handler side of it, so that's not what we'll talk about. We will instead talk about the Server side and what goes on behind the scenes.

### Timeouts

Timeouts are possibly the most dangerous edge case to overlook. Your service might get away with it on a controlled network, but it will not survive on the open Internet, especially (but not only) if maliciously attacked.

Applying timeouts is a matter of resource control. Even if goroutines are cheap, file descriptors are always limited. A connection that is stuck, not making progress or is maliciously stalling should not be allowed to consume them.

A server that ran out of file descriptors will fail to accept new connections with errors like

```
http: Accept error: accept tcp [::]:80: accept: too many open files; retrying in 1s
```

A zero/default `http.Server`, like the one used by the package-level helpers `http.ListenAndServe` and `http.ListenAndServeTLS`, comes with no timeouts. You don't want that.

![HTTP server phases](/postimages/advent-2016/Timeouts.png)

There are three main timeouts exposed in `http.Server`: `ReadTimeout`, `WriteTimeout` and `IdleTimeout`. You set them by explicitly using a Server:

```go
srv := &http.Server{
    ReadTimeout:  5 * time.Second,
    WriteTimeout: 10 * time.Second,
    IdleTimeout:  120 * time.Second,
    TLSConfig:    tlsConfig,
    Handler:      serveMux,
}
log.Println(srv.ListenAndServeTLS("", ""))
```

`ReadTimeout` covers the time from when the connection is accepted to when the request body is fully read (if you do read the body, otherwise to the end of the headers). It's implemented in `net/http` by calling `SetReadDeadline` [immediately after Accept][setreaddeadline].

The problem with a `ReadTimeout` is that it doesn't allow a server to give the client more time to stream the body of a request based on the path or the content. Go 1.8 introduces `ReadHeaderTimeout`, which only covers up to the request headers. However, there's still no clear way to do reads with timeouts from a Handler. Different designs are being discussed in issue [#16100][16100].

`WriteTimeout` normally covers the time from the end of the request header read to the end of the response write (a.k.a. the lifetime of the ServeHTTP), by calling `SetWriteDeadline` [at the end of readRequest][setwritedeadline].

However, when the connection is over HTTPS, `SetWriteDeadline` is called [immediately after Accept][tlsdeadline] so that it also covers the packets written as part of the TLS handshake. Annoyingly, this means that (in that case only) `WriteTimeout` ends up including the header read and the first byte wait.

Similarly to `ReadTimeout`, `WriteTimeout` is absolute, with no way to manipulate it from a Handler ([#16100][16100]).

Finally, Go 1.8 [introduces `IdleTimeout`][14204] which limits server-side the amount of time a Keep-Alive connection will be kept idle before being reused. Before Go 1.8, the `ReadTimeout` would start ticking again immediately after a request completed, making it very hostile to Keep-Alive connections: the idle time would consume time the client should have been allowed to send the request, causing unexpected timeouts also for fast clients.

You should set `Read`, `Write` and `Idle` timeouts when dealing with untrusted clients and/or networks, so that a client can't hold up a connection by being slow to write or read.

For detailed background on HTTP/1.1 timeouts (up to Go 1.7) read [my post on the Cloudflare blog][timeouts].

[14204]: https://github.com/golang/go/issues/14204
[16100]: https://golang.org/issue/16100
[timeouts]: https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
[setreaddeadline]: https://github.com/golang/go/blob/3ba31558d1bca8ae6d2f03209b4cae55381175b3/src/net/http/server.go#L750
[setwritedeadline]: https://github.com/golang/go/blob/3ba31558d1bca8ae6d2f03209b4cae55381175b3/src/net/http/server.go#L753-L755
[tlsdeadline]: https://github.com/golang/go/blob/3ba31558d1bca8ae6d2f03209b4cae55381175b3/src/net/http/server.go#L1477-L1483

#### HTTP/2

HTTP/2 is enabled automatically on any Go 1.6+ server if:

* the request is served over TLS/HTTPS
* `Server.TLSNextProto` is `nil` (setting it to an empty map is how you disable HTTP/2)
* `Server.TLSConfig` is set and `ListenAndServeTLS` is used  **or**
* `Serve` is used and `tls.Config.NextProtos` includes `"h2"` (like `[]string{"h2", "http/1.1"}`, since `Serve` is called [too late to auto-modify the TLS Config][15908])

HTTP/2 has a slightly different meaning since the same connection can be serving different requests at the same time, however, they are abstracted to the same set of Server timeouts in Go.

Sadly, `ReadTimeout` breaks HTTP/2 connections in Go 1.7. Instead of being reset for each request it's set once at the beginning of the connection and never reset, breaking all HTTP/2 connections after the `ReadTimeout` duration. [It's fixed in 1.8][16450].

Between this and the inclusion of idle time in `ReadTimeout`, my recommendation is to upgrade to 1.8 as soon as possible.

[15908]: https://github.com/golang/go/issues/15908
[16450]: https://github.com/golang/go/issues/16450

#### TCP Keep-Alives

If you use `ListenAndServe` (as opposed to passing a `net.Listener` to `Serve`, which offers zero protection by default) a TCP Keep-Alive period of three minutes [will be set automatically][tcpKeepAliveListener]. That *will* help with clients that disappear completely off the face of the earth leaving a connection open forever, but I’ve learned not to trust that, and to set timeouts anyway.

To begin with, three minutes might be too high, which you can solve by implementing your own [`tcpKeepAliveListener`][tcpKeepAliveListener].

More importantly, a Keep-Alive only makes sure that the client is still responding, but does not place an upper limit on how long the connection can be held. A single malicious client can just open as many connections as your server has file descriptors, hold them half-way through the headers, respond to the rare keep-alives, and effectively take down your service.

Finally, in my experience connections tend to leak anyway until [timeouts are in place][heartbleed].

[heartbleed]: https://github.com/FiloSottile/Heartbleed/commit/4a3332ca1dc07aedf24b8540857792f72624cdf7
[tcpKeepAliveListener]: https://github.com/golang/go/blob/61db2e4efa2a8f558fd3557958d1c86dbbe7d3cc/src/net/http/server.go#L3023-L3039

### ServeMux

Package level functions like `http.Handle[Func]` (and maybe your web framework) register handlers on the global `http.DefaultServeMux` which is used if `Server.Handler` is nil. You should avoid that.

Any package you import, directly or through other dependencies, has access to `http.DefaultServeMux` and might register routes you don't expect. 

For example, if any package somewhere in the tree imports `net/http/pprof` clients will be able to get CPU profiles for your application. You can still use `net/http/pprof` by registering [its handlers][pprof] manually.

Instead, instantiate an `http.ServeMux` yourself, register handlers on it, and set it as `Server.Handler`. Or set whatever your web framework exposes as `Server.Handler`.

[pprof]: https://github.com/golang/go/blob/1106512db54fc2736c7a9a67dd553fc9e1fca742/src/net/http/pprof/pprof.go#L67-L71

### Logging

`net/http` does a number of things before yielding control to your handlers: [`Accept`s the connections][accept], [runs the TLS Handshake][handshake], ...

If any of these go wrong a line is written directly to `Server.ErrorLog`. Some of these, like timeouts and connection resets, are expected on the open Internet. It's not clean, but you can intercept most of those and turn them into metrics by matching them with regexes from the Logger Writer, thanks to this guarantee:

> Each logging operation makes a single call to the Writer's Write method.

To abort from inside a Handler without logging a stack trace you can either `panic(nil)` or in Go 1.8 `panic(http.ErrAbortHandler)`.

[accept]: https://github.com/golang/go/blob/1106512db54fc2736c7a9a67dd553fc9e1fca742/src/net/http/server.go#L2631-L2653
[handshake]: https://github.com/golang/go/blob/1106512db54fc2736c7a9a67dd553fc9e1fca742/src/net/http/server.go#L1718-L1728

### Metrics

A metric you'll want to monitor is the number of open file descriptors. [Prometheus does that by using the `proc` filesystem][proc].

If you need to investigate a leak, you can use the `Server.ConnState` hook to get more detailed metrics of what stage the connections are in. However, note that there is no way to keep a correct count of `StateActive` connections without keeping state, so you'll need to maintain a `map[net.Conn]ConnState`.

[proc]: https://github.com/prometheus/client_golang/blob/575f371f7862609249a1be4c9145f429fe065e32/prometheus/process_collector.go

## Conclusion

The days of needing NGINX in front of all Go services are gone, but you still need to take a few precautions on the open Internet, and probably want to upgrade to the shiny, new Go 1.8.

Happy serving!

