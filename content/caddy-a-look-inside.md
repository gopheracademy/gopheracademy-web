+++
author = ["Matt Holt"]
date = "2015-05-27"
title = "A Look Inside Caddy, a Web Server Written in Go"
+++

[Caddy](https://caddyserver.com) is a unique web server with a modern feature set. Think nginx or Apache, but written in Go. With Caddy, you can serve your websites over HTTP/2. It can act as a [reverse proxy and load balancer](https://caddyserver.com/docs/proxy). [Front your PHP apps](https://caddyserver.com/docs/fastcgi) with it. You can even [deploy your site](https://caddyserver.com/docs/git) with `git push`. Cool, right?

Caddy serves the Gopher Academy websites, including this blog. Go ahead, check out the response headers. At the end of this post, we'll show you how this is done.

By the way, even if you're new to Go, Caddy is a great project to contribute to. For example, right now we need more tests. Much of the setup/boilerplate is already done. Please feel free to [get involved](https://github.com/mholt/caddy/blob/master/CONTRIBUTING.md)! There's a great community of Caddy users to collaborate with.

## Introduction

Caddy is basically a web app. Like other Go web applications, it imports [net/http](https://golang.org/pkg/net/http), embeds `*http.Server`, has `ServeHTTP()` methods, and uses `http.FileServer` as a basis for serving static files.

Even though Caddy resembles a regular web app, it diverges in several significant and challenging ways. Its _entire_ configuration could change from one execution to the next. (Soon, you'll be able to make changes to Caddy without needing to restart it.)

In this post, we'll examine a few of the critical design decisions that make Caddy tick.


## User Interface and Experience

Caddy is a headless application. There is no visual UI (yet) and it can run without any interaction from the user. This does not, however, eliminate the user interface/experience.

I believe the first key element in any application is the user experience. Nearly every technical decision should be checked-and-balanced with a cross-examination: "How does the user like this?" This can be an important discussion to have, and the first four months of development was me talking myself through the answers to that question.

As you read about and use Caddy, I hope you'll see what I mean when I say it was designed for _people_, with the Web in mind.


## Middleware

Middleware is absolutely the #1 reason that Caddy works. If I were to choose one thing that is Caddy's secret sauce, this is it.

In essence, Caddy has just one HTTP handler: [the file server](https://github.com/mholt/caddy/blob/master/server/fileserver.go). [The rest](https://github.com/mholt/caddy/tree/master/middleware) is all middleware. Each middleware does one thing very well. For example, [logging](https://github.com/mholt/caddy/blob/e42c6bf0bb00d2e5e966ec7d9923eb21627a6b74/middleware/log/log.go), [authentication](https://github.com/mholt/caddy/blob/e42c6bf0bb00d2e5e966ec7d9923eb21627a6b74/middleware/basicauth/basicauth.go), or [gzip compression](https://github.com/mholt/caddy/blob/e42c6bf0bb00d2e5e966ec7d9923eb21627a6b74/middleware/gzip/gzip.go).

All of Caddy's middleware [can be used in your own Go programs](https://github.com/mholt/caddy/wiki/Using-Caddy-Middleware-in-Your-Own-Programs) independently of Caddy.

Middleware is usually chained together. For example, to do both gzip compression and logging, you would wrap a handler like `logHandler(gzipHandler(fileServer))`. With most web apps, you could just hardcode this. But because users can customize Caddy, we can't hardcode its middleware chain. Instead, it [compiles a custom middleware stack](https://github.com/mholt/caddy/blob/e42c6bf0bb00d2e5e966ec7d9923eb21627a6b74/server/virtualhost.go#L34-L41) based on user input:

<script type="text/javascript" src="https://sourcegraph.com/R$3104360@535f95668282b829ed8f8b2c56e9576e1136e3cf===535f95668282b829ed8f8b2c56e9576e1136e3cf/.tree/server/virtualhost.go/.sourcebox.js?StartLine=34&EndLine=41"></script>

In that code, `layers` is a slice of functions that [take a Handler and return a Handler](https://github.com/mholt/caddy/blob/e42c6bf0bb00d2e5e966ec7d9923eb21627a6b74/middleware/middleware.go#L10-L13), true to the traditional middleware pattern. The `fileServer` is the HTTP handler at the core of every request (the "end" of the chain). When the loop finishes, `vh.stack` points to the beginning of the chain through which all requests will pass.

By compiling the middleware stack dynamically, the user can customize exactly what functionality they want their web server to have.


## Error Handling

There are [several ways to handle errors in HTTP handlers](http://mwholt.blogspot.com/2015/05/handling-errors-in-http-handlers-in-go.html) ([slides](https://docs.google.com/presentation/d/1QiyqQRDalifqKYSN9FwNkTHRv_5dxqY-5BjLf_xPZnA)). I [changed the signature](https://github.com/mholt/caddy/blob/e42c6bf0bb00d2e5e966ec7d9923eb21627a6b74/middleware/middleware.go#L15-L41) of  `ServeHTTP()` to return `(int, error)`. It's not directly compatible with net/http, but this pattern is one [recommended by the Go Blog](http://blog.golang.org/error-handling-and-go) and it works extremely well. This way, nobody does error handling except the application or a dedicated error-handling middleware. Middlewares don't even have to call an error handling function - they just [immediately return a status code and the error](https://github.com/mholt/caddy/blob/e42c6bf0bb00d2e5e966ec7d9923eb21627a6b74/middleware/fastcgi/fastcgi.go#L70).

This keeps error handling consistent, customizable, and reliable.


## Startup and Configuration

The first work on Caddy wasn't on the program itself. Rather, it was on its input. Long before Caddy even had a name, I planned how the user would configure it. Extensibility and a clean syntax were important. No semicolons, parentheses, or angle brackets were allowed. Non-programmers are going to use this, after all.

Caddy includes a robust, custom parser to make sense of the Caddyfile. When you run Caddy, the first thing it does is [configure itself](https://github.com/mholt/caddy/blob/e42c6bf0bb00d2e5e966ec7d9923eb21627a6b74/config/config.go#L26) based on the contents of the Caddyfile.

Caddy's parsing routine is... unconventional. First, the file is read into tokens. The only thing the [core parser](https://github.com/mholt/caddy/tree/master/config/parse) does is organize tokens by server address. Most of the parsing happens in another package called [setup](https://github.com/mholt/caddy/tree/master/config/setup), which sets up each middleware according to the tokens. By the time the resulting configuration is passed back up and out of the [config package](https://github.com/mholt/caddy/tree/master/config), Caddy has everything it needs to start serving.

This architecture makes for quite a few files, but it separates concerns. It makes it easy to extend Caddy and put it under test.


## Routing

Caddy doesn't use a conventional HTTP router. Instead, all requests take the same path through the middleware chain. If the request path matches what that middleware is configured for, the middleware activates and does its thing. Otherwise, it passes the request through to the next handler. For [example](https://github.com/mholt/caddy/blob/e42c6bf0bb00d2e5e966ec7d9923eb21627a6b74/middleware/headers/headers.go#L24):

    func (m MyMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
        if middleware.Path(r.URL.Path).Matches(m.BasePath) {
            // do something
        } else {
            // pass-thru
            return m.Next.ServeHTTP(w, r)
        }
    }

Some directives do not accept a base path and will run on every request. Others with an optional path argument will default to "/" (matching every request) if the base path is omitted.


## Virtual Hosting

I said before that there isn't any routing. That's only true for the request's _path_. There is, in fact, some routing on the _host_. Caddy can serve multiple sites (each with their own hostname) on the same port. The obvious problem is that only one listener can bind to, say, port 80. To work around this, Caddy will start one listener on port 80 and then multiplex requests from there [based on the value of the Host header](https://github.com/mholt/caddy/blob/e42c6bf0bb00d2e5e966ec7d9923eb21627a6b74/server/server.go#L165-L190).
    
<script type="text/javascript" src="https://sourcegraph.com/R$3104360@master===535f95668282b829ed8f8b2c56e9576e1136e3cf/.tree/server/server.go/.sourcebox.js?StartLine=177&EndLine=190"></script>


## Server Name Indication

You can serve multiple plaintext sites on the same port with Caddy, but what about multiple HTTPS sites on port 443? It's impossible without an extension to TLS called SNI, or Server Name Indication. Without it, a TLS handshake comes in from a client, but the server has no idea which key to use to complete the handshake because the Host information is encrypted! This prevents the TLS handshake from ever succeeding.

SNI solves this problem. It's actually built into the Go standard library, but you can't use it with a call to the usual `http.ListenAndServeTLS()`. You have to [roll your own](https://github.com/mholt/caddy/blob/e42c6bf0bb00d2e5e966ec7d9923eb21627a6b74/server/server.go#L123-L133), which [isn't hard](https://groups.google.com/d/msg/golang-nuts/rUm2iYTdrU4/PaEBya4dzvoJ). Basically, you just add each cert/key pair and attach them to their host names:

<script type="text/javascript" src="https://sourcegraph.com/R$3104360@535f95668282b829ed8f8b2c56e9576e1136e3cf===535f95668282b829ed8f8b2c56e9576e1136e3cf/.tree/server/server.go/.sourcebox.js?StartLine=123&EndLine=133"></script>

SNI is not something a Caddy user needs to think about. It just works.


## Example

The nginx.conf file for all the Gopher Academy websites was over 115 lines long. The equivalent Caddyfile is only 50 lines.

This Caddyfile serves the GopherCon website:

    http://gophercon.com, http://www.gophercon.com {
        root /var/www/gc15/public
        gzip
    }

Gophercon.com is generated by Hugo, so the static HTML files live in the gc15/public folder. Let's use the `git` directive to deploy the site when we `git push`:

    http://gophercon.com, http://www.gophercon.com {
        root /var/www/gc15/public
        gzip
        git {
            repo  https://github.com/gophercon/gc15
            path  ../
            then  hugo --theme=gophercon --destination=public
        }
    }

When the server starts, it pulls the entire repository into the folder above the site root, then runs `hugo` to generate the site, placing it in the "public" folder. Every hour, the latest is pulled and the site is re-generated. (A future release will allow immediate pulls via post-commit hook.)

So basically, each Gopher Academy site is served and deployed using a 9-line file that's easy to read and intuitive to write. (In reality, the files are combined into one, but you can do what you want.)

After the Caddyfile is prepared, we just run `caddy` in the same directory as the Caddyfile and we're done. (Initially, we forgot to raise `ulimit -n` to a value safe for a production website. Caddy showed a warning that the file descriptor limit was too low and recommended raising it. Phew!)


## Next Steps

We're working on an API that can change the server's configuration while it's running. And with that, an API client that will allow you to log in to your server and make changes and see requests in real-time.


## Conclusion

**Thank you** to all [contributors](https://github.com/mholt/caddy/graphs/contributors) so far.

I hope this was interesting to you! [Give Caddy a try](https://caddyserver.com/download) and let us know what you think. You can also reach out to me directly on Twitter [@mholt6](https://twitter.com/mholt6).
