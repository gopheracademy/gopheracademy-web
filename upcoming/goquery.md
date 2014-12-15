+++
author = ["Martin Angers"]
date = "2014-12-12T00:00:00-08:00"
title = "goquery: a little like that j-thing"
series = ["Advent 2014"]
+++

A little over 2 and a half years ago I started playing with that new language called Go. Coming mostly from .NET and node.js, I was at first intrigued by its concurrency features and its lack of object inheritance, and impressed by the quality of the team behind it. Fast-forward to today and Go is now my go-to (oh please), day-to-day language, and I'm lucky enough to use it both at work at [splice][splice] and in my [personal projects][github].

The first open-source project I created with Go is also my most popular one to this day, having crossed the 1000 stars milestone on GitHub just a few weeks ago: [goquery][goquery]. Back then I thought it might be useful to have a convenient and well-known API to manipulate HTML documents server-side, and I was hoping other people might like it too. Never in my wildest dreams had I hoped it would become *that* popular!

## Sowing The Seeds

Right from the start, I decided to mimic the API of jQuery. The reason was simple, jQuery being the ubiquitous library that even influenced the W3C selectors API, it seemed like a solid base. Much like Go's `fmt` package continued the C tradition of the `printf` family, `goquery` would perpetuate jQuery's heritage. And a large part of jQuery's success is its chainability, so in Go too, you can write something like this:

    // res being an *http.Response
    doc, err := goquery.NewDocumentFromResponse(res)

    doc.Find("div.container").Has("b").Each(func (i int, s *goquery.Selection) {
        fmt.Println(s.Text())
    })

However, jQuery's functions are heavily overloaded and I did not want to end up with a bunch of methods that accepted variadic empty interfaces as arguments, losing all of Go's static typing goodness. Since Go does not support overloaded methods, I came up with a naming convention derived from jQuery's original function names so that it is easy to infer the correct name for someone that already knows jQuery. This approach was inspired by the standard library's `regexp` package and the naming convention is detailed in the [project's readme file][naming].

Unlike a javascript library though, this package is not loaded as part of a DOM document, so there are two major differences with jQuery's API:

* The HTML document to manipulate must be explicitly loaded, via one of the `goquery.NewDocument*` functions;
* The DOM's stateful manipulation methods (`height`, `css` *et al.*) have been left off as they don't make much sense without a live DOM.

There are only three types exported by the package, `Document` to represent the loaded HTML document, `Selection` that holds most of the API methods, and `Matcher`, an interface that defines the required selector engine's methods. By default, goquery uses [cascadia][cascadia] as its selector engine but thanks to this interface, other implementations can be used.

## If I Could Turn Back Time

Being my first serious Go endeavour at the time, I was still learning idiomatic Go and as such, there are things I wish were done differently, but for API stability's sake I've kept the way they are.

Chief among those things is the fact that when a selection string is used (e.g. `doc.Find(".someclass")`), it calls cascadia's `MustCompile` under the hood. Of course, this is not the most efficient thing to do as it may recompile many times the same selection string, but perhaps more importantly, as experienced gophers will know, `Must*` means it will panic if it fails to parse the string. The `Must*` idiom usually exists for things that should be parsed or otherwise created at initialization time (a package-level variable initialization, a package-level `init` function, or somewhere in `main` before the actual work), where a panic is a reasonable thing to do before the process starts whatever it has to do.

This is the reason the `*Matcher` overloads have been added to the package recently - to allow users of the package to safely compile the selectors outside goquery and use the compiled version subsequently, in place of the selection strings:

    // So instead of:
    doc.Find(".someclass")

    // You can do, in a package-level declaration block (using
    // cascadia or any selector library that implements goquery.Matcher):
    var matcher = cascadia.MustCompile(".someclass")

    // Or dynamically, handling parsing errors as required:
    matcher, err := cascadia.Compile(someVar)
    if err != nil {
        // handle error
    }
    
    // ... and then when needed:
    doc.FindMatcher(matcher)

Another thing that bugs me is that the `goquery.Selection` struct is exported instead of an interface. I don't think there is much value to have this type exported, as some fields are private anyway and selections are created via the API methods - I don't see a valid use-case where you'd want to create it directly. I think interfaces would've been better for both the Selection and the Document, and the Document would've implemented the Selection interface too (although the excellent points made by Dave Cheney in [this blog post][dave] should be taken into consideration when thinking about exporting interfaces in lieu of structs).

Finally, the naming could've been better and shorter. I would've preferred `goquery.New` to `goquery.NewDocument`, as it is the most obvious (and ideally only) thing that should be created with this package. The other overloaded *constructors* would've followed suit. The naming convention could've benefitted from shorter names too, such as `FilterFunc` instead of `FilterFunction` to match Go's terse `func` keyword (and stdlib's convention, such as `regexp.ReplaceAllFunc`). `golint` also complains every time I commit because I used the field name `Url` instead of `URL` and `Html` instead of `HTML`. So please, take note and don't repeat my mistakes in your APIs! Go's [style guide][style] is a good reference, and running `golint` on your code a great habit to take (as is `go vet`).

## Come Together

I can't talk about goquery without mentioning the shoulders of giants upon which it stands. I've briefly talked about [cascadia][cascadia], this is an excellent package that can certainly be used directly in many cases where the higher-level API of goquery is not required.

Then there's the awesome [html][html] package in the `go.net` repository, an HTML5 parser. This is the building block of both cascadia and goquery.

Finally, some contributors helped make the package what it is today. In particular, [Andrew Stone][stone] pushed some nice pull requests to add manipulation functions such as `AddClass`, `SetAttr`, `Wrap` and the likes, so the HTML document can now be modified via goquery.

If you don't see your favorite jQuery function or simply want to help maintain the package, pull requests are always welcome!

[splice]: https://splice.com/
[github]: https://github.com/PuerkitoBio
[goquery]: https://github.com/PuerkitoBio/goquery
[naming]: https://github.com/puerkitobio/goquery#api
[cascadia]: https://code.google.com/p/cascadia/
[dave]: http://blog.gopheracademy.com/advent-2014/nigels-webdav-package/
[html]: http://godoc.org/golang.org/x/net/html
[style]: https://github.com/golang/go/wiki/CodeReviewComments
