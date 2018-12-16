+++
title = "Making Reflection Clear"
date = "2018-12-16T00:00:00+00:00"
series = ["Advent 2018"]
author = ["Ayan George"]
+++

Interfaces are one of the fundamental tools for abstraction in Go. They store
type information when assigned a value and a program can inspect portions of
the interface structure to examine and even modify itself at runtime using the
`reflect` package.

I've found that understanding a bit about how a feature is implemented can lead
to a better understanding its use.  With this post I hope to illustrate how the
reflect package uses parts of the inteface structure.

# Assigning a Value to an Interface

An inteface encodes three things: a value, a method set, and the type of
the stored value.

The structure for an interface looks like the following:

![interface-diagram](/postimages/advent-2018/interfaces-and-reflect/interface.svg)

When a function accepts an interface, passing a value to that function
effectively assigns that value to the interface at which time the compiler will
store the type, value, and method data.

We can clearly see the three parts of the inteface in that diagram: the `_type`
is type information, `*data` is a pointer to the actual value, and the `itab`
encodes the method set.

# Examining Interface Data At Runtime with the Reflect Package

Once a value is stored in an inteface, you can use the `reflect` package to
examine its parts.  The `reflect.Type` and `reflect.Value` types  provide
methods to access portions of the interface.

`reflect.Type` focuses on operating exposing data about types and is therefore
confined to the `_type` portion of the structure while `reflect.Value` has to
combine type information with the value to allow programmers to examine and
manipulate values and therefore has to peek into the `_type` as well as the
`data`.

## reflect.Type

The `reflect.TypeOf(interface{})` function is used to extract type information
for a value.  Since its only parameter is an empty interface, the value passed
to it gets assigned to an interface and therefore the type, methodset, and
value become available.

`reflect.TypeOf()` returnes a `reflect.Type` which has methods that allow you
to example the value's type.

Below are a few of the `Type` methods available and their corresponding bits of
the interface that they return.

![reflect-type-diagram](/postimages/advent-2018/interfaces-and-reflect/reflect-type.svg)

## reflect.Method

The `reflect.Type` type also allows you to access portions of the `itab` to
extract method information from the interface.

![reflect-method-diagram](/postimages/advent-2018/interfaces-and-reflect/reflect-method.svg)

## reflect.Value

The `reflect.ValueOf()` function 

Values sometimes combine type information with the actual value.  For example,
in order to extract fields from a struct, the reflect package has to combine
knowledge of the layout of the struct -- particularly the field and field
offset data stored in the `_type` -- with the actual value pointed to by the
`*data` portion of the inteface.

![reflect-value-diagram](/postimages/advent-2018/interfaces-and-reflect/reflect-value.svg)


# Conclusion

When using the reflect package you are quite literally accessing portions of
the underlying interface.  In this way, an interface almost behaves like a
mirror which allows a program examine iteself. When using the `reflect`
package, I've learned to think about it in terms.

# About the Author

Yo! I'm Ayan and I hope you enjoyed my blog post!  Part of my motivation for
writing this is to learn more about Go myself. In that spirit, if you have
anything comments or suggestions, please feel free to contact me at
[ayan@ayan.net](mailto:ayan@ayan.net) or
[@ayangeorge](https://twitter.com/ayangeorge) on twitter.
