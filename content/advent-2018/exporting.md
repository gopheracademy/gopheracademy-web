+++
author = ["Seebs"]
title = "Members, Methods, and Interfaces"
linktitle = "members methods interfaces"
date = 2018-12-21T00:00:00Z
series = ["Advent 2018"]
+++

# Opaque Datatypes Considered Awesome

I am periodically reminded that many people don't realize that C actually allows the creation of opaque data types. This is a possibly-surprising side effect of the option of never completing a type. Consider:

```c
/* export.h */
struct foo;
typedef struct foo *foo_t;
foo_t *foo_new();
void foo_free(foo_t);

/* internal.h */
struct foo {
	/* actual members go here */
};
```

API client code includes the `export.h` header, giving it an opaque data type. Client code can declare objects of type `foo_t`, but it can't do much with them. It can't even find out the size of the underlying datatype. The API implementation, by contrast, includes the `internal.h` header, so it has access to the type, and uses it to implement those functions.

This is nice, but it would sometimes be really neat to be able to just let users see *some* of a structure. Go's more fine-grained control of exporting allows more nuanced control, letting you decide on a case by case basis which parts of a type's implementation you want clients to be using. It's also stunningly simple:

A name is exported if, and only if, it begins with a capital letter.

No need to read back up in the code to see whether something's public or private. No need to go look at the definition. If you can see the name, you know whether it's exported. This struck me as basically insane for about ten minutes, and I've loved it ever since.


## Exported Fields, Exported Methods

It can be a bit daunting to try to figure out which parts of a data type's implementation you want to export. If you're not worried about this at all, you probably should be. Any aspect of implementation you expose will become impossible to change without breaking code depending on it. On the other hand, performance can be an important consideration, and simpler code can be easier to read.

If you want to see whether your design makes sense, consider using `godoc` to have a look at the documentation for your package. You're writing good doc comments, right? If you aren't, please rethink this. Even if it's just your own code; tools like [sourcegraph](https://sourcegraph.com/) can display doc comments as tooltips for items on mouseover, as can many editors, and once you get used to seeing doc comments for things, working in code that didn't make doc comments is just painful. However, `godoc` only shows documentation for *exported* names. The documentation it provides shows you the interface you've chosen to present to the world.

Does the interface look complete to you? Great! But there's another question you should ask: Does it look *redundant* to you?

```go
// Param represents a named value.
type Param struct {
	Value int    // the parameter's value
	Name  string // the name of the parameter
}

// SetName sets the param's name.
func (p *Param) SetName(name string) {
	delete(paramMap, p.Name)
	p.Name = name
	paramMap[p.Name] = p
}

// GetName gets the param's name. (But also don't use a method name like this.)
func (p *Param) GetName() string {
	return p.Name
}
```

This interface has a flaw. There's two ways to set the parameter's name, and the user doesn't necessarily know *why*. Should we use `SetName`? Should we just write to the field directly? Given that we *can* just write to the field directly, why does the function also exist? `SetName`'s doc comment doesn't tell us about the bookkeeping it's doing to track named parameters. In this case, it would be better if the `Name` field hadn't been exported; we wouldn't have to know what bookkeeping the code is going to do to know how we should access the field.

And this is where the ability to decide which fields to export starts showing its value: You can do it selectively. If you want to let people directly access the param's value, but you need to do extra bookkeeping for changes to the param's name, you can control that. You don't have to move everything to accessor methods.

## Accessors Considered Sort of Annoying

I sort of hate accessors. They're often very closely tied to a particular way of thinking about "object-oriented" development, and they tend to be clunky. It works better to design methods in terms of *functionality*, rather than in terms of getting and setting values. If you really just want to get and set values, just export those fields. And if you need to do bookkeeping or something, consider whether that might imply a better name. For instance, the real point of the `SetName` function is to do the additional updating of a store of named parameters. Maybe it should be named `Rename`, and return an error if it can't (for instance, if the proposed new name is already in use). I didn't think of the error until I started thinking about calling the function `Rename`, because I have different intuitions for what kinds of thing a "rename" operation might have.

Unexported fields are fine, and you don't need to expose their functionality, or even their existence. If you expose them, you're still stuck with them being part of your API just as much as you would be if you'd exported them. Expose the functionality you want.

Accessors do make sense for one category of interfaces, and that's `interface`s. When you add methods to an interface type, it's quite likely that some of them will be accessor methods for some implementations. They're necessary, not because you wouldn't want to export fields, but because interfaces can't specify "fields", just methods. (This may seem weird until you realize that a type doesn't have to be a `struct` to have methods and satisfy an interface.)

## Unexported Methods

Don't forget that methods need not be exported. If you commonly need a piece of functionality, and it makes sense for it to be a method, you don't *have* to export it, unless it's needed to satisfy an interface somewhere using an exported method name. Package-internal methods are a perfectly good design choice; they still give you the clear indication that the functionality is in some way innately tied to the receiver, and they don't become part of your externally visible API.

The ability to control how much of your design is visible to users gives you a lot of flexibility. Use it well, and you'll find that users of your code don't need to know about refactoring.
