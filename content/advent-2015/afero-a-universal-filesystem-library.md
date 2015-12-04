+++
author = ["Steve Francia"]
date = "2015-11-30T10:17:26-05:00"
linktitle = "afero: A Universal Filesystem Library"
series = ["Advent 2015"]
title = "afero: A Universal Filesystem Library"

+++

Anyone who spends a few minutes talking with me about development knows I love Go. [My Github](https://github.com/spf13) and [Blog](http://spf13.com) are practically a love letter to the language. Today I want to share with you something I've been working on for the past year and I hope you will really get excited about it. 

Many applications require access to the file system to create, modify or delete files and folders. I've always felt a bit odd making calls directly to the os package. From my experience with other languages I've learned that having external dependencies are to be avoided whenever possible so that you can properly test your code independent of environmental factors.

What if there was a abstract virtual memory framework where you could plug in a variety of backends? Not only would that be fantastic for unit testing but it could open up a lot of very powerful possibilities. This thought kept running through my mind as I was coding more and more "os" calls in [Hugo](https://gohugo.io). I finally gave in and decided that since nobody had created an abstract file system framework in Go that I would do it. Two days later [Afero](https://github.com/spf13/afero) was born. 

<img alt="Afero"
     src="/postimages/advent-2015/Afero Logo.png"
     style="float:right; margin-bottom:2em;"/>

# Designing the Afero FS framework

Go interfaces proved absolutely perfect for such a framework. With a standard set of defined interfaces anyone could extend Afero with another (interoperatable) backend. Through it's interface, Afero provides a single API for accessing a variety of different files systems, even things typically not thought of as a file system. Through these two interfaces, Afero presents a uniform view of the files for anything from memory caches, to zip files, to remote files systems. 

The standard library already provided the excellent [OS package](https://golang.org/pkg/os/) which provides provides a platform-independent interface to operating system and specifically for our purposes, to the OS file operations. The OS package does an amazing job of separating and partitioning out responsibilities to a core set of functions and types that represent all the file and directory operations a file system would need. 

I decided that I would use the OS as a guide as I defined the interfaces. It would not only give me a great starting point, it would also ensure that both the OS package would be a compatible backend and it would be very easy to migrate from the OS package to Afero. 

## The Afero File System Interface

```go
type Fs interface {
	Create(name string) (File, error)
	Mkdir(name string, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
	Open(name string) (File, error)
	OpenFile(name string, flag int, perm os.FileMode) (File, error)
	Remove(name string) error
	RemoveAll(path string) error
	Rename(oldname, newname string) error
	Stat(name string) (os.FileInfo, error)
	Name() string
	Chmod(name string, mode os.FileMode) error
	Chtimes(name string, atime time.Time, mtime time.Time) error
}

```

## The Afero File Interface

```go
type File interface {
	io.Closer
	io.Reader
	io.ReaderAt
	io.Seeker
	io.Writer
	io.WriterAt

	Name() string
	Readdir(count int) ([]os.FileInfo, error)
	Readdirnames(n int) ([]string, error)
	Stat() (os.FileInfo, error)
	Sync() error
	Truncate(size int64) error
	WriteString(s string) (ret int, err error)
}
```

# Implementing the Afero core library

I felt good about the interfaces but recognized that a without at least a couple libraries satisfying the interface I would never know if it held up. The first implementation was really just to prove that this idea would work. 

## Implementing the OS backend

True to form, the OS backend was trivial to implement as it was just a wrapper. 

```go
type OsFs struct{}

func (OsFs) Name() string { return "OsFs" }

func (OsFs) Create(name string) (File, error) { return os.Create(name)}

func (OsFs) Mkdir(name string, perm os.FileMode) error { return os.Mkdir(name, perm)}

func (OsFs) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm)}

func (OsFs) Open(name string) (File, error) { return os.Open(name)}

func (OsFs) OpenFile(name string, flag int, perm os.FileMode) (File, error) { return os.OpenFile(name, flag, perm)}

func (OsFs) Remove(name string) error { return os.Remove(name)}

func (OsFs) RemoveAll(path string) error { return os.RemoveAll(path)}

func (OsFs) Rename(oldname, newname string) error { return os.Rename(oldname, newname) }

func (OsFs) Stat(name string) (os.FileInfo, error) { return os.Stat(name)}

func (OsFs) Chmod(name string, mode os.FileMode) error { return os.Chmod(name, mode)}

func (OsFs) Chtimes(name string, atime time.Time, mtime time.Time) error { return os.Chtimes(name, atime, mtime)}

```

## Implementing a memory based backend 

This backend was the entire reason I wrote Afero in the first place. I has a suspicion that Hugo could be even faster if it rendered and served files from memory instead of from disk, but in a cross platform way without using something that requried complicated setup like a ram disk. I also wanted something that I could use for testing where I could easily setup a bunch of files without worrying about where the environment was testing. I didn't want to worry about the state of the files when I ran the test or cleaning up the files when I was done. 

I ended up creating two go files. One to satisfy the `File interface` and another to satisfy the filesystem interface. 

I knew that if I could pass the OS test suite using my memory based backend then it would be a pretty solid implementation. 

### Memory File
There was a lot to think through here. The OS provides things like locking for you. Here we had to implement it all ourselves. Additionally some things don't really have the same meaning when it's just some bytes in ram. I experimented a bit and found that the following struct would be sufficient to store all the needed information and data of a file. 

```go
type InMemoryFile struct {
	// atomic requires 64-bit alignment for struct field access
	at           int64
	readDirCount int64
	sync.Mutex
	name    string
	data    []byte
	memDir  MemDir
	dir     bool
	closed  bool
	mode    os.FileMode
	modtime time.Time
}
```

Using a byte slice proved a fantastic way to store the data for a file. It made it easy to satify the requirements of Seek and ReadAt while benefitting from Go's already highly optimized slice sizing routines. 

```go
func (f *InMemoryFile) Seek(offset int64, whence int) (int64, error) {
	if f.closed == true {
		return 0, ErrFileClosed
	}
	switch whence {
	case 0:
		atomic.StoreInt64(&f.at, offset)
	case 1:
		atomic.AddInt64(&f.at, int64(offset))
	case 2:
		atomic.StoreInt64(&f.at, int64(len(f.data))+offset)
	}
	return f.at, nil
}

```

I won't go through the entire implementation details, but for the curious reader you can find it on [github](https://github.com/spf13/afero/blob/master/memfile.go).

### Memory File System

I felt that the easiest way to implement a quick file system would be using a map. This proved to be a bit tricky as a map is a flat structure and a file system is a tree.. or at least that's how we typically perceive it. In reality all a file system really is just a list of files with paths. The paths are just strings and while we typically browse them as a tree, you could just as easily just view it as a list of files which have an attribute called path. This realization made it fairly easy to implement the filesystem as a map. While a map makes a lot of sense for small structures and it served well as a quick implementation, I believe that a radix or similar tree will be a much more performant backend particularly when searching for files. 

### Directories

I wasn't really sure how to approach a directory. On disk a directory serves a concrete purpose, but not in memory (at least not with the approach I was taking). After some thought I figured it should just be a file as it shares virtually all the same properties as a file. I added the `dir` flag to connote that this was a directory. Due to this approach it was no longer necessary to have a directory to create a file who had a corresponding path (remember, in this implementation a path is just a property). To ensure it would be more compatible with traditional file system operations we tried to emulate traditional behavior as much as possible. Each directory would contain a slice of pointers to all of it's children. 

```go
type MemDir interface {
	Len() int
	Names() []string
	Files() []File
	Add(File)
	Remove(File)
}
```

It's important to recognize that the interface is not a limit but a guide. One of the wonderful things about Go interfaces is that they aren't prescriptive meaning you are welcome to exceed the interface as long as you still satisfy it.  To accomplish our purposes we added additional functions to facilitate easy and fast filesystem operations. We would need to add some extra handling to ensure that this was kept always up to date. Even though this function isn't required by the interface, it was very beneficial to our keeping our implementation clean. This function serves to ensure that a given file or directory is registered to it's parent (directory). 

```go
func (m *MemMapFs) registerWithParent(f File) {
	if f == nil {
		return
	}
	parent := m.findParent(f)
	if parent == nil {
		pdir := filepath.Dir(path.Clean(f.Name()))
		err := m.lockfreeMkdir(pdir, 0777)
		if err != nil {
			return
		}
		parent, err = m.lockfreeOpen(pdir)
		if err != nil {
			return
		}
	}
	pmem := parent.(*InMemoryFile)

	if pmem.memDir == nil {
		pmem.dir = true
		pmem.memDir = &MemDirMap{}
	}

	pmem.memDir.Add(f)
}

``` 


# Extending Afero

My initial idea was accomplished in that Hugo is now using Afero to handle all of it's filesystem operations. The latest version of Hugo uses the memory storage backend for it's `hugo server` operation with significant performance benefits. This is all still just the beginning as we as a community could do a lot more to extend and take Afero further. 

It would be amazing if Afero had support for tar & zip archives. The standard library already provides a robust and complete zip and tar package, someone just needs to proivide an Afero compatible interface to them. It would be great if Afero had support for SCP/SSH, S3 and other cloud storage. If these implementations were created then someone could trivially add support for all these different systems natively with virtually no code changes. 

With these filesystems in place we could take things even further. There would be tremendous value in filesystems that wrapped or combined other filesystems. For instance a caching filesystem that front a slower S3 filesystem with a local on disk one. Another obvious benefit would be to use a memory filesystem in front of a disk or network one. There are a lot of potential features when combining filesystems in this way, versioning, journaling, on the fly compression.

I hope you find value in this project and look forward to seeing the amazing and creative backends that the community produces. 

