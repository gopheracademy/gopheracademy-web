+++
author = ["Quintessence Anx"]
date = "2015-12-12T00:05:27-05:00"
linktitle = "Lessons in Go Learned while Implementing SHIELD"
series = ["Advent 2015"]
title = "shield backup system"

+++

# Lessons in Go Learned while Implementing SHIELD

## Quick background: What is SHIELD?

[SHIELD][shield] is a backup solution for Cloud Foundry and BOSH deployed services such as Redis, PostgreSQL, and Docker. (For the interested, [here][CFBlogPost] is a quick summary of the basics of BOSH and Cloud Foundry.) The original design was inspired by a client's need to have both broad and granular backups of their private Cloud Foundry and its ecosystem. Specifically, in addition to being able to recover from a meteor strike they also wanted to be able to create more granular backups so they could restore specific containers, credentials, databases, and so on. Since there was not an existing backup solution of this type available for Cloud Foundry/etc., we designed a new solution named SHIELD.

## Functions as Fields in Structs

Something we employed both in the CLI and in the server-side code was the ability to validate the received data to help keep out both non-sense and malicious values. Since different fields were expected to have different inputs, we ended up with several validation functions checking everything from whether input is appropriately structured and valid JSON to checking the values and types. There were so many different functions floating around and we needed a way to consistently work them into the input validation process.

The example code below is a scaled down version of the process. There are the `Form` and `Field` structs, but there is also a function type declared called `FieldValidator`. The `FieldValidator` type allows any function to be stored in the `Validator` field as long as it (1) takes in a `string` and an `int` and (2) outputs an `error`. You can see from the example that this holds true not just for the named functions `InputIsNotBigEnough` and `InputIsNotSmallEnough` but anonymous functions as well. The value in `Value` is evaluated when the form is shown and instead of some complicated legwork to match the appropriate validation functions to their field outside the struct, I can just call the validation function stored in the struct as `field.Validator`.

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

func DateIsNotLateEnough(name string, value int) error {
	if value < 20151206 {
		return fmt.Errorf("date must be after 20151206")
	}
	return nil
}

func DateIsNotEarlyEnough(name string, value int) error {
	if value > 20151214 {
		return fmt.Errorf("date must be earlier than 20151214")
	}
	return nil
}

func (f *Form) Show() error {
	for _, field := range f.Fields {
		err := field.Validator(field.Name, field.Value)
		if err == nil {
			err = fmt.Errorf("date is correct")
		}
		fmt.Printf("name: '%s', value: '%d', validate: '%v'\n", field.Name, field.Value, err)
	}
	return nil
}

func main() {
	in := NewForm()
	in.NewField("Hanukkah", 20151205, DateIsNotLateEnough)
	in.NewField("Another day of Hanukkah", 20151215, DateIsNotEarlyEnough)
	in.NewField("Diwali", 20151111, func(n string, v int) error {
		if v != 20151111 {
			return fmt.Errorf("diwali must be 20151111")
		}
		return nil
	})
	in.Show()
}
```

<small>[Click here][form-play] to test on the Go Playground.</small>

### Using Interfaces with JSON

Some of you may be familiar with the concept of "function overloading". For those who aren't, function overloading is when a language permits you to create multiple functions with the same name. A trivial example might be to create two `add` functions: one for adding ints and another for adding floats. This allows you to avoid cluttering your code with function names like `addInt` and `addFloat`.

BUT: [Go explicitly doesn't support function overloading][GoForCPP] (fourth to last bullet point). They provide a quick explanation about why on their FAQ page [here][GoFAQ]. And that's ok - we don't really need function overloading to accomplish our goals. Why? Because we have interfaces!

In the SHIELD project we were mostly concerned with this for error handling in JSON: we wanted to use a single function for writing JSON errors to the client calling the API endpoint. Tying this into what we did above with validation functions in structs, interfaces provided a consistent way to pass JSON error payloads from different sources. Simplified example:

```go
package main

import (
	"encoding/json"
	"fmt"
)

type JSONError interface {
	JSON() string
}

type MissingParametersError struct {
	Missing []string `json:"missing"`
}
func (e *MissingParametersError) Check(name string, value string) {
	if value == "" {
		e.Missing = append(e.Missing, name)
	}
}
func (e MissingParametersError) Error() string {
	return fmt.Sprintf("missing: %v", e.Missing)
}
func (e MissingParametersError) JSON() string {
	b, err := json.Marshal(e)
	if err != nil {
		return fmt.Sprintf(`{"error":"failed to unmarshal JSON: %s"}`, err)
	}
	return string(b)
}




type InvalidParametersError struct {
	Errors map[string]string `json:"invalid"`
}
func (e *InvalidParametersError) Validate(name string, value string, fn func(string, string) error) {
	err := fn(name, value)
	if err != nil {
		e.Errors[name] = err.Error()
	}
}
func (e InvalidParametersError) Error() string {
	return fmt.Sprintf("invalid parameters: %v", e.Errors)
}
func (e InvalidParametersError) JSON() string {
	b, err := json.Marshal(e)
	if err != nil {
		return fmt.Sprintf(`{"error":"failed to unmarshal JSON: %s"}`, err)
	}
	return string(b)
}

func respond(e JSONError) {
	fmt.Printf("%s\n", e.JSON())
}

func main() {
	e1 := &MissingParametersError{}
	e1.Check("name",    "A Thing")
	e1.Check("summary", "")
	e1.Check("required", "")
	respond(e1)

	e2 := &InvalidParametersError{Errors: map[string]string{}}
	e2.Validate("number", "a string", func (name string, value string) error {
		return fmt.Errorf("%s: %v is not a number", name, value)
	})
	e2.Validate("value", "42", func(name string, value string) error {
		return fmt.Errorf("%s: %v is too small", name, value)
	})
	respond(e2)
}
```

<small>[Click here][json-play] to test on the Go Playground.</small>

## A Race Condition in the Pipes

First, what is a race condition?

> "A race condition occurs when one goroutine modifies a variable and another reads it or modifies it without any synchronization." *--[Source][race-def]*

The code that caused our race condition was a result of the way we had originally tried to implement the backup and restore processes. SHIELD has both target and store plugins so a user can select what type of data is being backed up or restored (e.g. PostgreSQL) and what type of storage the data is being backed up to or restored from (e.g. S3). To run the backups and restores, we created stdin, stdout, stderr pipes for both the target and store. In the case of a backup, the store read what was being sent by the target (i.e. the target's standard output was piped into the store's standard input). Likewise, during a restore the target read what was being sent by the store (i.e. the store's standard output was piped into the target's standard input).

While this conceptually makes sense and should work, we ran into issues during testing when we started to see non-reproducible, inconsistent, and seemingly random failures when we deployed the dev release to our testing environment. Things like leaking pipes where not all the data from stdout was making it to the corresponding stdin and corrupted archives. We upped the ante on our testing adding in various payload sizes, random sleeps, etc. but no matter what we did to our tests they still all passed. After our attempts to expose the issues we were experiencing with tests failed, we began to suspect we had created non-deterministic code. But how? To help show what happened, here's a trimmed down example:

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

<small>Due to limitations in Go Playground this code must be run locally to reproduce, please see [example code on Github][race-code].</small>

In this example, `ls` and `sort` are taking over the roles of `target` and `store` from SHIELD. Skimming over the code you can see that it is going to drain stdout and stderr into their respective pipes to `sort` the output of the `ls` command. `sort.Start` is run first so that `sort` is ready to sort the output of `ls`, similar to the way that store would wait to read the output of target (or vice versa). At this point, `sort` is running in the background and will continue to do so until its stdin is complete. Since `sort.Stdin` is defined as `ls.StdoutPipe`, that means waiting for the `ls.StdoutPipe` to complete. Then Go hits the `Wait` command. [Taking a look at `Wait`][go-src-code] and then looking back at the code, we can see that the descriptors that `Wait` needs to close before exiting are the read and stdout pipes. This read is the same read that `Drain` is trying to read (`rd`) from. Since `Wait` is trying to close what `Drain` is trying to read, the program fails with a data race condition like so:

```
$ go build -race
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

Since this example is a lot simpler than the one we ran into in SHIELD it still works in these sense that when you build the binary without the [data race detector][race-dedect], `go build -race`, the code will compile and run as expected. This is because you are essentially just running `ls | sort` which, unless you have a truly massive set of subdirectories, should resolve itself relatively quickly. The same cannot be said of of the similar data race we created and found in SHIELD.

To see exactly how the race condition appeared in SHIELD at that time, take a look at [task.go in commit 6020baa][task-6020baa]. When we researched our data race, we discovered that it is a known problem with exec'ing certain commands/pipes (see [here][race-1], [here][race-2], and [here][race-3]). As a result, we ended up resolving the issue by implementing a [BASH script][shield-bash] to handle the pipes. The first commit with this fix is [3548036][commit-3548036]. Since then, the relevant go code has been refactored into `request.go`.

## Open Invitation: Want to test and contribute to SHIELD?

To setup a local testing environment for SHIELD, please use the `setup-env` script in our [testbed][shield-test] repository after [setting up a local Cloud Foundry on BOSH Lite][bosh-lite]. Our testbed deploys Cloud Foundry and docker-postgres with SHIELD and sets up some initial dummy values in SHIELD itself.

The CLI is pretty straightforward - for example `shield create target` and `shield list stores`. There are also some aliased commands, e.g. `shield ls` for `shield list`. For a full list of commands, please take a look at the CLI documentation on the project [README][shield-cli].

We welcome feedback and pull requests!

### And also...

I hope everyone is enjoying their winter holidays and Merry Christmas to those who celebrate! :)

[shield]: https://github.com/starkandwayne/shield
[CFBlogPost]: http://gnuconsulting.com/blog/2014/09/07/intro-to-cloud-foundry-and-bosh/
[form-play]: http://play.golang.org/p/gsgKJ6HL5X
[json-play]: http://play.golang.org/p/-cj2QO-_RY
[race-code]: https://github.com/starkandwayne/shield-race
[GoForCPP]: https://github.com/golang/go/wiki/GoForCPPProgrammers
[GoFAQ]: https://golang.org/doc/faq#overloading
[race-def]: http://www.airs.com/blog/archives/482
[go-src-code]: https://golang.org/src/os/exec/exec.go#L372
[race-dedect]: http://blog.golang.org/race-detector
[task-6020baa]: https://github.com/starkandwayne/shield/blob/6020baae38d37e4233e57d75a9a49202c4c4ce5e/supervisor/task.go
[commit-3548036]: https://github.com/starkandwayne/shield/commit/35480364275f12d6fc122eed8089e2113fa5a162
[race-1]: https://github.com/golang/go/issues/9307
[race-2]: https://github.com/golang/go/issues/9382
[race-3]: https://code.google.com/p/go/issues/detail?id=2266
[shield-bash]: https://github.com/starkandwayne/shield/blob/master/bin/shield-pipe
[shield-test]: https://github.com/starkandwayne/shield-testbed-deployments
[shield-cli]: https://github.com/starkandwayne/shield#cli-usage-examples
[bosh-lite]: https://github.com/cloudfoundry/bosh-lite/
