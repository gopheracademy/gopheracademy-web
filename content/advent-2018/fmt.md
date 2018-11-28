vim: tw=80
+++
title = "Dive Into the `fmt` Package"
date = "2018-11-24T16:07:17+02:00"
series = ["Advent 2018"]
author = ["Miki Tebeka"]
draft = true
+++

We usually use the `fmt` package without giving it much thought. A `fmt.Printf`
here, a `fmt.Sprintf` there and on we go. However, if you'll take a closer look,
you'll be able to get much out of it.

Since Go is used a lot to write servers or services, our main mean of debugging
is the logging system. The `log` package provides `log.Printf` which has the
same semantics as `fmt.Printf`. Good and informative log messages are worth
their weight in gold and adding some formatting support to your data structure
will add valuable information to your log messages.

## Formatting Output

Go formatted output supports several verbs, the most common ones are `%s` for
strings, '%d` for integers and `%f` for floats. Let's see some more advanced verbs.

### %v & %T

`%v` will print any Go value and `%T` will print the type of the variable. Handy
when trying. I use these verbse when debugging.

```go
var e interface{} = 2.7182
fmt.Printf("e = %v (%T)\n", e, e) // e = 2.7182 (float64)
```

### Width

You can specify the width of a printed number, for example

```go
fmt.Printf("%10d\n", 353)  // will print "       353"
```

You can also specify the width as a parameter to `Printf` by specifying the
width as `*`. For example:
```go
fmt.Printf("%*d\n", 10, 353)  // will print "       353"
```

This is useful when print out number and would like to align them to the right.

```go
func alignSize(nums []int) int {
	size := 0
	for _, n := range nums {
		if s := int(math.Log10(float64(n))) + 1; s > size {
			size = s
		}
	}

	return size
}

func main() {
	nums := []int{12, 237, 3878, 3}
	size := alignSize(nums)
	for i, n := range nums {
		fmt.Printf("%02d %*d\n", i, size, n)
	}
}
```

will print
```
00   12
01  237
02 3878
03    3
```

Making it easier for us to compare the numbers.

### Refernce by Position

If you're referencing a variable several times inside a format string. You can
reference by position using `%[n]` where n is the index of the parameter (1
based).

```go
fmt.Printf("The price of %[1]s was $%[2]d. $%[2]d! imagine that.\n", "carrot", 23)
```

will print
```
The price of carrot was $23. $23! imagine that.
```

### %v
`%v` will print a Go value, it can be modified with `+` to print field names in
a struct and with `#` to print field names and type.

```go
// Point is a 2D point
type Point struct {
	X int
	Y int
}

func main() {
	p := &Point{1, 2}
	fmt.Printf("%v %+v %#v \n", p, p, p)
}
```

will print
```
&{1 2} &{X:1 Y:2} &main.Point{X:1, Y:2} 
```

I tend to use the `%+v` verb most.


## fmt.Stringer & fmt.Formatter

Sometimes you'd like a finer control on how your objects are printed. For
example you'd like one string representation for an error when it is shown to the
user and another, more detailed, when it is written to log.

One good exeample is the excellent
[`github.com/pkg/errors`](https://github.com/pkg/errors) is makeing use of
`fmt.Formatter`. Say you'd like to load our configuration file with and you have
an error. You can print a short error to the user (or return it in API ...) and
print a more detailed error to the log.

```go
cfg, err := loadConfig("/no/such/config.toml")
if err != nil {
	fmt.Printf("error: %s\n", err)
	log.Printf("can't load config\n%+v", err)
}
```

this will emit to the user
```
error: can't open config file: open /no/such/file.toml: no such file or directory
```

and to the log file

```
2018/11/28 10:43:00 can't load config
open /no/such/file.toml: no such file or directory
can't open config file
main.loadConfig
	/home/miki/Projects/gopheracademy-web/content/advent-2018/fmt.go:101
main.main
	/home/miki/Projects/gopheracademy-web/content/advent-2018/fmt.go:135
runtime.main
	/usr/lib/go/src/runtime/proc.go:201
runtime.goexit
	/usr/lib/go/src/runtime/asm_amd64.s:1333
```


To control how your objects are printed, you need to implement
[`fmt.Formatter`](https://golang.org/pkg/fmt/#Formatter) interface and
optionally [`fmt.Stringer`](https://golang.org/pkg/fmt/#Stringer).

Say you have an `AuthInfo` struct for a user

```go
// AuthInfo is authentication information
type AuthInfo struct {
	Login  string // Login user
	ACL    uint   // ACL bitmask
	APIKey string // API key
}
```

You'd like to limit the chances that the `APIKey` will be printed out (say when
you log). You decided to print a mask instead of the key

```
const (
	keyMask = "*****"
)
```

First the easy case `fmt.Stringer`.

```go
// String implements Stringer interface
func (ai *AuthInfo) String() string {
	return fmt.Sprintf("Login:%s, ACL:%08b, APIKey: %s", ai.Login, ai.ACL, keyMask)
}
```

And now `fmt.Formatter` which gets a
[`fmt.State`](https://golang.org/pkg/fmt/#State) and rune for the verb.
`fmt.State` implements [`io.Writer`](https://golang.org/pkg/io/#Writer),
enabling you to write directly to it.

To know all the fields available in a struct, you can use the [`reflect`](https://golang.org/pkg/reflect/).
package. This will make sure your code works even when `AuthInfo` changes.

```go
var authInfoFields []string

func init() {
	typ := reflect.TypeOf(AuthInfo{})
	authInfoFields = make([]string, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		authInfoFields[i] = typ.Field(i).Name
	}
	sort.Strings(authInfoFields) // People are better with sorted data
}
```

And now you're ready to implement `fmt.Formatter`
```go
// Format implements fmt.Formatter
func (ai *AuthInfo) Format(state fmt.State, verb rune) {
	switch verb {
	case 's', 'q':
		val := ai.String()
		if verb == 'q' {
			val = fmt.Sprintf("%q", val)
		}
		fmt.Fprint(state, val)
	case 'v':
		if state.Flag('#') {
			// Emit type before
			fmt.Fprintf(state, "%T", ai)
		}
		fmt.Fprint(state, "{")
		val := reflect.ValueOf(*ai)
		for i, name := range authInfoFields {
			if state.Flag('#') || state.Flag('+') {
				fmt.Fprintf(state, "%s:", name)
			}
			if name == "APIKey" {
				fmt.Fprint(state, keyMask)
			} else {
				fmt.Fprint(state, val.FieldByName(name))
			}
			if i < len(authInfoFields)-1 {
				fmt.Fprint(state, " ")
			}
		}
		fmt.Fprint(state, "}")
	}
}
```

Let's try it out:
```go
ai := &AuthInfo{
	Login:  "daffy",
	ACL:    ReadACL | WriteACL,
	APIKey: "duck season",
}
fmt.Println(ai.String())
fmt.Printf("ai %%s: %s\n", ai)
fmt.Printf("ai %%q: %q\n", ai)
fmt.Printf("ai %%v: %v\n", ai)
fmt.Printf("ai %%+v: %+v\n", ai)
fmt.Printf("ai %%#v: %#v\n", ai)
```

which will emit
```
Login:daffy, ACL:00000011, APIKey: *****
ai %s: Login:daffy, ACL:00000011, APIKey: *****
ai %q: "Login:daffy, ACL:00000011, APIKey: *****"
ai %v: {3 ***** daffy}
ai %+v: {ACL:3 APIKey:***** Login:daffy}
ai %#v: *main.AuthInfo{ACL:3 APIKey:***** Login:daffy}
```

To summarize, the `fmt` package has many capabilities other than the trivial
use. Once you'll familiarize yourself with these capabilities, I'm sure you find
many interesting uses for them.
