+++
author = ["Dmitri Shuralyov"]
date = "2017-12-08T09:15:00Z"
title = "Custom JSON unmarshaler for a GraphQL client"
series = ["Advent 2017"]
+++

In this post, I will tell a story of how I had to build a custom JSON unmarshaler for the needs of a GraphQL client library in Go. I'll start with the history of how everything started, build motivation for why a custom JSON marshaler was truly needed, and then describe how it was implemented. This is going to be a long journey, so strap yourself in, and here we go!

History of GraphQL Client Library in Go
---------------------------------------

In May of 2017, I set out to build the first [GraphQL client library for Go](http://graphql.org/code/#go-1). Back then, only [GraphQL server libraries](http://graphql.org/code/#go) existed in Go. My main motivation was wanting to be able to access [GitHub GraphQL API v4](https://developer.github.com/v4/) from my Go code, which had just [come out of early access](https://github.com/blog/2359-introducing-github-marketplace-and-more-tools-to-customize-your-workflow) back then. I also knew that a general-purpose GraphQL client library would be useful, enabling Go projects to access any GraphQL API. There were GraphQL clients available in other languages, and I didn't want Go users to be missing out.

[GraphQL](http://graphql.org/) is a data query language for APIs, developed internally by Facebook in 2012, and made publicly available in 2015. It can be used as a replacement for, or in addition to REST APIs. It some ways, it offers significant advantages compared to REST APIs, making it an attractive option. Of course, as any newer technology, it's less mature and has some weaknesses in certain areas.

I spent a week on the initial investigation and research into what a Go client for GraphQL could look like. GraphQL is strongly typed, which is a good fit for Go. However, it also has some more advanced query syntax and features that play better with more dynamic languages, so I had my share of concerns whether a good client in Go would even be viable. Fortunately, at the end of that week, I found that a reasonable client in Go was indeed possible, and pushed a working [initial prototype](https://github.com/shurcooL/githubql/commit/78a7455460db3b5f51a2ec5640d7e47326a9ef12) that had most basic functionality implemented, with a plan for how to implement and address the [remaining features](https://github.com/shurcooL/githubql/issues/22).

I documented the history of my findings and design decisions made in [this issue](https://github.com/google/go-github/issues/646). I've also given a [talk](https://www.youtube.com/watch?v=mEqJbeAazow) (slides [here](https://dmitri.shuralyov.com/talks/2017/githubql/githubql.slide)) about the rationale and thinking behind many of the API and design decisions that went into the library. In this post, I want to talk about something I haven't covered before: implementing a custom JSON unmarshaler specifically for the needs of the GraphQL client library, in order to improve support for GraphQL unions.

JSON Unmarshaling Task at Hand
------------------------------

Unmarshaling JSON into a structure is a very common and well understood problem. It's already implemented inside the `encoding/json` package in Go standard library. Given that JSON is such a well specified standard, why would I need to implement my own JSON unmarshaler?

To answer that, I need to provide a little context about how GraphQL works. The GraphQL client begins by sending a request containing a GraphQL query, for example:

```GraphQL
query {
	me {
		name
		bio
	}
}
```

The GraphQL server receives it, processes it, and sends a JSON-encoded response for that query. The response contains a `data` object and potentially other miscellaneous fields. We're primarily interested in the `data` object, which looks like this:

```JSON
{
	"me": {
		"name": "gopher",
		"bio": "The Go gopher."
	}
}
```

Notice it has the same shape as the query. Taking advantage of this property turned out to be a critical factor in making the Go GraphQL client library convenient and useful.

That's why the `graphql` package was designed so that to make a query, you start by defining a Go struct variable. That variable then both defines the GraphQL query that will be made, and gets populated with the response data from the GraphQL server:

```Go
var query struct {
	Me struct {
		Name string
		Bio  string
	}
}
err := client.Query(context.Background(), &query, nil)
if err != nil {
	// Handle error.
}
fmt.Println(query.Me.Name)
fmt.Println(query.Me.Bio)

// Output:
// gopher
// The Go gopher.
```

Initially, `encoding/json` was used for unmarshaling the GraphQL response into the query structure and it worked well. But eventually, some edge cases and advanced queries were discovered, where using `encoding/json` was no longer working out.

Motivation for Custom JSON Unmarshaler
--------------------------------------

There were at least 3 clear problems with `encoding/json` for unmarshaling GraphQL responses into the query structure. These served as motivation to write a custom JSON unmarshaler for `graphql` needs.

1.	Consider if the user supplied a query struct that happened to contain `json` struct field tags, for example:

	```Go
	type query struct {
		Me struct {
			Name string `json:"full_name"`
		}
	}
	```

	(Suppose the user wants to serialize the response later, or uses some struct that happens to have `json` tags defined for other reasons.)

	The JSON-encoded response from GraphQL server will contain:

	```JSON
	{
		"me": {
			"name": "gopher"
		}
	}
	```

	As a result, `query.Me.Name` will not be populated, since the Go struct has a JSON tag calling it "full_name", but the field is "name" in the response, which doesn't match.

	This happens because `encoding/json` unmarshaler is affected by `json` struct field tags.

2.	To have additional control over the GraphQL query that is generated from the query struct, the `graphql` struct field tag can be used. It allows overriding how a given struct field gets encoded in the GraphQL query. Suppose the user happens to use a field with a name that doesn't match that of the GraphQL field:

	```Go
	var query struct {
		Me struct {
			Photo string `graphql:"avatarUrl(size: 72)"`
		}
	}
	```

	The JSON-encoded response from GraphQL server will contain:

	```JSON
	{
		"me": {
			"avatarUrl": "https://golang.org/doc/gopher/run.png"
		}
	}
	```

	As a result, `query.Me.Photo` will not be populated, since the field is "avatarUrl" in the response, and the Go struct has a field named "Photo", which doesn't match.

	This happens because `encoding/json` unmarshaler is unaware of the `graphql` struct field tags.

3.	Perhaps the largest problem with using `encoding/json` came to light when looking to support the GraphQL [unions](https://facebook.github.io/graphql/October2016/#sec-Unions) feature. In GraphQL, a union is a type of object representing many objects.

	```GraphQL
	query {
		mascot(language: "Go") {
			... on Human {
				name
				height
			}
			... on Animal {
				name
				hasTail
			}
		}
	}
	```

	In this query, we're asking for information about Go's mascot. We don't know in advance what exact type it is, but we know what types it can be. Depending on whether it's an Animal or Human, we ask for additional fields of that type.

	To express that GraphQL query, you can create the following query struct:

	```Go
	var query struct {
		Mascot struct {
			Human struct {
				Name   string
				Height float64
			} `graphql:"... on Human"`
			Animal struct {
				Name    string
				HasTail bool
			} `graphql:"... on Animal"`
		} `graphql:"mascot(language: \"Go\")"`
	}
	```

	The JSON-encoded response from GraphQL server will contain:

	```JSON
	{
		"mascot": {
			"name": "Gopher",
			"hasTail": true
		}
	}
	```

	You can see that in this case the shape of the response doesn't quite align with the query struct. GraphQL inlines or embeds the fields from Animal into the "mascot" object. The `encoding/json` unmarshaler will not be able to handle that in the way we'd want, and the fields in the query struct will be left unset. See proof on the [playground](https://play.golang.org/p/ug4T4Tt4n2).

	You could try to work around it by using Go's embedded structs. If you define query as:

	```Go
	type Human struct {
		Name   string
		Height float64
	}
	type Animal struct {
		Name    string
		HasTail bool
	} `graphql:"... on Animal"`
	var query struct {
		Mascot struct {
			Human  `graphql:"... on Human"`  // Embedded struct.
			Animal `graphql:"... on Animal"` // Embedded struct.
		} `graphql:"mascot(language: \"Go\")"`
	}
	```

	That gets you almost the right results, but there's a significant limitation at play. Both Human and Animal structs have a field with the same name, `Name`.

	According to the `encoding/json` unmarshaling rules:

	> If there are multiple fields at the same level, and that level is the least nested (and would therefore be the nesting level selected by the usual Go rules), the following extra rules apply:
	>
	> 1.	Of those fields, if any are JSON-tagged, only tagged fields are considered, even if there are multiple untagged fields that would otherwise conflict.
	>
	> 2.	If there is exactly one field (tagged or not according to the first rule), that is selected.
	>
	> 3.	Otherwise there are multiple fields, and all are ignored; no error occurs.

	Multiple fields are ignored. So, `Name` would be left unset. See proof on the [playground](https://play.golang.org/p/qT7n2P0sSk).

	An initial reaction might be that it's a bug or flaw in `encoding/json` package and should be fixed. However, upon careful consideration, this is a very ambiguous situation, and there's no single clear "correct" behavior. The `encoding/json` unmarshaler makes a sensible compromise for generic needs, not GraphQL-specific needs.

This motivation lead to the conclusion that for GraphQL-specific needs, a custom JSON unmarshaler is unavoidably needed.

Implementing a Custom JSON Unmarshaler
--------------------------------------

Discarding a well written, thoroughly tested, battle proven JSON unmarshaler in the Go standard library and writing one from scratch is not a decision to be taken lightly. I spent considerable time looking at my options and comparing their trade-offs.

Writing it from scratch would've been the last option to consider. I could've made a copy of `encoding/json` and modified it. But that would mean having to maintain a copy of `encoding/json` and keep it up to date with any upstream changes.

Luckily, I found a better option. The key insight was that the process of JSON unmarshaling consists of two independent parts: parsing JSON, and populating the target struct fields with the parsed values. The JSON that GraphQL servers respond with is completely standard, specification-compliant JSON. I didn't need to make any changes there. It was only the behavior of populating target struct fields that I needed to customize.

In Go 1.5, the `encoding/json` package API was expanded to expose a JSON tokenizer to the outside world. A JSON tokenizer parses JSON and returns a sequence of JSON tokens, which are higher-level and easier to work with compared to the original byte stream. I could make use of this to avoid having to parse the JSON myself.

The `encoding/json` JSON tokenizer is exposed via a [`Token`](https://godoc.org/encoding/json#Decoder.Token) method of `json.Decoder` struct:

```Go
// Token returns the next JSON token in the input stream.
// At the end of the input stream, Token returns nil, io.EOF.
//
// ...
func (dec *Decoder) Token() (Token, error)
```

Calling `Token` repeatedly on an input like this:

```JSON
{
	"Message": "Hello",
	"Array": [1, 2, 3],
	"Number": 1.234
}
```

Returns a sequence of JSON tokens, followed by io.EOF error:

```
json.Delim: {
string: "Message"
string: "Hello"
string: "Array"
json.Delim: [
float64: 1
float64: 2
float64: 3
json.Delim: ]
string: "Number"
float64: 1.234
json.Delim: }
io.EOF error
```

Great! We don't have to deal with all the low-level nuances of parsing JSON strings, escaped characters, quotes, floating point numbers, and so on. We'll be able to reuse the JSON tokenizer from `encoding/json` for all that. Now, we just need to build our unmarshaler on top of it.

Let's start by defining and iterating on our `decoder` struct that contains the necessary state. We know we're going to base it on a JSON tokenizer. To make it very clear we're only ever using the `Token` method and nothing else from `json.Decoder`, we can make it a small interface. This is our starting point:

```Go
// decoder is a JSON decoder that performs custom unmarshaling behavior
// for GraphQL query data structures. It's implemented on top of a JSON tokenizer.
type decoder struct {
	tokenizer interface {
		Token() (json.Token, error)
	}
}
```

And the exported unmarshal function will look like this:

```Go
// UnmarshalGraphQL parses the JSON-encoded GraphQL response data and stores
// the result in the GraphQL query data structure pointed to by v.
//
// The implementation is created on top of the JSON tokenizer available
// in "encoding/json".Decoder.
func UnmarshalGraphQL(data []byte, v interface{}) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	err := (&decoder{tokenizer: dec}).Decode(v)
	return err
}
```

We create a new JSON decoder around `data`, which will act as our JSON tokenizer. Then we decode a single JSON value into `v`, and return error, if any.

### Pop Quiz: Is there a difference in behavior between unmarshaling and decoding a single JSON value?

Here's a pop quiz. Suppose you have some JSON data and and you're looking to unmarshal it into a Go variable. You could do one of two things:

```Go
err := json.Unmarshal(data, &v)
```

```Go
err := json.NewDecoder(r).Decode(&v)
```

They have slightly different signatures; `json.Unmarshal` takes a `[]byte` while `json.NewDecoder` accepts an `io.Reader`. We know that the decoder is meant to be used on streams of JSON values from a reader, but if we only care about reading one JSON value, is there any difference in behavior between them?

In other words, is there an input for which the two would behave differently? If so, what would such an input be?

This was something I didn't quite know the answer to, not before I set out on this journey. But now it's very clear. Yes, the behavior indeed differs: it differs in how the two handle the remaining data after the first JSON value. `Decode` will read just enough to decode the JSON value and stop there. `Unmarshal` will do the same, but it doesn't stop there; it continues reading to check there's no extraneous data following the first JSON value (other than whitespace). If there are any additional JSON tokens, it returns an "invalid token after top-level value" error.

---

To stay true to unmarshaling behavior, we perform a check to ensure there are no additional JSON tokens following our top-level JSON value; if there is, that's an error:

```Go
func UnmarshalGraphQL(data []byte, v interface{}) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	err := (&decoder{tokenizer: dec}).Decode(v)
	if err != nil {
		return err
	}
	tok, err := dec.Token()
	switch err {
	case io.EOF:
		// Expect to get io.EOF. There shouldn't be any more
		// tokens left after we've decoded v successfully.
		return nil
	case nil:
		return fmt.Errorf("invalid token '%v' after top-level value", tok)
	default:
		return err
	}
}
```

Ok, now let's figure out all the remaining state we need to keep track of in the `decoder` struct. We will implement unmarshaling with an iterative algorithm rather than recursive, and keep all relevant state in `decoder` struct.

We know that the JSON tokenizer provides us with one token at a time. So, it's up to us to track whether we're in the middle of a JSON object or array. Imagine you get a `string` token. If the preceding token was `[`, then this string is an element of an array. But if the preceding token was `{`, then this string is the key of an object, and the following token will be its value. We'll use `parseState json.Delim` to track that.

We'll also keep a reference to the value where we want to unmarshal JSON into, say, a `v reflect.Value` field (short for "value").

What we have so far is:

```Go
type decoder struct {
	tokenizer interface {
		Token() (json.Token, error)
	}

	// What part of input JSON we're in the middle of - object, array. Zero value means neither.
	parseState json.Delim

	// Value where to unmarshal.
	v reflect.Value
}
```

That's a good start, but what happens when we encounter a `]` or `}` token? That means we leave the current array or object, and... end up in the parent, whatever that was, if any.

JSON values can be nested. Objects inside arrays inside other objects. We will change `parseState` to be a stack of states `parseState []json.Delim`. Whenever we get to the beginning of a JSON object or array, we push to the stack, and when we get to end, we pop off the stack. Top of the stack is always the current state.

We need to apply the same change to `v`, so we know where to unmarshal into after end of array or object. We'll also make it a stack and rename to `vs []reflect.Value` (short for "values").

Now we have something that should be capable of unmarshaling deeply nested JSON values:

```Go
type decoder struct {
	tokenizer interface {
		Token() (json.Token, error)
	}

	// Stack of what part of input JSON we're in the middle of - objects, arrays.
	parseState []json.Delim

	// Stack of values where to unmarshal.
	// The top of stack is the reflect.Value where to unmarshal next JSON value.
	vs []reflect.Value
}
```

We'll create these helpers to help manage the `parseState` stack:

```Go
// pushState pushes a new parse state s onto the stack.
func (d *decoder) pushState(s json.Delim) {
	d.parseState = append(d.parseState, s)
}

// popState pops a parse state (already obtained) off the stack.
// The stack must be non-empty.
func (d *decoder) popState() {
	d.parseState = d.parseState[:len(d.parseState)-1]
}

// state reports the parse state on top of stack, or 0 if empty.
func (d *decoder) state() json.Delim {
	if len(d.parseState) == 0 {
		return 0
	}
	return d.parseState[len(d.parseState)-1]
}
```

The `popState` helper happens to be called only when stack is known to be non-empty, so there's no need to have it check for that condition. We couldn't do that if it weren't an unexported helper.

That should be enough for now. Let's look at the code for unmarshaling next.

Remember that the `UnmarshalGraphQL` function calls `decoder.Decode` method. `Decode` will accept `v`, set up the decoder state, and call `decode`, where the actual decoding logic will take place.

```Go
// Decode decodes a single JSON value from d.tokenizer into v.
func (d *decoder) Decode(v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("cannot decode into non-pointer %T", v)
	}
	d.vs = []reflect.Value{rv.Elem()}
	return d.decode()
}
```

`decode` is implemented as an iterative algorithm that uses the state in `decoder` struct. This is entire algorithm at a high level:

```Go
// decode decodes a single JSON value from d.tokenizer into d.vs.
func (d *decoder) decode() error {
	// The loop invariant is that the top of the d.vs stack
	// is where we try to unmarshal the next JSON value we see.
	for len(d.vs) > 0 {
		tok, err := d.tokenizer.Token()
		if err == io.EOF {
			return io.ErrUnexpectedEOF
		} else if err != nil {
			return err
		}

		// Process the token. Potentially decode a JSON value,
		// or handle one of {, }, [, ] tokens.
		switch tok := tok.(type) {
			...
		}
	}
	return nil
}
```

There's a big outer loop. At the top of the loop, we call `d.tokenizer.Token` to get the next JSON token. The loop invariant is that the top of the `vs` stack is where we unmarshal the next JSON value we get from `Token`. The loop condition is `len(d.vs) > 0`, meaning we have some value to unmarshal into. When the `vs` stack becomes empty, that means we've reached the end of the JSON value we're decoding, so we break out and return `nil` error.

Each loop iteration makes a call to `Token` and processes the token:

-	If it's a value, it's unmarshaled into the value at the top of `vs` stack.
-	If it's an opening of an array or object, then the `parseState` and `vs` stacks are pushed to.
-	If it's the ending of an array or object, those stacks are popped.

That's basically it. The rest of the code are the details, managing the `parseState` and `vs` stacks, checking for `graphql` struct field tags, handling all the error conditions, etc. But the algorithm is conceptually simple and easy to understand at this high level.

Except... We're still missing one critical aspect of making it handle the GraphQL-specific needs that we set out to resolve originally.

Let's recall the GraphQL unions example, where the JSON-encoded GraphQL server response was:

```JSON
{
	"mascot": {
		"name": "Gopher",
		"hasTail": true
	}
}
```

And we're trying to unmarshal contents of `data.mascot` into `query.Mascot`:

```Go
var query struct {
	Mascot struct {
		Human struct {
			Name   string
			Height float64
		} `graphql:"... on Human"`
		Animal struct {
			Name    string
			HasTail bool
		} `graphql:"... on Animal"`
	} `graphql:"mascot(language: \"Go\")"`
}
```

The behavior we want is to unmarshal "Gopher" into all matching fields (rather than none at all). It should get unmarshaled into two values:

-	`query.Mascot.Human.Name`
-	`query.Mascot.Animal.Name`

But the top of our `vs` stack only contains one value... What do we do?

We must go deeper. Cue the music from Inception, and get ready to replace `vs []reflect.Value` with `vs [][]reflect.Value`!

Multiple Stacks of Values
-------------------------

That's right, to be able to deal with having potentially multiple places to unmarshal a single JSON value into, we have a slice of slices of `reflect.Value`s. Essentially, we have multiple (1 or more) `[]reflect.Value` stacks. `decoder` now looks like this:

```Go
type decoder struct {
	tokenizer interface {
		Token() (json.Token, error)
	}

	// Stack of what part of input JSON we're in the middle of - objects, arrays.
	parseState []json.Delim

	// Stacks of values where to unmarshal.
	// The top of each stack is the reflect.Value where to unmarshal next JSON value.
	//
	// The reason there's more than one stack is because we might be unmarshaling
	// a single JSON value into multiple GraphQL fragments or embedded structs, so
	// we keep track of them all.
	vs [][]reflect.Value
}
```

We need to modify `decode` to create additional stacks whenever we encounter an embedded struct or a struct with `graphql:"... on Type"` field tag, do some additional bookkeeping to manage multiple stacks of values, check for additional error conditions if our stacks run empty. Aside from that, the same algorithm continues to work.

I think getting the data structure to contain just enough information to resolve the task was the most challenging part of getting this to work. Once it's there, the rest of the algorithm details fall into place.

If you'd like to learn even more of the low-level details of the implementation, I invite you to look at the [source code](https://github.com/shurcooL/graphql/blob/master/internal/jsonutil/graphql.go) of package [`github.com/shurcooL/graphql/internal/jsonutil`](https://godoc.org/github.com/shurcooL/graphql/internal/jsonutil). It should be easy to read after this post.

Payoff
------

Let's quickly revisit our original GraphQL unions example that wasn't working with standard `encoding/json` unmarshaler. When we replace `json.UnmarshalJSON` with `jsonutil.UnmarshalGraphQL`, the `Name` field gets populated! That's good news, it means we didn't do all that work for nothing.

See proof on the [playground](https://play.golang.org/p/Xfu2mqxZ5m).

`jsonutil.UnmarshalGraphQL` also takes `graphql` struct field tags into account when unmarshaling, and doesn't get misled by `json` field tags. If there are additional GraphQL-specific tweaks to unmarshaling behavior that need to be applied in the future, they'll be easy to apply. Best part is we're reusing the rigorous JSON tokenizer of `encoding/json` and its public API, so no need to deal with maintaining a fork.

Conclusion
----------

If you got this far, thanks for following along with me on this journey! I hope you enjoyed it and/or learned something new.

It has been a lot of fun implementing the GraphQL client library for Go, and trying to make the best [API design decisions](https://github.com/shurcooL/githubql/issues?q=label%3A%22API+decision%22). I enjoyed using the tools that Go gives me to tackle this task. Even after using Go for 4 years, I'm still finding Go to be the absolutely most fun programming language to use, and feeling same joy I did back when I was just starting out!

I think GraphQL is an exciting new technology. Its strongly typed nature is a great fit for Go. APIs that are created with it can be a pleasure to use. Keep in mind that GraphQL shines most when you're able to replace multiple REST API calls with a single carefully crafted GraphQL query. This requires high quality and completeness of the GraphQL schema, so not all GraphQL APIs are made equal.

Note that there are [two GraphQL client packages](https://dmitri.shuralyov.com/packages?pattern=...ql) to choose from:

-	[`github.com/shurcooL/graphql`](https://github.com/shurcooL/graphql) is a general-purpose GraphQL client library.
-	[`github.com/shurcooL/githubql`](https://github.com/shurcooL/githubql) is a client library specifically for accessing GitHub GraphQL API v4. It's powered by `graphql` internally.

I've had a chance to actually use `githubql` for real tasks in some of my Go projects, and it was a pleasant experience. That said, their GraphQL API v4 is still missing many things present in [GitHub REST API v3](https://developer.github.com/v3/), so I couldn't do as much with it as I would've [liked](https://platform.github.community/t/3114). They're working on expanding it, and it'll be even better when fully complete.

If you want to play around with GraphQL or take a stab at creating your own API with it, you'll need a GraphQL server library. I would suggest considering the [`neelance/graphql-go`](https://github.com/neelance/graphql-go) project as a starting point (if you want a complete list of options, see [here](http://graphql.org/code/#go)). Then, you can use any [GraphQL client](http://graphql.org/code/#graphql-clients) to execute queries, including the `graphql` package from this post.

If you run into any issues, please report in the issue tracker of the corresponding repository. For anything else, I'm on Twitter as [@shurcooL](https://twitter.com/shurcooL).

Happy holidays, and enjoy using Go (and GraphQL) in the upcoming next year!
