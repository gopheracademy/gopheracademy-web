+++
author = ["Quintessence Anx"]
date = "2015-12-12T00:05:27-05:00"
linktitle = "SHIELD Backup System"
series = ["Advent 2015"]
title = "shield backup system"

+++

# Go in SHIELD

## Quick background: What is SHIELD?

Like many software projects, SHIELD was conceived out of need. In particular, we had a
client who deployed their own Cloud Foundry with BOSH and they needed regular backups of
Cloud Foundry itself and its services, such as Redis, PostgreSQL, Docker ... you get the idea.
For the interested, [here](http://gnuconsulting.com/blog/2014/09/07/intro-to-cloud-foundry-and-bosh/)
is a quick summary of the basics of BOSH and Cloud Foundry.

//FIXME - trim the next paragraph down more.

Since our client was using vSphere we initially considered/tried using vSphere's
snapshot abilities, but ultimately switched to a custom solution four a couple reasons.
One was that BOSH doesn't work well with traditional snapshots because of the way it
resurrects failed VMs. Since the IDs for VMs, persistent disk, etc. will change restoring
a snapshot with now-outdated IDs led to issues like the persistent disk going missing
because the restored snapshot didn't have the correct ID. The other reason was a change
in the client requirements. In addition to wanting to be able to do broader restores in the
event of meteor strikes, they also wanted to be able to do more granular restores for
day to day activities. For example, if operations wanted to restore specific
containers or credentials or if developers wanted to restore specific databases after
tinkering. Since this new requirement lent itself to a custom solution, we rolled
both requirements into SHIELD's design.

## Functions in Structs
Something we employed both in the CLI and in the server-side code was the ability
to validate the received data to help keep out both non-sense and malicious values. A
streamlined example of the validation functions from our CLI form looks like this:

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

The `FieldValidator` type allows for any function as long as it takes in a `string` and `int` and outputs
an `error`. This allows us to pass different named functions, or even anonymous functions, as an argument.
Since the function is stored in the struct, all three functions can just be called using dot notation just
like the other elements of the struct: `field.Validator` (ln 43).

### Interfaces

Take a quick look at the following code:

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

type JSONDate struct {
	Holiday string `json:"holiday"`
	Date    int    `json:"date"`
}

func (d JSONDate) JSON() string {
	b, err := json.Marshal(d)
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
	hdate := JSONDate{"First Day of Hanukkah", 20151206}
	fmt.Printf("%s\n", JSONify(message))
	fmt.Printf("%s\n", JSONify(hdate))
}
```

<small>[Click here](http://play.golang.org/p/SaZV522EBc) to test on the Go Playground.</small>

In SHIELD we used an interface for JSON payloads similar to the above, but for error handling.
Specifically we used a single function for writing the JSON error the client calling the API
endpoint. Tying this into what we did above with validation functions in structs, this
provided a consistent way to pass JSON error payloads since Go does not support function
overloading.

## A Race Condition in the Pipes

It all started with a snippet of code that looked similar to this:

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

<small>Due to limitations in Go Playground this code must be run locally to reproduce:<br/>
`mkdir -p $GOTPAH/src/github.com/starkandwayne && cd !$ && git clone {{URL}} && cd race-condition && go build`</small>

As you can see, we're trying to drain stdout and stderr into their respective pipes to sort the output
of the ls command. When you do this, you run into a race condition. What is a race condition? Well
simply put it means that two things are accessing the same thing at the same time. (Worst. Sentence. Ever.) //FIXME
But how and why does this happen in our code?

Skimming over the code you can see that it is going to drain stdout and stderr
into their respective pipes to sort the output of the ls command. `sort.Start` is
run first so that `sort` is ready to sort for the output of `ls`, similar to running `ls | sort`.
At this point, `sort` is running in the background until it's stdin is complete. In the code, you
can see `sort.Stdin` is defined as `ls.StdoutPipe`. Then Go hits the `Wait` commands.
[Taking a look at `Wait`](https://golang.org/src/os/exec/exec.go#L372), the descriptors that
it is waiting to close before exiting are the read and stdout pipes. This read is the same
read that `Drain` is trying to read from, defined as `rd`. Since `Wait` is trying to
close what `Drain` is trying to read, the program fails with a data race condition.

To see the exact situation we ran into in SHIELD, take a look at [task.go in commit 6020baa](https://github.com/starkandwayne/shield/blob/6020baae38d37e4233e57d75a9a49202c4c4ce5e/supervisor/task.go).
When we researched the data race, we discovered that this is a known problem with execing
certain commands/pipes (see [here](https://github.com/golang/go/issues/9307), [here](https://github.com/golang/go/issues/9382), and [here](https://code.google.com/p/go/issues/detail?id=2266)). As a result, we ended
up resolving the issue by implementing a [BASH script](https://github.com/starkandwayne/shield/blob/master/bin/shield-pipe)
to handle the pipes. The first commit with this fix is [3548036](https://github.com/starkandwayne/shield/commit/35480364275f12d6fc122eed8089e2113fa5a162).
Since then, the relevant go code has been refactored into `request.go`.

## Addendum: Want to test and contribute to SHIELD?

To setup a local testing environment for SHIELD, please use the `setup-env` script in our [testbed](https://github.com/starkandwayne/shield-testbed-deployments) repository after
[setting up a local Cloud Foundry on BOSH Lite](https://github.com/cloudfoundry/bosh-lite/).
Our testbed deploys Cloud Foundry and docker-postgres with SHIELD and sets up some initial dummy values in SHIELD itself.

The CLI is pretty straightforward - for example `shield create target` and `shield list stores`. There are also some aliased commands, e.g. `shield ls` for `shield list`. For a full list of commands, please take a look at the CLI documentation on the project [README](https://github.com/starkandwayne/shield#cli-usage-examples).

We welcome feedback and pull requests!

### And also...

I hope everyone is enjoying their winter holidays and Merry Christmas to those who celebrate! :)
