+++
author = ["Jiahua Chen"]
date = "2014-12-06T08:00:00+00:00"
title = "Macaron: Martini-style, but faster and cheaper"
series = ["Advent 2014"]
+++

[Macaron](https://github.com/Unknwon/macaron) is a high productive and modular design web framework in Go. It takes basic ideology of Martini and extends in advance.

## Why another web framework? 

The story began with the [Gogs](https://github.com/gogits/gogs) project, it first uses Martini as its web framework, worked quite well. Soon after, our team found that Martini is good but too minimal, also too many reflections that cause performance issue. Finally, I came up an idea that why don't we just integrate most frequently used middlewares as interfaces(huge reduction for reflection), and replace default router layer with faster one. It turns out Macaron actually requires [less memory overhead for every request and faster speed than Martini](https://github.com/Unknwon/go-http-routing-benchmark), and at the same time, it reserves my favorite Martini-style coding.

## Examples

If you are using Martini already, you shouldn't be surprised with following code:

```go
package main

import "github.com/Unknwon/macaron"

func main() {
    m := macaron.Classic()
    m.Get("/", func() string {
        return "Hello world!"
    })
    m.Run()
}
```

And here is another simple example:

```go
package main

import (
    "log"
    "net/http"

    "github.com/Unknwon/macaron"
)

func main() {
    m := macaron.Classic()
    m.Get("/", myHandler)

    log.Println("Server is running...")
    log.Println(http.ListenAndServe("0.0.0.0:4000", m))
}

func myHandler(ctx *macaron.Context) string {
    return "the request path is: " + ctx.Req.RequestURI
}
```

## How about middlewares?

There are already many [middlewares](https://github.com/macaron-contrib) to simplify your work:

- [binding](https://github.com/macaron-contrib/binding) - Request data binding and validation
- [i18n](https://github.com/macaron-contrib/i18n) - Internationalization and Localization
- [cache](https://github.com/macaron-contrib/cache) - Cache manager
- [session](https://github.com/macaron-contrib/session) - Session manager
- [csrf](https://github.com/macaron-contrib/csrf) - Generates and validates csrf tokens
- [captcha](https://github.com/macaron-contrib/captcha) - Captcha service
- [pongo2](https://github.com/macaron-contrib/pongo2) - Pongo2 template engine support
- [sockets](https://github.com/macaron-contrib/sockets) - WebSockets channels binding
- [bindata](https://github.com/macaron-contrib/bindata) - Embed binary data as static and template files
- [toolbox](https://github.com/macaron-contrib/toolbox) - Health check, pprof, profile and statistic services
- [oauth2](https://github.com/macaron-contrib/oauth2) - OAuth 2.0 backend
- [switcher](https://github.com/macaron-contrib/switcher) - Multiple-site support
- [method](https://github.com/macaron-contrib/method) - HTTP method override
- [permissions2](https://github.com/xyproto/permissions2) - Cookies, users and permissions
- [renders](https://github.com/macaron-contrib/renders) - Beego-like render engine(Macaron has built-in template engine, this is another option)

Want to know more about Macaron? Check out its [documentation site](http://macaron.gogs.io/)!
