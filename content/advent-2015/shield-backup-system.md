+++
author = ["Quintessence Anx"]
date = "2015-12-12T00:05:27-05:00"
linktitle = "SHIELD Backup System"
series = ["Advent 2015"]
title = "shield backup system"

+++

# Go in SHIELD

## Quick background: What is SHIELD?

[SHIELD](https://github.com/starkandwayne/shield) is a backup solution for Cloud Foundry and BOSH deployed services such as Redis, PostgreSQL, and Docker. (For the interested, [here](http://gnuconsulting.com/blog/2014/09/07/intro-to-cloud-foundry-and-bosh/) is a quick summary of the basics of BOSH and Cloud Foundry.) The original design was inspired by a client's need to have both broad and granular backups of their private Cloud Foundry and its ecosystem. Specifically, in addition to being able to recover from a meteor strike they also wanted to be able to create more granular backups so they could restore specific containers, credentials, databases, and so on. Since there was not an existing backup solution of this type available for Cloud Foundry/etc., we designed a new solution named SHIELD.

## Functions in Structs

Something we employed both in the CLI and in the server-side code was the ability to validate the received data to help keep out both non-sense and malicious values. A streamlined example of the validation functions from our CLI form looks like this:

```go
package main

import "fmt"

type FieldValidator func(name string, value int) error

type Form struct {
	Fields []*Field
}

type Field struct {
	Name      string
	Value     int
	Validator FieldValidator
}

func NewForm() *Form {
	f := Form{}
	return &f
}

func (f *Form) NewField(name string, value int, fcn FieldValidator) error {
	f.Fields = append(f.Fields, &Field{Name: name, Value: value, Validator: fcn})
	return nil
}

func InputIsNotBigEnough(name string, value int) error {
	if value < 3600 {
		return fmt.Errorf("input value must be greater than 3600")
	}
	return nil
}

func InputIsNotSmallEnough(name string, value int) error {
	if value > 3600 {
		return fmt.Errorf("input value must be less than 3600")
	}
	return nil
}

func (f *Form) Show() error {
	for _, field := range f.Fields {
		err := field.Validator(field.Name, field.Value)
		if err == nil {
			err = fmt.Errorf("value is valid")
		}
		fmt.Printf("name: '%s', value: '%d', validate: '%v'\n", field.Name, field.Value, err)
	}
	return nil
}

func main() {
	in := NewForm()
	in.NewField("number", 3599, InputIsNotBigEnough)
	in.NewField("another number", 3601, InputIsNotSmallEnough)
	in.NewField("last number", 2500, func(n string, v int) error {
		if v != 2500 {
			return fmt.Errorf("value is not 2500")
		}
		return nil
	})
	in.Show()
}
```

<small>[Click here](http://play.golang.org/p/GhW7dL-pCv) to test on the Go Playground.</small>

The `FieldValidator` type allows for any function as long as it takes in a `string` and `int` and outputs and an `error`. This allows us to pass different named functions, or even anonymous functions, as an argument. Since the function is stored in the struct, all three functions can just be called using dot notation just like the other elements of the struct (`field.Validator`).

### Using Interfaces with JSON

Some of you may be familiar with the concept of "function overloading". For those who aren't, function overloading is when a language permits you to create multiple functions with the same name. A trivial example might be to create two `add` functions: one for adding ints and another for adding floats. This allows you to avoid cluttering your code with function names like `addInt` and `addFloat`.

BUT: [Go explicitly doesn't support function overloading.](https://github.com/golang/go/wiki/GoForCPPProgrammers) (fourth to last bullet point). They provide a quick explanation about why not on their FAQ page [here](https://golang.org/doc/faq#overloading). And that's ok. Why? Because we have interfaces, and with interfaces we can accomplish the same result.

Specifically, we used interfaces a lot for handling JSON messages like this:

```go
package main

import (
	"encoding/json"
	"fmt"
)

type JSONMessage struct {
	Message []string `json:"message"`
}

func (m JSONMessage) JSON() string {
	b, err := json.Marshal(m)
	if err != nil {
		return fmt.Sprintf(`{"error":"failed to unmarshal JSON: %s"}`, err)
	}
	return string(b)
}

type JSONHoliday struct {
	Holiday string `json:"holiday"`
	Date    int    `json:"date"`
}

func (h JSONHoliday) JSON() string {
	b, err := json.Marshal(h)
	if err != nil {
		return fmt.Sprintf(`{"error":"failed to unmarshal JSON: %s"}`, err)
	}
	return string(b)
}

type ToJSON interface {
	JSON() string
}

func JSONify(j ToJSON) string {
	return fmt.Sprintf("%s", j.JSON())
}

func main() {
	message := JSONMessage{[]string{"Merry Christmas", "Happy New Year"}}
	holiday := JSONHoliday{"Boxing Day", 20151226}
	fmt.Printf("%s\n", JSONify(message))
	fmt.Printf("%s\n", JSONify(holiday))
}
```

<small>[Click here](http://play.golang.org/p/zNbfiko6Ds) to test on the Go Playground.</small>

In the SHIELD project we were mostly concerned error handling in JSON, so we wanted to use a single function for writing the JSON error to the client calling the API endpoint. Tying this into what we did above with validation functions in structs, this provided a consistent way to pass JSON error payloads from different sources.

## A Race Condition in the Pipes

It all started with a snippet of code that functioned like this:

```go
package main
​
import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
)
​
func Drain(prefix string, rd io.Reader) {
	b := bufio.NewScanner(rd)
	for b.Scan() {
		fmt.Printf("%s> %s\n", prefix, b.Text())
	}
}
​
func main() {
	/* <go> | ls -alh / | sort | <go> */
	ls := exec.Command("ls", "-alh", "/")
	sort := exec.Command("sort")
​
	/* grab standard error from both commands */
	err1, _ := ls.StderrPipe()
	err2, _ := sort.StderrPipe()
​
	/* wire up ls's stdout to sort's stdin */
	sort.Stdin, _ = ls.StdoutPipe()
​
	/* grab the output from the `sort' command */
	out, _ := sort.StdoutPipe()
​
	/* spin up two goroutines to stream errors */
	go Drain("error", err1)
	go Drain("error", err2)
​
	/* spin up a goroutine to stream output from `sort' */
	go Drain("output", out)
​
	_ = sort.Start()
	_ = ls.Start()
​
	_ = ls.Wait()
	_ = sort.Wait()
}
```

<small>Due to limitations in Go Playground this code must be run locally to reproduce, please see [example code on Github](https://github.com/starkandwayne/shield-race).</small>

As you can see, we're trying to drain stdout and stderr into their respective pipes to sort the output of the ls command. When you do this, you run into a race condition. What is a race  condition? [Simply put](http://www.airs.com/blog/archives/482): "A race condition occurs when one goroutine modifies a variable and another reads it or modifies it without any synchronization."

But how and why does this happen in our code?

Skimming over the code you can see that it is going to drain stdout and stderr into their respective pipes to sort the output of the ls command. `sort.Start` is run first so that `sort` is ready to sort for the output of `ls`, similar to running `ls | sort`. At this point, `sort` is running in the background until it's stdin is complete. In the code, you can see `sort.Stdin` is defined as `ls.StdoutPipe`. Then Go hits the `Wait` command. [Taking a look at `Wait`](https://golang.org/src/os/exec/exec.go#L372) and then looking back at the code, we can see that the descriptors that `Wait` needs to close before exiting are the read and stdout pipes. This read is the same read that `Drain` is trying to read (`rd`) from. Since `Wait` is trying to close what `Drain` is trying to read, the program fails with a data race condition like so:

```
$ ./shield-race
==================
WARNING: DATA RACE
Write by main goroutine:
  os.(*file).close()
      /private/var/folders/q8/bf_4b1ts2zj0l7b0p1dv36lr0000gp/T/workdir/go/src/os/file_unix.go:129 +0x1be
  os.(*File).Close()
      /private/var/folders/q8/bf_4b1ts2zj0l7b0p1dv36lr0000gp/T/workdir/go/src/os/file_unix.go:118 +0x88
  os/exec.(*Cmd).closeDescriptors()
      /private/var/folders/q8/bf_4b1ts2zj0l7b0p1dv36lr0000gp/T/workdir/go/src/os/exec/exec.go:241 +0x9f
  os/exec.(*Cmd).Wait()
      /private/var/folders/q8/bf_4b1ts2zj0l7b0p1dv36lr0000gp/T/workdir/go/src/os/exec/exec.go:390 +0x4c4
  main.main()
      /Users/quinn/dev/go/src/github.com/starkandwayne/shield-race/main.go:41 +0x3b7

Previous read by goroutine 6:
  os.(*File).read()
      /private/var/folders/q8/bf_4b1ts2zj0l7b0p1dv36lr0000gp/T/workdir/go/src/os/file_unix.go:211 +0x84
  os.(*File).Read()
      /private/var/folders/q8/bf_4b1ts2zj0l7b0p1dv36lr0000gp/T/workdir/go/src/os/file.go:95 +0xbc
  bufio.(*Scanner).Scan()
      /private/var/folders/q8/bf_4b1ts2zj0l7b0p1dv36lr0000gp/T/workdir/go/src/bufio/scan.go:180 +0x78a
  main.Drain()
      /Users/quinn/dev/go/src/github.com/starkandwayne/shield-race/main.go:12 +0x1a8

Goroutine 6 (running) created at:
  main.main()
      /Users/quinn/dev/go/src/github.com/starkandwayne/shield-race/main.go:32 +0x27b
==================
output> ----------     1 root  admin     0B Sep 15 23:00 .file
output> -rw-r--r--@    1 root  wheel   313B Aug 22 22:35 installer.failurerequests
output> -rw-rw-r--     1 root  admin     0B Aug 22 17:35 .DS_Store
output> d--x--x--x     9 root  wheel   306B Dec  9 20:52 .DocumentRevisions-V100
output> d-wx-wx-wt     2 root  wheel    68B Sep  1 18:30 .Trashes
output> dr-xr-xr-x     2 root  wheel     1B Dec  9 20:52 home
output> dr-xr-xr-x     2 root  wheel     1B Dec  9 20:52 net
output> dr-xr-xr-x     3 root  wheel   4.1K Dec  9 20:52 dev
output> drwx------     5 root  wheel   170B Apr 11  2015 .Spotlight-V100
output> drwx------  1927 root  wheel    64K Dec 21 16:00 .fseventsd
output> drwxr-xr-x     2 root  admin    68B Jan  8  2015 .PKInstallSandboxManager
output> drwxr-xr-x     3 root  wheel   102B Dec 21 15:23 src
==================
WARNING: DATA RACE
output> drwxr-xr-x     6 root  admin   204B Oct 13 14:18 Users
Write by main goroutine:
  os.(*file).close()
      /private/var/folders/q8/bf_4b1ts2zj0l7b0p1dv36lr0000gp/T/workdir/go/src/os/file_unix.go:129 +0x1be
  os.(*File).Close()
      /private/var/folders/q8/bf_4b1ts2zj0l7b0p1dv36lr0000gp/T/workdir/go/src/os/file_unix.go:118 +0x88
  os/exec.(*Cmd).closeDescriptors()
      /private/var/folders/q8/bf_4b1ts2zj0l7b0p1dv36lr0000gp/T/workdir/go/src/os/exec/exec.go:241 +0x9f
output> drwxr-xr-x    33 root  wheel   1.2K Dec 21 15:23 .
  os/exec.(*Cmd).Wait()
      /private/var/folders/q8/bf_4b1ts2zj0l7b0p1dv36lr0000gp/T/workdir/go/src/os/exec/exec.go:390 +0x4c4
  main.main()
      /Users/quinn/dev/go/src/github.com/starkandwayne/shield-race/main.go:42 +0x3dc

Previous read by goroutine 7:
  os.(*File).read()
      /private/var/folders/q8/bf_4b1ts2zj0l7b0p1dv36lr0000gp/T/workdir/go/src/os/file_unix.go:211 +0x84
  os.(*File).Read()
      /private/var/folders/q8/bf_4b1ts2zj0l7b0p1dv36lr0000gp/T/workdir/go/src/os/file.go:95 +0xbc
  bufio.(*Scanner).Scan()
      /private/var/folders/q8/bf_4b1ts2zj0l7b0p1dv36lr0000gp/T/workdir/go/src/bufio/scan.go:180 +0x78a
output> drwxr-xr-x    33 root  wheel   1.2K Dec 21 15:23 ..
  main.Drain()
      /Users/quinn/dev/go/src/github.com/starkandwayne/shield-race/main.go:12 +0x1a8

Goroutine 7 (finished) created at:
  main.main()
      /Users/quinn/dev/go/src/github.com/starkandwayne/shield-race/main.go:33 +0x2ed
==================
output> drwxr-xr-x+    3 root  wheel   102B Nov 10 01:46 .MobileBackups
output> drwxr-xr-x+   67 root  wheel   2.2K Nov 24 19:23 Library
output> drwxr-xr-x@    2 root  wheel    68B Oct 13 14:18 .vol
output> drwxr-xr-x@    2 root  wheel    68B Oct 13 14:18 Network
output> drwxr-xr-x@    4 root  wheel   136B Dec  9 20:50 System
output> drwxr-xr-x@    6 root  wheel   204B Oct 13 14:18 private
output> drwxr-xr-x@   13 root  wheel   442B Nov 24 19:23 usr
output> drwxr-xr-x@   39 root  wheel   1.3K Dec  9 20:50 bin
output> drwxr-xr-x@   59 root  wheel   2.0K Dec  9 20:50 sbin
output> drwxrwxr-t@    2 root  admin    68B Oct 13 14:18 cores
output> drwxrwxr-x+   75 root  admin   2.5K Dec 18 21:56 Applications
output> drwxrwxr-x@    3 root  wheel   102B Sep  2 15:11 opt
output> drwxrwxrwt@    5 root  admin   170B Dec 21 17:03 Volumes
output> lrwxr-xr-x@    1 root  wheel    11B Oct 13 14:17 etc -> private/etc
output> lrwxr-xr-x@    1 root  wheel    11B Oct 13 14:17 tmp -> private/tmp
output> lrwxr-xr-x@    1 root  wheel    11B Oct 13 14:18 var -> private/var
output> total 45
Found 2 data race(s)
```

Note that two data races are found - one for each of the goroutines draining the stderr pipes.

Since this example is a lot simpler than the one we ran into in SHIELD it still works in these sense that when you build the binary without the [data race detector](http://blog.golang.org/race-detector) the code will compile and run as expected. This is because you are essentially just running `ls | sort` which, unless you have a truly massive set of subdirectories, should resolve itself relatively quickly. The same cannot be said of of the similar data race we created and found in SHIELD. Finding that data race was a bit arduous and felt a little like receiving a lump of coal in our stockings from our past selves.

We found the data race by //FIXME

To see the exact situation we ran into in SHIELD, take a look at [task.go in commit 6020baa](https://github.com/starkandwayne/shield/blob/6020baae38d37e4233e57d75a9a49202c4c4ce5e/supervisor/task.go). When we researched the data race, we discovered that this is a known problem with execing certain commands/pipes (see [here](https://github.com/golang/go/issues/9307), [here](https://github.com/golang/go/issues/9382), and [here](https://code.google.com/p/go/issues/detail?id=2266)). As a result, we ended up resolving the issue by implementing a [BASH script](https://github.com/starkandwayne/shield/blob/master/bin/shield-pipe) to handle the pipes. The first commit with this fix is [3548036](https://github.com/starkandwayne/shield/commit/35480364275f12d6fc122eed8089e2113fa5a162). Since then, the relevant go code has been refactored into `request.go`.

## Addendum: Want to test and contribute to SHIELD?

To setup a local testing environment for SHIELD, please use the `setup-env` script in our [testbed](https://github.com/starkandwayne/shield-testbed-deployments) repository after [setting up a local Cloud Foundry on BOSH Lite](https://github.com/cloudfoundry/bosh-lite/). Our testbed deploys Cloud Foundry and docker-postgres with SHIELD and sets up some initial dummy values in SHIELD itself.

The CLI is pretty straightforward - for example `shield create target` and `shield list stores`. There are also some aliased commands, e.g. `shield ls` for `shield list`. For a full list of commands, please take a look at the CLI documentation on the project [README](https://github.com/starkandwayne/shield#cli-usage-examples).

We welcome feedback and pull requests!

### And also...

I hope everyone is enjoying their winter holidays and Merry Christmas to those who celebrate! :)
