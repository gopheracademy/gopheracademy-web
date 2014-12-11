+++
author = ["Tommi Virtanen"]
date = "2014-12-11T08:00:00+00:00"
title = "Writing file systems in Go with FUSE"
series = ["Advent 2014"]
+++


# Motivation

Some time ago, I decided I wanted to solve my own storage needs
better, and I realized that I can't just rely on synchronizing files.
I needed a *filesystem* that combines the best of three worlds: local
files, network file systems, and file synchronization. This project is
called [Bazil](http://bazil.org/), as in bazillion bytes.

To make [Bazil](http://bazil.org/) possible, I needed to be able to
easily write a filesystem in Go. And now you can, too, with
[bazil.org/fuse](http://bazil.org/fuse).

What we'll build today is an example Go application that serves a Zip
archive as a filesystem:

``` console
$ unzip -v archive.zip
Archive:  archive.zip
 Length   Method    Size  Cmpr    Date    Time   CRC-32   Name
--------  ------  ------- ---- ---------- ----- --------  ----
       0  Stored        0   0% 2014-12-11 04:03 00000000  buried/
       0  Stored        0   0% 2014-12-11 04:03 00000000  buried/deep/
       5  Stored        5   0% 2014-12-11 04:03 2efcceec  buried/deep/loot
      13  Stored       13   0% 2014-12-11 04:03 f4247453  greeting
--------          -------  ---                            -------
      18               18   0%                            4 files
$ zipfs archive.zip mnt &
$ tree mnt
mnt
├── buried
│   └── deep
│       └── loot
└── greeting

2 directories, 2 files
$ cat mnt/greeting
hello, world
```

# FUSE

[FUSE](https://git.kernel.org/cgit/linux/kernel/git/torvalds/linux.git/tree/Documentation/filesystems/fuse.txt)
(Filesystem In Userpace) is a Linux kernel filesystem that sends the
incoming requests over a file descriptor to userspace. Historically,
these have been served with a
[C library of the same name](http://fuse.sourceforge.net/), but
ultimately FUSE is just a protocol. Since then, the protocol has been
implemented for other platforms such as OS X, FreeBSD and OpenBSD.

[bazil.org/fuse](http://bazil.org/fuse) is a reimplementation of that
protocol in pure Go.


# Structure of Unix filesystems

Unix filesystems consist of *inodes* ("index nodes"). These nodes are
files, directories, etc. *Directories* contain *directory entries*
(*dirent*) that point to child *inodes*. A directory entry is
identified by its name, and carries very little metadata. The *inode*
manages both the metadata (including things like access control) and
the content of the file.

Open files are identified in userspace with *file descriptors*, which
are just safe references to kernel objects known as *handles*.


# Go API

Our FUSE library is split into two parts. The low-level protocol is in
[`bazil.org/fuse`](http://godoc.org/bazil.org/fuse) while the
higher-level, optional, state machine keeping track of object
lifetimes is
[`bazil.org/fuse/fs`](http://godoc.org/bazil.org/fuse/fs).

Each file system has a *root entry*. The interface
[`fs.FS`](http://godoc.org/bazil.org/fuse/fs#FS) has a method
[`Root`](http://godoc.org/bazil.org/fuse/fs#FS.Root) that returns an
[`fs.Node`](http://godoc.org/bazil.org/fuse/fs#Node).

To access a file (see its metadata, open it, etc), the kernel looks it
up by name by sending a
[`fuse.LookupRequest`](http://godoc.org/bazil.org/fuse#LookupRequest)
to the FUSE server, stating the parent directory and basename. This
request is served by a
[`Lookup`](http://godoc.org/bazil.org/fuse/fs#NodeRequestLookuper)
method on the parent
[`fs.Node`](http://godoc.org/bazil.org/fuse/fs#Node). The method
returns an [`fs.Node`](http://godoc.org/bazil.org/fuse/fs#Node), and
the result is cached in the kernel and reference counted. Dropping a
cache entry sends a
[`ForgetRequest`](http://godoc.org/bazil.org/fuse#ForgetRequest), and
when the reference count reaches zero,
[`Forget`](http://godoc.org/bazil.org/fuse/fs#NodeForgetter) gets
called.

Files are renamed with
[`Rename`](http://godoc.org/bazil.org/fuse/fs#NodeRenamer), deleted
with [`Remove`](http://godoc.org/bazil.org/fuse/fs#NodeRemover), and
so on.

Kernel file *handles* are created for example by opening a file.
Opening an existing file sends an
[`OpenRequest`](http://godoc.org/bazil.org/fuse#OpenRequest), you
guessed it, served by
[`Open`](http://godoc.org/bazil.org/fuse/fs#NodeOpener). All methods
creating new handles return a
[`Handle`](http://godoc.org/bazil.org/fuse/fs#Handle). Handles are
closed by a combination of
[`Flush`](http://godoc.org/bazil.org/fuse/fs#HandleFlusher) and
[`Release`](http://godoc.org/bazil.org/fuse/fs#HandleReleaser).

The default [`Open`](http://godoc.org/bazil.org/fuse/fs#NodeOpener)
action, if the method is not implemented, is to use the
[`fs.Node`](http://godoc.org/bazil.org/fuse/fs#Node) also as a
[`Handle`](http://godoc.org/bazil.org/fuse/fs#Handle); this tends to
work well for stateless read-only files.

Reads from a [`Handle`](http://godoc.org/bazil.org/fuse/fs#Handle) are
served by [`Read`](http://godoc.org/bazil.org/fuse/fs#HandleReader),
writes with
[`Write`](http://godoc.org/bazil.org/fuse/fs#HandleWriter), and apart
from all the extra data available these look similar to
[`io.ReaderAt`](http://golang.org/pkg/io/#ReaderAt) and
[`io.WriterAt`](http://golang.org/pkg/io/#WriterAt). Note that file
size changes via
[`Setattr`](http://godoc.org/bazil.org/fuse/fs#NodeSetattrer), not
based on [`Write`](http://godoc.org/bazil.org/fuse/fs#HandleWriter),
and [`Attr`](http://godoc.org/bazil.org/fuse/fs#Node) needs to return
the correct [`Size`](http://godoc.org/bazil.org/fuse#Attr.Size).

Listing a directory happens by reading an open file handle that is a
directory. Instead of file contents, the read returns marshaled
directory entries. The
[`ReadDir`](http://godoc.org/bazil.org/fuse/fs#HandleReadDirer) method
implements a slightly higher-level API, where you return a slice of
directory entries.

And so on. Learning to write a file system requires a decent
understanding of the kernel data structures and their state changes
on an abstract level, but the actual Go parts of it are quite simple.
So let's dive into the code.


# `zipfs`

As our example project, we'll write a filesystem that shows a
read-only view of the contents of a
[Zip archive](http://golang.org/pkg/archive/zip/).

The full source code is available at
https://github.com/bazillion/zipfs

## Skeleton

Let's start with a skeleton with a argument parsing:

``` go
package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

// We assume the zip file contains entries for directories too.

var progName = filepath.Base(os.Args[0])

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", progName)
	fmt.Fprintf(os.Stderr, "  %s ZIP MOUNTPOINT\n", progName)
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(0)
	log.SetPrefix(progName + ": ")

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 2 {
		usage()
		os.Exit(2)
	}
	path := flag.Arg(0)
	mountpoint := flag.Arg(1)
	if err := mount(path, mountpoint); err != nil {
		log.Fatal(err)
	}
}
```

Mounting is a bit cumbersome due to OSXFUSE behaving very differently
from Linux; there are several stages where errors may show up.

``` go
func mount(path, mountpoint string) error {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer archive.Close()

	c, err := fuse.Mount(mountpoint)
	if err != nil {
		return err
	}
	defer c.Close()

	filesys := &FS{
		archive: &archive.Reader,
	}
	if err := fs.Serve(c, filesys); err != nil {
		return err
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		return err
	}

	return nil
}
```

## Filesystem

On to the actual file system. We just hold a pointer to the zip
archive:

``` go
type FS struct {
	archive *zip.Reader
}
```

And we need to provide the `Root` method:

``` go
var _ fs.FS = (*FS)(nil)

func (f *FS) Root() (fs.Node, fuse.Error) {
	n := &Dir{
		archive: f.archive,
	}
	return n, nil
}
```

## Directories

Zip files contain a list of files, but typical zip archivers include
entries for the directories, with a name ending in a slash. We rely on
this behavior later.

Let's define our `Dir` type, and implement the mandatory `Attr`
method. We use the `*zip.File` to serve directory metadata.

``` go
type Dir struct {
	archive *zip.Reader
	// nil for the root directory, which has no entry in the zip
	file *zip.File
}

var _ fs.Node = (*Dir)(nil)

func zipAttr(f *zip.File) fuse.Attr {
	return fuse.Attr{
		Size:   f.UncompressedSize64,
		Mode:   f.Mode(),
		Mtime:  f.ModTime(),
		Ctime:  f.ModTime(),
		Crtime: f.ModTime(),
	}
}

func (d *Dir) Attr() fuse.Attr {
	if d.file == nil {
		// root directory
		return fuse.Attr{Mode: os.ModeDir | 0755}
	}
	return zipAttr(d.file)
}
```

## Directory entry lookup

For our filesystem to contain anything useful, we need to be able to
find entries by name. We just iterate over the zip entries, matching
paths:

``` go
var _ = fs.NodeRequestLookuper(&Dir{})

func (d *Dir) Lookup(req *fuse.LookupRequest, resp *fuse.LookupResponse, intr fs.Intr) (fs.Node, fuse.Error) {
	path := req.Name
	if d.file != nil {
		path = d.file.Name + path
	}
	for _, f := range d.archive.File {
		switch {
		case f.Name == path:
			child := &File{
				file: f,
			}
			return child, nil
		case f.Name[:len(f.Name)-1] == path && f.Name[len(f.Name)-1] == '/':
			child := &Dir{
				archive: d.archive,
				file:    f,
			}
			return child, nil
		}
	}
	return nil, fuse.ENOENT
}
```

## Files

Our `Lookup` above returned `File` types when the matched entry did
not end in a slash. Let's define type `File`, using the same `zipAttr`
helper as for directories:

``` go
type File struct {
	file *zip.File
}

var _ fs.Node = (*File)(nil)

func (f *File) Attr() fuse.Attr {
	return zipAttr(f.file)
}
```

Files are not very useful unless you can open them:

``` go
var _ = fs.NodeOpener(&File{})

func (f *File) Open(req *fuse.OpenRequest, resp *fuse.OpenResponse, intr fs.Intr) (fs.Handle, fuse.Error) {
	r, err := f.file.Open()
	if err != nil {
		return nil, err
	}
	// individual entries inside a zip file are not seekable
	resp.Flags |= fuse.OpenNonSeekable
	return &FileHandle{r: r}, nil
}
```

## Handles


``` go
type FileHandle struct {
	r io.ReadCloser
}

var _ fs.Handle = (*FileHandle)(nil)
```

We hold an "open file" inside our handle. In this case, it's just a
helper type in `archive/zip`, but in another filesystem this might be
a `*os.File`, a network connection, or such. We should be careful to
close them:

``` go
var _ fs.HandleReleaser = (*FileHandle)(nil)

func (fh *FileHandle) Release(req *fuse.ReleaseRequest, intr fs.Intr) fuse.Error {
	return fh.r.Close()
}
```

And then let's handle actual `Read` operations:

``` go
var _ = fs.HandleReader(&FileHandle{})

func (fh *FileHandle) Read(req *fuse.ReadRequest, resp *fuse.ReadResponse, intr fs.Intr) fuse.Error {
	// We don't actually enforce Offset to match where previous read
	// ended. Maybe we should, but that would mean'd we need to track
	// it. The kernel *should* do it for us, based on the
	// fuse.OpenNonSeekable flag.
	buf := make([]byte, req.Size)
	n, err := fh.r.Read(buf)
	resp.Data = buf[:n]
	return err
}
```

## Readdir

At this point, our files are accessible by `cat` and such, but you
need to know their names. Let's add support for `ReadDir`:

``` go
var _ = fs.HandleReadDirer(&Dir{})

func (d *Dir) ReadDir(intr fs.Intr) ([]fuse.Dirent, fuse.Error) {
	prefix := ""
	if d.file != nil {
		prefix = d.file.Name
	}

	var res []fuse.Dirent
	for _, f := range d.archive.File {
		if !strings.HasPrefix(f.Name, prefix) {
			continue
		}
		name := f.Name[len(prefix):]
		if name == "" {
			// the dir itself, not a child
			continue
		}
		if strings.ContainsRune(name[:len(name)-1], '/') {
			// contains slash in the middle -> is in a deeper subdir
			continue
		}
		var de fuse.Dirent
		if name[len(name)-1] == '/' {
			// directory
			name = name[:len(name)-1]
			de.Type = fuse.DT_Dir
		}
		de.Name = name
		res = append(res, de)
	}
	return res, nil
}
```

# Testing zipfs

Prepare a zip file:

``` console
$ mkdir -p data/buried/deep
$ echo hello, world >data/greeting
$ echo gold >data/buried/deep/loot
$ ( cd data && zip -r -q ../archive.zip . )
```

Mount it:

``` console
$ mkdir mnt
$ zipfs archive.zip mnt &
```

Lookup directory entries:

``` console
$ ls -ld mnt/greeting
-rw-r--r-- 1 root root 13 Dec 11  2014 mnt/greeting
$ ls -ld mnt/buried
drwxr-xr-x 1 root root 0 Dec 11  2014 mnt/buried
```

Read file contents:

``` console
$ cat mnt/greeting
hello, world
$ cat mnt/buried/deep/loot
gold
```

Readdir (the "total 0" is not correct, but that doesn't matter):

``` console
$ ls -l mnt
total 0
drwxr-xr-x 1 root root  0 Dec 11  2014 buried
-rw-r--r-- 1 root root 13 Dec 11  2014 greeting
$ ls -l mnt/buried
total 0
drwxr-xr-x 1 root root 0 Dec 11  2014 deep
```

Unmount (for OS X, use `umount mnt`):

``` console
$ fusermount -u mnt
```

That's it! For a longer and more featureful examples to read, see
https://github.com/bazillion/bolt-mount
([screencast of a code walkthrough](http://eagain.net/talks/bolt-mount/))
and all of the
[projects importing fuse](http://godoc.org/bazil.org/fuse?importers).

# Resources

- [Bazil](http://bazil.org/) is a distributed file system designed for
  single-person disconnected operation. It lets you share your files
  across all your computers, with or without cloud services.

- [FUSE](https://git.kernel.org/cgit/linux/kernel/git/torvalds/linux.git/tree/Documentation/filesystems/fuse.txt)
  is a Linux kernel filesystem that makes calls to userspace to serve
  filesystem content.

- Confusingly also known as [FUSE](http://fuse.sourceforge.net/) is
  the C library for implementing userspace FUSE filesystems.

- [bazil.org/fuse](http://bazil.org/fuse) is a Go library for writing
  filesystems. See also GoDoc for
  [`fuse`](http://godoc.org/bazil.org/fuse) and
  [`fuse/fs`](http://godoc.org/bazil.org/fuse/fs)

- [OSXFUSE](https://osxfuse.github.io/) is a FUSE kernel
  implementation for OS X.

- [`bolt-mount`](https://github.com/bazillion/bolt-mount) is a more
  comprehensive example filesystem, including write operations. See
  also a
  [screencast of a code walkthrough](http://eagain.net/talks/bolt-mount/).

- [*Writing a file system in Go*](http://bazil.org/talks/2013-06-10-la-gophers/)
  is an earlier talk that explains FUSE a bit more.

- FUSE questions are welcome on the
  [bazil-dev Google Group](https://groups.google.com/forum/#!forum/bazil-dev)
  or on IRC channel
  [#go-nuts on irc.freenode.net](irc:irc.freenode.net/go-nuts).
