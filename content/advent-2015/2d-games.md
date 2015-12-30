+++
author = ["Gregory Roseberry"]
date = "2015-12-30T00:00:00Z"
series = ["Advent 2015"]
title = "2D Game Libraries for Go"
+++

By now, we all know that Go is a great language for writing servers and command line tools. But what about games? Can you make desktop, web, or mobile games in Go too? The answer is yes, but not necessarily all at once... yet!

Last summer, I set up shop at [Comiket 88](https://en.wikipedia.org/wiki/Comiket) and managed to sell a few copies of [HOT PLUG](http://hotplug.kawaii.solutions), a simple 2D action game written in Go for OS X and Windows. This article explores some options for making 2D games in Go. No OpenGL knowledge required.

## Desktop

It's surprisingly easy to get a cross-platform game started in Go. All of these libraries install their dependencies automatically with `go get`, making development a breeze. 

### engi family

**[ajhager/engi](https://github.com/ajhager/engi)** is one of the oldest Go game libraries, and the one I used to get my game started. It's a very simple library whose API reminds me a little bit of Microsoft's abandoned XNA. engi is essentially a wrapper around [GLFW](http://www.glfw.org) with some handy functions for drawing images, taking input, and keeping track of time. Although it has essentially zero documentation, it's not too hard to start with one of the examples and make it into a real game. It supports Windows, OS X, Linux, and web browsers via GopherJS. Unfortunately, just like XNA, ajhager/engi seems to be abandoned. 

**[guregu/engi](https://github.com/guregu/engi)** is my quick and dirty fork of ajhager/engi. The good news is that I hacked in something very important missing in the original: audio support. The bad news is that adding audio broke web support, so this library is desktop-only for now. This fork also includes some minor changes to the version of GLFW it uses (no more broken VSync in Windows!) and some other improvements with the assets loading system. Originally I planned to merge everything back upstream, but now I'm considering maintaining this as a full-fledged project. HOT PLUG uses this library, and you can find lots of interesting and increasingly desperate commits as the deadline for Comiket was approaching.


**[paked/engi](https://github.com/paked/engi)** is another fork of engi worth keeping an eye on if you fancy Entity Component Systems (ECS). paked's fork of engi is more "batteries included", with sprite sheets and cameras and Tiled map support, all tied into the ECS. Audio support was recently added but doesn't work on Windows yet. Web browsers are also unsupported at the moment. This could be the most actively developed library of all of them I will introduce in this article.

### Ebiten

**[Ebiten](http://hajimehoshi.github.io/ebiten/)** is another GLFW wrapper library, but independent from the engi lineage. Ebiten includes tons of cool stuff like controller support, web browser support, and filters for images. It also has experimental audio support, including web browser support for it! This library is under relatively active development and is definitely worth checking out.

## Mobile

Official support for [Go on mobile devices](https://github.com/golang/mobile) has come a long way and only continues to improve. Check out Andrew Gerrand's [Flappy Gopher](https://github.com/adg/game), an open-source full-Go mobile game. He recently gave a talk at GoCon Winter 2015 in Tokyo, walking through the entire commit history. All the commits are broken into very easy to understand chunks, so it's a great example of how to make a 2D game for mobile devices in Go from scratch. Just start at the beginning!

The mobile packages work on Linux and OS X, but [not on Windows](https://github.com/golang/go/issues/9306). Once Windows support lands (soon!) the mobile packages could become the best way to develop a cross-platform game in Go. Add on a [WebGL implementation](https://github.com/goxjs/gl) and it could become the perfect tool for Go game development. 

## Why Go?

With any of the libraries mentioned above, you could get started on a Go game right now. But what are the advantages of using Go? 

Having your whole development environment fully set up by a simple `go get` is awesome.

Go's type system to be conducive to making games. You can use interfaces to define what some might call a "component", and have various game objects (structs) implment those interfaces by embedding common functionality. Here is [some example code](https://go-talks.appspot.com/github.com/guregu/slides/comiket/comiket.slide#20).

Ultimately, why not? I could tell you all about how Go has compiles fast and has cool concurrency primitives, but you already know that. If you enjoy working with Go, a game is a fun hobby project to expand your horizons. 

If you'd like to read more about the process of making my game and selling it at Comiket and see some example code, check out [this presentation](https://go-talks.appspot.com/github.com/guregu/slides/comiket/comiket.slide). 

Go isn't just for servers, let's make some games!