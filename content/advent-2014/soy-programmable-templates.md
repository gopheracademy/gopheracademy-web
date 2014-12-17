+++
author = ["Rob Figueiredo"]
date = "2014-12-17T08:00:00+00:00"
title = "Soy - Programmable templates for Go, Java, JS"
series = ["Advent 2014"]
+++

[Closure Templates](https://developers.google.com/closure/templates/) (aka Soy
Templates) is a client and server-side templating language developed at Google.
The [Go implementation](http://godoc.org/github.com/robfig/soy) exposes the the
internal structure of the template (the
[AST](http://en.wikipedia.org/wiki/Abstract_syntax_tree)). This article
highlights a couple of interesting applications where we've benefited from being
able to programmatically inspect and modify this structure.

# Background

Earlier this year, we developed a system for publishing a web site based on
information in our CMS (content management system). The pages are regenerated
when any relevant information changes. Efficiency is important so that
developers working on the site can see their updates reflected quickly, so
that the system can keep up with the large number of updates flying around, and
of course to impress the client.

We chose Closure Templates for this project for a few reasons:

* Use the same templates from Java, JS, and Go - As a former Java shop, having
  one template language work across the system is wonderful.
* Internationalization support - Clients want localized versions of their web
  sites. Language and tooling support makes it relatively painless.
* Great documentation / easy to learn - Outside developers would be working on
  the site templates, not us. Closure Templates has been around for a long
  time, and the documentation site is solid.

This article assumes basic familiarity with the syntax.  It may be helpful to
read through [the introductory example](http://godoc.org/github.com/robfig/soy)
first, if you haven't seen it before.

# Inspecting a template

Let's look at example code that prints a simple template's AST. It involves 3 different types:

* The
[template.Registry](http://godoc.org/github.com/robfig/soy/template#Registry) is
the top-level type returned by the template compiler
* [ast.Node](https://github.com/robfig/soy/blob/master/ast/node.go#L15) is the
  standard interface implemented by all elements of the AST
* [ast.ParentNode](https://github.com/robfig/soy/blob/master/ast/node.go#L22) is
  implemented by all nodes that contain other nodes

Here's the code
```go
package main

import (
	"fmt"
	"strings"
	"github.com/robfig/soy"
	"github.com/robfig/soy/ast"
)

const example = `
{namespace example}

/** @param name */
{template .helloWorld}
 Hello {$name ?: 'world'}
{/template}`

func main() {
	registry, _ := soy.NewBundle().
		AddTemplateString("example", example).
		Compile()
	for _, t := range registry.Templates {
		fmt.Println("Template:", t.Node.Name)
		fmt.Println("Params:", t.Doc.Params)
		walk(t.Node, 0)
	}
}

func walk(node ast.Node, indent int) {
	fmt.Printf("%s%T\n", strings.Repeat("\t", indent), node)
	if parent, ok := node.(ast.ParentNode); ok {
		for _, child := range parent.Children() {
			walk(child, indent+1)
		}
	}
}
```

The above program produces the output:
```
$ go run ~/test.go
Template: example.helloWorld
Params: [@param name]
*ast.TemplateNode
	*ast.ListNode
		*ast.RawTextNode
		*ast.PrintNode
			*ast.ElvisNode
				*ast.DataRefNode
				*ast.StringNode
```

With just a small modification to the program, we could do something like
process every data reference in the template (`$name`).

Let's see one application of this technique by the web publishing system.

# Lazily load source data

## Page templates

A client's web page template looks something like this

```
/**
 * @param name
 * @param address
 * @param city
 * @param state
 * ... (~30 more params) ...
 */
{template .location}
<!doctype html>
<html>
  <head>
    <title>{$name} - {$address1}</title>
  ...
{/template}
```

The client's web site templates are written using Closure Templates, with a list
of parameters covering all of the available content.  When any of the content
changes, the relevant pages should be automatically regenerated and deployed.

Beyond simple values like name and address, the business location page may use a
 lot of information across the system.  For example:

- Show the N nearest business locations to this location
- Show the N latest Facebook posts or Instagram tags
- Show the current list of featured products at this location
- Show links to any of their listings (e.g. their Yelp page)

In a system organized around function, RPCs are required to fetch the
information from the owning system. Especially when regenerating the entire site
(thousands of locations), it is desirable to avoid loading data unless we really
need it.

Here's a quick overview of our solution

## Data sources and usage

We created a type to track the various data sources. It's a bitmask to represent
the set of possible data sources that may be used.

```go
// DataSource is an enumeration of sources (beyond core profile data) that may
// be queried by a location template.
type DataSource uint16

const (
	ECLs     DataSource = 1 << iota // Enhanced Content Lists
	Posts                           // Social Posts
	Photos                          // Photos (By Label)
	Nearby                          // Nearby Locations
	Listings                        // Listings
	...
)

// DataUsage tracks which data sources are used by a particular template.
type DataUsage struct {
	sources     DataSource
}

// Has queries whether or not the specified data source is marked as used.
func (u DataUsage) Has(source DataSource) bool { return (u.sources & source) != 0 }
```

In order to create a `DataUsage`, we read the list of params:

```go
// UsageOf deduces the location data required by a soy template by analyzing
// its parameters for known special names.
func UsageOf(template template.Template) (DataUsage, error) {
	var usage DataUsage
	for _, param := range template.Doc.Params {
		switch param.Name {
		case "productLists", "calendars", "bios", "menus":
			usage.sources |= ECLs
		case "posts":
			usage.sources |= Posts
		case "nearby":
			usage.sources |= Nearby
		case "listings":
			usage.sources |= Listings
		case "photos":
			usage.sources |= Photos
		...
```

Now that we know what data the template needs, we just have to edit the data
loading code to be a bit lazier.

## Be lazy

Now, it's easy to only load data that's used by the template.

```go
	var (
		loc      *profile.Location
		ecls     []*enhancedlists.ListProto
		posts    []pagedata.Post
		nearby   []NearbyLocation
		...
	)

	loc = loader.Profile(id)
	if usage.Has(ECLs) {
		ecls = loader.Lists(loc)
	}
	if usage.Has(Posts) {
		posts = loader.Posts(id)
	}
	...
```

## RPCs via Map lookup

The optimization I've described so far relies just on reading the parameters to
a template.  Here's one that actually needs to inspect the template.

1. The template has access to a map which it can use to look up photo assets in
   the account given a label. For example, `photosByLabel['storefront']` would
   return the photos labeled "storefront"

2. This information is accessible via RPC to our photo search service.

3. We inspect the template to see which labels are requested, and we load the
   results in bulk ahead of time.

Here is how it may be used in a template

```
{foreach $photo in $photosByLabel['storefront']}
  <div class="storefront-photo">
    <img height="{$photo.height}" width="{$photo.width}" src="{$photo.url}"/>
    <span class="caption">{$photo.caption}</span>
  </div>
{/foreach}
```

Here is a function that extracts the map references from the templates, or
returns an error if the map is accessed by something other than a string
constant.


```go
// extractPhotoLabels adds label text found in expressions of the form
// photosByLabel['label'] to the given set.
// The node traversal we use to access the label is the following:
// *ast.DataRefNode.Access[0].(*ast.DataRefExprNode).Arg.(*ast.StringNode).Value
func extractPhotoLabels(node ast.Node, labelPhotoKeySet map[string]struct{}) error {
	if dataRef, ok := node.(*ast.DataRefNode); ok && dataRef.Key == photosParam {
		if len(dataRef.Access) == 0 {
			return newErr(dataRef.String())
		}
		var exprNode, ok = dataRef.Access[0].(*ast.DataRefExprNode)
		if !ok {
			return newErr(dataRef.String())
		}
		stringNode, ok := exprNode.Arg.(*ast.StringNode)
		if !ok {
			return newErr(exprNode.String())
		}
		labelPhotoKeySet[stringNode.Value] = struct{}{}
	}

	if parent, ok := node.(ast.ParentNode); ok {
		for _, child := range parent.Children() {
			if err := extractPhotoLabels(child, labelPhotoKeySet); err != nil {
				return err
			}
		}
	}
	return nil
}
```

This solution is superior to the usual alternative of writing a template
function that directly issues the RPC, because we may be rendering the same
template thousands of times and loading the information in bulk provides a
dramatic speedup.

In just a few lines of code, we've shown how template authors can access data
from across the system, while we arrange for just the data that's used to be
efficiently loaded and provided to the template.

# Conclusion

This article covers just the tip of a web publishing iceberg that we've built in
Go (30k+ LOC). Soon we'll be extending it to support international/localized
versions of client sites, automatic compilation of Soy to JS for client-side
rendering, and other fun stuff.

The super fast builds, great tooling, and simple yet effective language
primitives have made it great fun to develop in Go.  The end result performs
very well and is easy to maintain.

If you'd like to join a small motivated team in NYC building software used by
the largest brands in the world, message me
[@robfig](https://twitter.com/robfig), or check us out at
http://www.yext.com/company/careers/engineering/.
