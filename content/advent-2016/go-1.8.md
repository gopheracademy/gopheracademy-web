+++
author = ["Florin Pățan", "Filippo Valsorda"]
date = "2016-12-30T00:00:00+00:00"
title = "Go 1.8"
series = ["Advent 2016"]
+++

With Go following a predetermined release schedule of February - August and
a Release Candidate for Go 1.8 just a few days after this article, it looks
like we should be able to talk about Go 1.8 without too much fear that things
will change.

Lets start with some of the low-level changes.

You may remember that Go 1.7 introduced a new compiler backend that is based on
[SSA](https://en.wikipedia.org/wiki/Static_single_assignment_form), or Static
Single Assignment form, which helps improving code generation and allows for
more optimizations to happen. At the time that backend was available only for
the x86 amd64 platform. As of Go 1.8 however, this will be present for all the
other platforms as well. And while for the x86-64 platform the performance
increase was up to 10%, for all the other platforms it's expected to be between
20% and 30%.

Not only the compiler backend has suffered changes. The compiler frontend has
also been overhauled in order to allow further performance improvements.

Overall, Go 1.8 continues the work on getting back the compile speed lost in
the 1.5 release and it's now around 15% faster to compile than 1.7 but still
slower than 1.4.

Speaking of compiler changes, you can now enjoy deploying Go to even more
platforms than before with the addition of MIPS 32, both LE and BE, on
platforms that support MIPS32r1 instruction set with FPU (either hardware or
emulated by the kernel).

The Go tooling has also been updated with `go fix` now converting the
`golang.org/x/net/context` to `context` in order to facilitate the migration
to the new context package. `go vet` as also been updated so remember to run
it on your codebase in order to check for potential issues.

If you happen to encounter a bug in Go then you can use the new `go bug`
command which will help you pre-fill the bug report with all the system
information needed in order to ensure that the needed details are present.

There's also a new supported build mode which enables Go applications to
support compiling to and loading plugins. This will effectively allow new ways
for applications to provide functionality based on the needs of their users. If
you want to read more about it, you can see the [documentation](https://tip.golang.org/pkg/plugin).

From 1.8, Go will not need an environment variable in order to determine the
GOPATH. If this will not be present anymore, it will default to a the HOME/go
as default directory but if you already have this set, then it will continue to
work as before. You can see the commit which enables this [here](https://golang.org/cl/32019).

Go's garbage collector did not go untouched in this release as well. As part of
the ongoing effort to remove the stop-the-world stack rescanning, see the
[proposal document](https://github.com/golang/proposal/blob/master/design/17503-eliminate-rescan.md) Go gained a new [hybrid write barrier](https://golang.org/cl/31765)
which should lower the garbage collection time pause to under 100us.

The function arguments will also no longer live until the end of the function,
which means that memory can be freed before the function execution ends. For
details about this functionality, you can read the [issue](https://github.com/golang/go/issues/15843) and the [implementation](https://go-review.googlesource.com/c/28310/).

If you think that's already a long list of changes, lets quickly see the
various packages from the Go standard library and see what's new or fixed for
them. For the purpose of time, we'll skip over 100 commits which are meant to
enhance the performance across the board.

Go 1.8 brings a much more mature TLS stack to `crypto/tls`, benefiting HTTPS
clients and servers.

Support for [ChaCha20-Poly1305 based cipher suites](https://blog.cloudflare.com/do-the-chacha-better-mobile-performance-with-cryptography/),
and the X25519 key exchange bring speed and security improvements.

If you were setting [tls.Config.CurvePreferences](https://tip.golang.org/pkg/crypto/tls/#Config.CurvePreferences)
you will want to add `tls.X25519`, and if you were setting [tls.Config.CipherSuites](https://tip.golang.org/pkg/crypto/tls/#Config.CipherSuites) add:

```
tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
```

Maturity is not just features, but flexibility as well. 1.8 introduces a few
callbacks like the existing [GetCertificate](https://tip.golang.org/pkg/crypto/tls/#Config.GetCertificate) that allow a lot of custom extensions.

[GetConfigForClient](https://tip.golang.org/pkg/crypto/tls/#Config.GetConfigCertificate) is
run for every server connection, and allows customizing all fields of the
`Config` based on the information in the [ClientHelloInfo](https://tip.golang.org/pkg/crypto/tls/#ClientHelloInfo),
which has been expanded. This allows all kinds of things, like enabling client
certificates ([#15707](https://github.com/golang/go/issues/15707)) or HTTP/2 selectively based on the website being requested:

```
type getConfigForClient func(*tls.ClientHelloInfo) (*tls.Config, error)

func partialHTTP2Config(baseConfig *tls.Config, http2Sites []string) getConfigForClient {
    http2Config := baseConfig.Clone() // new in 1.8, too
    http2Config.NextProtos = []string{"h2", "http/1.1"}

    http1Config := baseConfig.Clone()
    http1Config.NextProtos = []string{"http/1.1"}

    return func(chi *tls.ClientHelloInfo) (*tls.Config, error) {
        for _, enabled := range http2Sites {
            if chi.ServerName == enabled ||
                strings.HasSuffix(chi.ServerName, "."+enabled) {
                return http2Config, nil
            }
        }
        return http1Config, nil
    }
}
```

The new `Conn` field in `ClientHelloInfo` allows `GetConfigForClient` or
`GetCertificate` to make decisions based on the remote or local IP (needed for
clients that don't support SNI), but shouldn't be used to write or read data as
that would corrupt the handshake.

[GetClientCertificate](https://tip.golang.org/pkg/crypto/tls/#Config.GetClientCertificate)
lets clients pick their certificate based on the server CA preferences.

Finally, [VerifyPeerCertificate](https://tip.golang.org/pkg/crypto/tls/#Config.VerifyPeerCertificate)
enables custom certificate checking logic, which I hope will be implemented by
external libraries to provide for example revocation and CT checks.

The Go TLS stack also learned to write debugging key files to a [KeyLogWriter](https://tip.golang.org/pkg/crypto/tls/#Config.KeyLogWriter)
file. These files can be [loaded into Wireshark to decrypt the TLS traffic](https://github.com/joneskoo/http2-keylog).

Interestingly, [a comment by Adam Langley in the `crypto/tls` source](https://github.com/golang/go/blob/230a376b5a67f0e9341e1fa47e670ff762213c83/src/crypto/tls/conn.go#L321-L330)
had predicted the Lucky13 vulnerability in CBC cipher suites. It has since been
fixed in OpenSSL, but full countermeasures are extremely complex, so they
haven't been ported to the Go library since better cipher suites are available.
1.8 introduces [partial mitigation](https://github.com/golang/go/commit/f28cf8346c4ce7cb74bf97c7c69da21c43a78034), which make the attack harder.

Finally, AES-GCM cipher suites will automatically be preferred when hardware
support is present in the default `crypto/tls`, see [this change](https://go-review.googlesource.com/c/32871/).

Here's the [full changelog](https://tip.golang.org/doc/go1.8#crypto_tls) with some other minor improvements.

[GetConfigForClient]: https://tip.golang.org/pkg/crypto/tls/#Config. GetConfigForClient

`database/sql` was already covered in a [previous article](https://blog.gopheracademy.com/advent-2016/database_sql/) by Daniel Theophanes.

`encoding` package has a few changes:

- `encoding/binary`: supports [bool values](https://golang.org/cl/28514)
- `encoding/json`: adds struct and field name to UnmarshalTypeError message with [this change](https://golang.org/cl/18692)
- `encoding/json`: uses standard ES6 formatting for numbers during marshal, with [this change](https://golang.org/cl/30371)
- `encoding/xml`: wildcard support for collecting all attributes, with [this change](https://golang.org/cl/30946)

`expvar` gained a way to retrieve the value back using the new [Value method](https://golang.org/cl/30917),
and it also [exports the the handler](https://golang.org/cl/24722) thus allowing
usage via different ServeMux.

`net` has some interesting changes as well:

- adds Buffers type and writev on [Unix](https://golang.org/cl/29951) and [Windows](https://golang.org/cl/32371)
- adds [Resolver type](https://golang.org/cl/29440), Dialer.Resolver, and DefaultResolver
- will break up >1GB reads and writes on stream connections, with [this change](https://golang.org/cl/31584)
- will respect resolv.conf rotate option, with [this change](https://golang.org/cl/29233)
- use libresolv rules for ndots range and validation, with [this change](https://golang.org/cl/24901)

`net/http`, one of the most popular packages in Go, was also very popular with
a lot of contributions to it. Below you can find a selection of them:

- the Server [gained](https://golang.org/cl/32329) Server.Close & Server.Shutdown for forced & graceful shutdown
- add Server.ReadHeaderTimeout, IdleTimeout, document WriteTimeout, with [this commit](https://golang.org/cl/32024)
- adds Transport.ProxyConnectHeader to control headers to proxies, with [this commit](https://golang.org/cl/32481)
- HTTP/2 server push is now [supported](https://golang.org/cl/32012)
- make Server Handler's Request.Context be done on conn errors, with [this commit](https://golang.org/cl/31173)
- make Server log on bad requests from clients, with [this commit](https://golang.org/cl/27950)
- make Transport support international domain names, with [this commit](https://golang.org/cl/29072)
- support If-Match in ServeContent, with [this commit](https://golang.org/cl/32014)

The `runtime` also gained additional features like:

- profile goroutines holding contended mutexes, with [this commit](https://golang.org/cl/29650)
- disable stack rescanning by default, with [this commit](https://golang.org/cl/31766)
- include pre-panic/throw logs in core dumps, with [this commit](https://golang.org/cl/32013)

`sort` gained new helpers for sorting slices, with [this commit](https://golang.org/cl/27321).
As such, sorting slices is a lot simpler now and you can see an example bellow
for the new way to do it:

```
sort.Slice(s, func(i, j int) bool {
    if s[i].Foo != s[j].Foo {
        return s[i].Foo < s[j].Foo
    }
    return s[i].Bar < s[j].Bar
})
```

The `testing` package gained new functionality, among which:

- add Name method to *T and *B, with [this commit](https://golang.org/cl/29970)
- tests and benchmarks failed if a race occurs during execution will be marked as such, with [this commit](https://golang.org/cl/32615)
- benchtime is now respected on very fast benchmarks, with [this commit](https://golang.org/cl/26664)

The overall documentation of the Go standard library has received a lot of
contributions. As always, if you think something can be improved, please open
up issues in order to at least let others know where the problems are or even
give it a try contribute to it.

If you are eager to try Go 1.8, you can either do it so now, and use the Go 1.8
Beta 2 release or you can wait a few more days for the Release Candidate 1, and
you are highly encouraged to do so in order to guarantee a smooth release of
1.8 and enjoy it in production without any problems.

To view the full list of changes, you can read this the [Draft Release Notes](https://beta.golang.org/doc/go1.8).

Finally, if you want to stay up to date with the developments of Go, you are
encouraged to follow the [@golang_cls](https://twitter.com/golang_cls) Twitter account
which will provide a list of curated commits as they are added to Go.
