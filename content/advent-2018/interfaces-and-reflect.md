+++
title = "The Relationship Between Interfaces and Reflection"
date = "2018-12-16T00:00:00+00:00"
series = ["Advent 2018"]
author = ["Ayan George"]
+++

Interfaces are one of the fundamental tools for abstraction in Go. Interfaces
store type information when assigned a value.  Reflection is a method of
examining type and value information at runtime.

Go implements reflection with the `reflect` package which provides types and
methods for inspecting portions of the interface structure and even modifying
values at runtime.

With this post I hope to illustrate how parts of the interface structure relate
to the reflect API and ultimately make using the reflect package more
approachable!

# Assigning a Value to an Interface

An interface encodes three things: a value, a method set, and the type of the
stored value.

The structure for an interface looks like the following:

![interface-diagram](/postimages/advent-2018/interfaces-and-reflect/interface.svg)

We can clearly see the three parts of the interface in that diagram: the
`_type` is type information, `*data` is a pointer to the actual value, and the
`itab` encodes the method set.

When a function accepts an interface as a parameter, passing a value to that
function packs the value, method set, and type into the interface.

# Examining Interface Data At Runtime with the Reflect Package

Once a value is stored in an interface, you can use the `reflect` package to
examine its parts.  We can't examine the interface struct directly; instead the
reflect package maintains its own copies of the interface structure to which we
do have access.

Even though we're accessing the interface via reflect objects, there's a direct
correlation to the underlying interface.

The `reflect.Type` and `reflect.Value` types  provide methods to access
portions of the interface.

`reflect.Type` focuses on exposing data about types and is therefore confined
to the `_type` portion of the structure while `reflect.Value` has to combine
type information with the value to allow programmers to examine and manipulate
values and therefore has to peek into the `_type` as well as the `data`.

## reflect.Type -- Examining Types

The `reflect.TypeOf()` function is used to extract type information for a
value.  Since its only parameter is an empty interface, the value passed to it
gets assigned to an interface and therefore the type, methodset, and value
become available.

`reflect.TypeOf()` returns a `reflect.Type` which has methods that allow you
to example the value's type.

Below are a few of the `Type` methods available and their corresponding bits of
the interface that they return.

![reflect-type-diagram](/postimages/advent-2018/interfaces-and-reflect/reflect-type.svg)

### An Example `reflect.Type` Usage

```Go
package main

import (
	"log"
	"reflect"
)

type Gift struct {
	Sender    string
	Recipient string
	Number    uint
	Contents  string
}

func main() {
	g := Gift{
		Sender:    "Hank",
		Recipient: "Sue",
		Number:    1,
		Contents:  "Scarf",
	}

	t := reflect.TypeOf(g)

	if kind := t.Kind(); kind != reflect.Struct {
		log.Fatalf("This program expects to work on a struct; we got a %v instead.", kind)
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		log.Printf("Field %03d: %-10.10s %v", i, f.Name, f.Type.Kind())
	}
}
```

The purpose of this program is to print the fields in our `Gift` struct.  When
the `g` value is passed to `reflect.TypeOf()`, `g` is assigned to an interface
which the compiler populates with type and method set information.  This allows
us to walk the `[]fields` of the type portion of the interface structure and we get
the following:

```
2018/12/16 12:00:00 Field 000: Sender     string
2018/12/16 12:00:00 Field 001: Recipient  string
2018/12/16 12:00:00 Field 002: Number     uint
2018/12/16 12:00:00 Field 003: Contents   string
```

## reflect.Method - Examining the itab/Method-Set

The `reflect.Type` type also allows you to access portions of the `itab` to
extract method information from the interface.

![reflect-method-diagram](/postimages/advent-2018/interfaces-and-reflect/reflect-method.svg)

### Examining Methods with Reflect

```Go
package main

import (
	"log"
	"reflect"
)

type Reindeer string

func (r Reindeer) TakeOff() {
	log.Printf("%q lifts off.", r)
}

func (r Reindeer) Land() {
	log.Printf("%q gently lands.", r)
}

func (r Reindeer) ToggleNose() {
	if r != "rudolph" {
		panic("invalid reindeer operation")
	}
	log.Printf("%q nose changes state.", r)
}

func main() {
	r := Reindeer("rudolph")

	t := reflect.TypeOf(r)

	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		log.Printf("%s", m.Name)
	}
}
```

This code quite literally iterates over the function data stored in the `itab`
and displays the name of each method:

```
2018/12/16 12:00:00 Land
2018/12/16 12:00:00 TakeOff
2018/12/16 12:00:00 ToggleNose
```

## reflect.Value -- Examining Values

So far we've only talked about type information -- fields, methods, etc.
`reflect.Value` gives us information about the actual value stored by an
interface.

Methods associated with `reflect.Value`s necessarily combine type information
with the actual value.  For example, in order to extract fields from a struct,
the reflect package has to combine knowledge of the layout of the struct --
particularly information about the fields and field offsets stored in the
`_type` -- with the actual value pointed to by the `*data` portion of the
interface in order to properly decode the struct.

![reflect-value-diagram](/postimages/advent-2018/interfaces-and-reflect/reflect-value.svg)

### Example of Viewing an Modifying Values

```Go
package main

import (
	"log"
	"reflect"
)

type Child struct {
	Name  string
	Grade int
	Nice  bool
}

type Adult struct {
	Name       string
	Occupation string
	Nice       bool
}

// search a slice of structs for Name field that is "Hank" and set its Nice
// field to true.
func nice(i interface{}) {
	// retrieve the underlying value of i.  we know that i is an
	// interface.
	v := reflect.ValueOf(i)

	// we're only interested in slices to let's check what kind of value v is. if
	// it isn't a slice, return immediately.
	if v.Kind() != reflect.Slice {
		return
	}

	// v is a slice.  now let's ensure that it is a slice of structs.  if not,
	// return immediately.
	if e := v.Type().Elem(); e.Kind() != reflect.Struct {
		return
	}

	// determine if our struct has a Name field of type string and a Nice field
	// of type bool
	st := v.Type().Elem()

	if nameField, found := st.FieldByName("Name"); found == false || nameField.Type.Kind() != reflect.String {
		return
	}

	if niceField, found := st.FieldByName("Nice"); found == false || niceField.Type.Kind() != reflect.Bool {
		return
	}

	// Set any Nice fields to true where the Name is "Hank"
	for i := 0; i < v.Len(); i++ {
		e := v.Index(i)
		name := e.FieldByName("Name")
		nice := e.FieldByName("Nice")

		if name.String() == "Hank" {
			nice.SetBool(true)
		}
	}
}

func main() {
	children := []Child{
		{Name: "Sue", Grade: 1, Nice: true},
		{Name: "Ava", Grade: 3, Nice: true},
		{Name: "Hank", Grade: 6, Nice: false},
		{Name: "Nancy", Grade: 5, Nice: true},
	}

	adults := []Adult{
		{Name: "Bob", Occupation: "Carpenter", Nice: true},
		{Name: "Steve", Occupation: "Clerk", Nice: true},
		{Name: "Nikki", Occupation: "Rad Tech", Nice: false},
		{Name: "Hank", Occupation: "Go Programmer", Nice: false},
	}

	log.Printf("adults before nice: %v", adults)
	nice(adults)
	log.Printf("adults after nice: %v", adults)

	log.Printf("children before nice: %v", children)
	nice(children)
	log.Printf("children after nice: %v", children)
}
```

```
2018/12/16 12:00:00 adults before nice: [{Bob Carpenter true} {Steve Clerk true} {Nikki Rad Tech false} {Hank Go Programmer false}]
2018/12/16 12:00:00 adults after nice: [{Bob Carpenter true} {Steve Clerk true} {Nikki Rad Tech false} {Hank Go Programmer true}]
2018/12/16 12:00:00 children before nice: [{Sue 1 true} {Ava 3 true} {Hank 6 false} {Nancy 5 true}]
2018/12/16 12:00:00 children after nice: [{Sue 1 true} {Ava 3 true} {Hank 6 true} {Nancy 5 true}]
```

In this last example, we combine what we've learned to actually modify a value
via a `reflect.Value`.  In this case, someone wrote a function called `nice()`
(probably Hank) that will toggle any struct item in a slice from naughty to
nice where the name is "Hank".

Notice that `nice()` is able to modify the value of any slice you pass to it
and it doesn't matter exactly what type it receives -- as long as it is a slice
of a struct that has a `Name` and `Nice` field.

# Conclusion

Reflection in Go is implemented using interfaces and the `reflect` package.
There's no magic to it -- when you use reflection, you directly access parts of
an interface and values stored within.

In this way an interface almost behaves like a mirror, allowing a program to
examine itself.

Though Go is a staticly-typed language, reflection and interfaces combine to
provide extremely powerful techniques that are usually reserved for dynamic
lanugages.

For more information about reflection in Go, definitely read the package
documentation as well as the many other great blog posts on the subject.

# About the Author

Yo! I'm Ayan and I hope you enjoyed my blog post!  Part of my motivation for
writing this is to learn more about Go myself. In that spirit, if you have
anything comments or suggestions, please feel free to contact me at
[ayan@ayan.net](mailto:ayan@ayan.net) or
[@ayangeorge](https://twitter.com/ayangeorge) on twitter.
