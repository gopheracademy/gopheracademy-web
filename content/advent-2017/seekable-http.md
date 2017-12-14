+++
author = ["Jeff R. Allen"]
title = "Seeking around in an HTTP object"
linktitle = "Seeking aroud in an HTTP object"
date = "2017-12-16T00:00:00Z"
series = ["Advent 2017"]
+++

Imagine there's a giant ZIP file on a HTTP server, and you want to
know what's inside it. You don't know if it's got what you are looking
for, and you don't want to download the whole thing. Is it possible to
do something like `unzip -l https://example.com/giant.zip`?

This is not a theoretical problem just to demonstrate something in
Go. In fact, I wasn't looking to write an article at all, except that
I wanted to know the structure of the <a
href="https://bulkdata.uspto.gov/data/patent/officialgazette/2017/">bulk
patent downloads from the US Patent and Trademark Office
(USPTO)</a> from those ZIP files. Or, I thought, how cool would it be
to be able to fetch <a
href="https://bulkdata.uspto.gov/data/patent/grant/multipagepdf/1790_1999/">individual
images of some of the patents issued in 1790</a> out of these
tarfiles?

Go take a look. There's hundreds of huge ZIP and tarfiles to explore there!

The <a
href="https://pkware.cachefly.net/webdocs/casestudies/APPNOTE.TXT">ZIP
file format</a> has a table of contents in it, located at the end. So
on local disk, "unzip -l" is as simple as "seek to the end, find the
TOC, parse it and print it". And in fact, we can see that's how Go is
going to proceed, because the <a
href="https://godoc.org/archive/zip#NewReader">zip.NewReader
function</a> wants a file it <a
href="https://godoc.org/io#ReaderAt">will be able to seek around
in</a>. As for TAR files, they were designed for a time when tape
streamed and memory was scarce, so their table of contents is
interleaved among the files themselves.

But we're not on local disk, we've given ourselves the challenge of
reading from a URL. What to do? Where to start?

We've got a couple things to check and then we can plan a way
forward. The equivalent of seeking and reading from a file for HTTP is
the Range header. So, do the USPTO servers support <a
href="https://tools.ietf.org/html/rfc7233">the Range header</a>?
That's easy enough to check using curl and an HTTP HEAD request:

```
$ curl -I https://bulkdata.uspto.gov/data/patent/officialgazette/2017/e-OG20170103_1434-1.zip
HTTP/1.1 200 OK
Date: Mon, 11 Dec 2017 21:10:26 GMT
Server: Apache
Last-Modified: Tue, 03 Jan 2017 11:58:45 GMT
ETag: "afb8ac8-5452f63e0a82f"
Accept-Ranges: bytes
Content-Length: 184257224
X-Frame-Options: DENY
Content-Type: application/zip
```

Note the "Accept-Ranges" header in there, which says that we can send
byte ranges to it. Range headers let you implement the HTTP equivalent
of the operating system's random-access reads (i.e. the <a
href="https://godoc.org/io#ReaderAt">io.ReaderAt</a> interface).

So it would theoretically be possible to pick and choose which bytes
we download from the web server, in order to download only the parts
of a file that have the metadata (table of contents) in it.

Now we need an implementation of the ZIP file format that will let us
replace the "read next table of contents header" part with an
implementation of read that reads only the metadata, using an HTTP GET
with a Range header. And that is where Go's <a
href="https://golang.org/pkg/archive/zip">archive/zip</a> and <a
href="https://godoc.org/archive/tar">archive/tar</a> packages come in!

As we've already noted, <a
href="https://godoc.org/archive/zip#NewReader">zip.NewReader</a> is
chomping at the bit to start seeking. However as we take a look at
TAR, we find a problem. The <a
href="https://golang.org/pkg/archive/tar/#NewReader">tar.NewReader</a>
method takes an io.Reader. The problem with the io.Reader is that it
does not let us get random access to the resource, like io.ReaderAt
does. It is implemented that way because it makes the tar package more
adaptable. In particular, you can hook the Go tar package directly up
to the <a
href="https://golang.org/pkg/compress/gzip/">compress/gzip</a> package
and read tar.gz files -- as long as you are content to read them
sequentially and not jump around in them, as we wish to.

So what to do? Use the source, Luke! Go dig into the <a
href="https://github.com/golang/go/blob/c007ce824d9a4fccb148f9204e04c23ed2984b71/src/archive/tar/reader.go#L88">Next
method</a>, and look around. That's where we'd expect it to go find
the next piece of metadata. Within a few lines, we find an intriguing
function call, to <a
href="https://github.com/golang/go/blob/c007ce824d9a4fccb148f9204e04c23ed2984b71/src/archive/tar/reader.go#L407">skipUnread</a>. And there, we find something very interesting:

```go
// skipUnread skips any unread bytes in the existing file entry, as well as any alignment padding.
func (tr *Reader) skipUnread() {
  nr := tr.numBytes() + tr.pad // number of bytes to skip
  tr.curr, tr.pad = nil, 0
  if sr, ok := tr.r.(io.Seeker); ok {
    if _, err := sr.Seek(nr, os.SEEK_CUR); err == nil {
      return
    }
  }
  _, tr.err = io.CopyN(ioutil.Discard, tr.r, nr)
}

// Note: This is from Go 1.4, which had a simpler skipUnread than go 1.9 does.
```

The type assertion in there says, "if the io.Reader is actually
capable of seeking as well, then instead of reading and discarding, we
seek directly to the right place". Eureka! We just need to send an
io.Reader into tar.NewReader that also satisfies <a
href="https://golang.org/pkg/io/#Seeker">io.Seeker</a> (thus, it is an
<a href="https://golang.org/pkg/io/#ReadSeeker">io.ReadSeeker</a>).

So, now go check out package <a
href="https://godoc.org/github.com/jeffallen/seekinghttp">github.com/jeffallen/seekinghttp</a>
which is, as it's name implies, a package for seeking around in HTTP
objects (<a
href="https://github.com/jeffallen/seekinghttp">source on Github</a>).

This package <a href="https://github.com/jeffallen/seekinghttp/blob/master/seekinghttp.go#L26">implements</a>
not only io.ReadSeeker, but also
io.ReaderAt. Why? Because, as I mentioned above, reading a ZIP file
requires an io.ReaderAt. It also needs the length of the file passed
to it, so that it can look at the end of the file for the table of
contents. The HTTP HEAD method works nicely to get the Content-Length
of the HTTP object, without downloading the entire thing.

A command-line tool to get the table of contents of tar and zip files
remotely is in <a
href="https://github.com/jeffallen/seekinghttp/tree/master/cmd/remote-archive-ls">remote-archive-ls</a>. Turn
on the "-debug" option in order to see what's happening. It is
fascinating to watch as the TAR or ZIP readers from Go's standard
library "call back" into our code and ask for a few bytes here, a few
bytes there.

Soon after I first got this program working, I found a serious flaw. Here's an example run:
```bash
$ ./remote-archive-ls -debug 'https://bulkdata.uspto.gov/data/patent/grant/multipagepdf/1790_1999/grant_pdf_17900731_18641101.tar'
2017/12/12 00:07:38 got read len 512
2017/12/12 00:07:38 ReadAt len 512 off 0
2017/12/12 00:07:38 Start HTTP GET with Range: bytes=0-511
2017/12/12 00:07:39 HTTP ok.
File: 00000001-X009741H/
2017/12/12 00:07:39 got read len 512
2017/12/12 00:07:39 ReadAt len 512 off 512
2017/12/12 00:07:39 Start HTTP GET with Range: bytes=512-1023
2017/12/12 00:07:39 HTTP ok.
File: 00000001-X009741H/00/
2017/12/12 00:07:39 got read len 512
2017/12/12 00:07:39 ReadAt len 512 off 1024
2017/12/12 00:07:39 Start HTTP GET with Range: bytes=1024-1535
2017/12/12 00:07:39 HTTP ok.
File: 00000001-X009741H/00/000/
2017/12/12 00:07:39 got read len 512
2017/12/12 00:07:39 ReadAt len 512 off 1536
2017/12/12 00:07:39 Start HTTP GET with Range: bytes=1536-2047
2017/12/12 00:07:39 HTTP ok.
File: 00000001-X009741H/00/000/001/
2017/12/12 00:07:39 got read len 512
2017/12/12 00:07:39 ReadAt len 512 off 2048
2017/12/12 00:07:39 Start HTTP GET with Range: bytes=2048-2559
2017/12/12 00:07:39 HTTP ok.
File: 00000001-X009741H/00/000/001/us-patent-image.xml
2017/12/12 00:07:39 got seek 0 1
2017/12/12 00:07:39 got seek 982 1
2017/12/12 00:07:39 got read len 42
2017/12/12 00:07:39 ReadAt len 42 off 3542
2017/12/12 00:07:39 Start HTTP GET with Range: bytes=3542-3583
2017/12/12 00:07:39 HTTP ok.
2017/12/12 00:07:39 got read len 512
2017/12/12 00:07:39 ReadAt len 512 off 3584
2017/12/12 00:07:39 Start HTTP GET with Range: bytes=3584-4095
2017/12/12 00:07:39 HTTP ok.
File: 00000001-X009741H/00/000/001/00000001.pdf
2017/12/12 00:07:39 got seek 0 1
2017/12/12 00:07:39 got seek 320840 1
2017/12/12 00:07:39 got read len 184
2017/12/12 00:07:39 ReadAt len 184 off 324936
...etc...
```

Can you see the problem? That's a whole lot of HTTP transactions! The
TAR reader is working through the TAR stream a little bit at a time,
issuing a stream of tiny reads. All of those short HTTP transactions
are hard on the server, and terrible for our throughput, since each
one entails many round-trips to the server.

The solution, of course, it caching. Instead of reading just the first
512 bytes that the TAR reader asks for, I read 10 times that many, so
that the next several reads will be serviced directly from cache. If
there is a read that is outside of the cache, we assume that the other
reads will come in that area as well, and drop the entire current
cache, in order to fill it with 10x the current read amount from the
new current offset.

The fact that the TAR reader sends a lot of small reads points out
something very important about buffering. Giving the results of <a
href="https://godoc.org/os#Open">os.Open</a> directly to tar.NewReader
isn't very clever, especially if your plan is to jump through the file
looking for metadata. While it is true that *os.File implements
io.ReadSeeker, we now know that TAR is going to issue a huge number of
small system calls to the kernel. The solution, much like caching was
the solution above, might be to use the <a
href="https://godoc.org/bufio">bufio</a> package to buffer the
*os.File, so that the small reads that TAR is issuing will come out of
RAM instead of going to the operating system. But be careful: is it
really the solution? Does <a
href="https://godoc.org/bufio#Reader">bufio.Reader</a> really
implement io.ReadSeeker and io.ReadAt like we need? (Spoiler: it
doesn't; maybe one of you readers would like to show us how to speed
up Go's tar using a replacement for bufio next advent season?)

I hope you have enjoyed this little trip through the standard library
and HTTP, taking a look at how to work together with the standard
library in order to help it achieve more than it was was written to
do, so that it can help you get your job done. When you implement
io.Reader and friends, you get a chance to go behind the scenes of the
libraries you are calling into, and feed them data from places their
authors never expected!
