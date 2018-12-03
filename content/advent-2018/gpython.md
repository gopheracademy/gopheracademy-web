+++
author = ["Nick Craig-Wood"]
title = "gpython: a Python interpreter written in Go \"batteries not included\""
linktitle = "gpython: a Python interpreter written in Go"
date = 2018-12-12T00:00:00Z
series = ["Advent 2018"]
+++

[Gpython](https://github.com/go-python/gpython/) is a Python 3.4
interpreter written in Go. This is the story of how it came to be, how
it works and where it is going.

This includes a quick run through how an interpreted language like
Python/Gpython works with a dip into the Virtual Machine (VM), lexing
source, parsing it and compiling to byte code.

## Genesis

In 2013 I had a health problem which meant I needed to take 3 months
off work, so I needed a project to take my mind off things.  I came up
with the ridiculous idea of porting the Python interpreter to Go as a
project which would take at least 3 months and would keep me occupied.
It also had the nice property that it was completely self contained -
just me and 380,000 lines of C and 483,000 lines of Python so I could
pick it up and put it down according to how I was feeling.

So with that overambitious plan decided, I worked out how I was going
to attack the project:

1. Implement some Python objects, just enough for...
2. Implement a 100% compatible byte code interpreter
3. Implement the lexer
4. Implement the parser
5. Implement the compiler
6. ...
7. PROFIT!

I ended up completing step 1 & 2 in the 3 months I was off work and
the other steps a lot later ;-)

## 1. Python objects

Before we get to the Virtual Machine (VM) we need to define some
Python objects to act upon. Everything in Python is an object. Python
objects are implemented as C structures with a pointer to a type, so
this suggested that the Python object should be implemented as an
interface in Go.

```go
// A Python object
type Object interface {
    Type() *Type
}
```

Where `Type` is defined like this:

```go
type Type struct {
    ObjectType *Type  // Type of this object
    Name       string // For printing, in format "<module>.<name>"
    Doc        string // Documentation string
    Base       *Type
    Dict       StringDict
    //... more stuff here!
}
```

If you are familiar with Python you'll see some things you recognise
there, the name of the object, its docstring, a dictionary etc.  Note
that in Python types are also objects hence the `ObjectType`.

Python objects are reference counted and garbage collected when not in
use.  Gpython uses the go garbage collector rather than re-implement
this.

### Strings

Because Gpython objects are implemented by an interface, we can use
native go types to implement them.  Here is how the string type is
implemented.

```go
type String string

var StringType = ObjectType.NewType("str",
    `str(object='') -> str
str(bytes_or_buffer[, encoding[, errors]]) -> str
[snip docs]
`, StrNew, nil)

// Type of this object
func (s String) Type() *Type {
    return StringType
}
```

That makes interoperability between Gpython and Go really easy.  If
you want a Gpython string you just cast it with `py.String("gpython
string")` and likewise a type assertion gets a native string from a
Python object.

### Methods

Objects wouldn't be much use without methods.  Here Gpython takes a
hybrid approach to methods.

There are two ways of defining methods:

1. As specially named methods on objects
2. As items in the `Type`'s dictionary (`Dict`)

Here is how the `__len__` method is defined on strings:

```go
// len returns length of the string in Unicode characters
func (s String) len() int {
    return utf8.RuneCountInString(string(s))
}

// M__len__ implements the __len__ special method
func (s String) M__len__() (Object, error) {
    return Int(s.len()), nil
}

```

## 2. The VM

Now we've defined a few objects we are ready to start implementing the
byte code interpreter.

Unlike most of Python, the VM isn't well documented.  It is
implemented in
[ceval.c](https://github.com/python/cpython/blob/3.4/Python/ceval.c).
This contains a lot of highly optimised C code!

The Python byte code is 8 bit op-codes which implement a stack
machine.  All things on the stack are Python objects.  The Python
stack is part of the `Frame` object which you can see below.  There is
one `Frame` per function, so if you call a function, Python makes a
new `Frame`.

```go
// A Python Frame object
type Frame struct {
    Code            *Code      // code segment
    Builtins        StringDict // builtin symbol table
    Globals         StringDict // global symbol table
    Locals          StringDict // local symbol table
    Stack           []Object   // Object stack
    LocalVars       Tuple      // Fast access local vars
    // ... and more
```

A typical byte code operation might be `BINARY_ADD`.  This pops two
values off the stack, adds them together and pushes the result back on
the stack.

```go
// Implements TOS = TOS1 + TOS.
func do_BINARY_ADD(vm *Vm, arg int32) error {
    b := vm.POP()
    a := vm.TOP()
    return vm.setTopAndCheckErr(py.Add(a, b))
}
```

Opcodes can have a 32 bit integer parameter which is encoded with
variable length extension op codes. This is used for program control
for example in the `JUMP_ABSOLUTE` instruction which is for jumping to
different parts of the program.

```go
// Set bytecode counter to target.
func do_JUMP_ABSOLUTE(vm *Vm, target int32) error {
    vm.frame.Lasti = target
    return nil
}
```

### VM complete!

Skipping lightly over a few details in the VM now we've reached the
point we can run Python programs!

Sort of...

Well we have to get Python to compile the Python source code into byte
code for us and then we can run the byte code with Gpython so we
aren't self hosting yet... but we are getting there.

Next step is compiling and running our own code.  This is broken down
into:

- Lexing
- Parsing
- Compiling

## 3. Lexing python

Lexing is the process of taking a stream of input (UTF-8 in gpython's
case) and turning it into tokens.  The tokens are traditionally
written in UPPER_CASE and are numeric values.

[Gpython's lexer](https://github.com/go-python/gpython/blob/master/parser/lexer.go)
is a from scratch implementation of the [excellent
specification](https://docs.python.org/3/reference/lexical_analysis.html).

For example we turn the input Python code:

```py
while True:
    pass
else:
    return
```

Into a slice of `LexToken`:

```go
[]LexToken{
    {WHILE, nil, ast.Pos{1, 0}},
    {TRUE, nil, ast.Pos{1, 6}},
    {':', nil, ast.Pos{1, 10}},
    {NEWLINE, nil, ast.Pos{1, 11}},
    {INDENT, nil, ast.Pos{2, 0}},
    {PASS, nil, ast.Pos{2, 1}},
    {NEWLINE, nil, ast.Pos{2, 5}},
    {DEDENT, nil, ast.Pos{3, 0}},
    {ELSE, nil, ast.Pos{3, 0}},
    {':', nil, ast.Pos{3, 4}},
    {NEWLINE, nil, ast.Pos{3, 5}},
    {INDENT, nil, ast.Pos{4, 0}},
    {RETURN, nil, ast.Pos{4, 1}},
    {NEWLINE, nil, ast.Pos{4, 7}},
    {DEDENT, nil, ast.Pos{5, 0}},
    {NEWLINE, nil, ast.Pos{5, 0}},
}
```

Unusually for a lexer, the Python lexer encodes the white space at the
start of lines with the `INDENT` and `DEDENT` tokens.  You can see
also the lexer records the position of each symbol for error reporting
in the `ast.Pos` type.

## 4. Parsing Python

Once the input source has been tokenised it is ready to feed into a
parser.  The parser takes the stream of lexed tokens and outputs an
Abstract Syntax Tree.  The AST is a tree representation of the source
code suitable for compilation.

Gpython's parser is implemented with
[goyacc](https://godoc.org/golang.org/x/tools/cmd/goyacc).  This will
be immediately familiar to anyone who has used yacc or bison but it
probably will seem a bit strange if you haven't used a parser
generator before.

The parser is implemented in
[parser.y](https://github.com/go-python/gpython/blob/master/parser/grammar.y)
which is a mixture of goyacc directives and go code.  This is
processed by the goyacc tool into the go code (using `go generate` to
control the process).

Here is how the `for` statement is defined.  You can see the tokens
expected in the input `FOR`, `IN` and the literal `:`.  An `exprlist`
is a sequence of expressions _eg_ `a` or `a, b`.  A `testlist` is an
expression for the `for` statement to range over.  `suite` is the body
of the for loop and the `optional_else` (not shown) defines what to do
with the `else:` part of the `for` loop.  The `$N` refer to the input
line so `$2` is the `exprlist`.

```go
for_stmt:
    FOR exprlist IN testlist ':' suite optional_else
    {
        target := tupleOrExpr($<pos>$, $2, false)
        setCtx(yylex, target, ast.Store)
        $$ = &ast.For{StmtBase: ast.StmtBase{Pos: $<pos>$}, Target: target, Iter: $4, Body: $6, Orelse: $7}
    }
```

As an example this input Python code:

```py
for a, b in b:
    pass
else:
    break
```

Would produce this AST.  You can see the `for` the two parameters `a,
b` the iterable `b` and the body `pass` and the `else` clause with
`break` in.
    
```py
Module(body=[
    For(target=Tuple(elts=[
        Name(id='a', ctx=Store()),
        Name(id='b', ctx=Store())
    ], ctx=Store()),
    iter=Name(id='b', ctx=Load()),
    body=[
        Pass()
    ],
    orelse=[
        Break()
    ])
])
```

Python has an [AST defined as part of the
language](https://docs.python.org/3/library/ast.html) and gpython
follows that to build the tree of objects that represents an input
program.

## 5. Compile

Now we have the AST we can compile it!

Compiling is the process of transforming the AST into byte codes which
can be run directly by the VM.  In a C compiler the output would be
machine language, but here we output Python byte codes instead.

In gpython this is actually a two step process.  The compiler outputs
an internal assembler format which has labels and then that internal
assembler format is assembled into byte code.

[The compiler in
gpython](https://github.com/go-python/gpython/blob/master/compile/compile.go)
was mostly written from scratch, however I did get quite a bit of
assistance looking at the Python source code for the tricky bits.
Having used exactly the same AST representation as Python was helpful
here. The assembler was written entirely from scratch.

Given that the Python VM is a stack machine it is comparatively easy
to implement the compiler - we don't have to allocate registers or
keep track of memory or anything like that.

Here is a very cut down version of the expression compiler which shows
how the binary operators get compiled.  An `Expr` node represents an
arbitrarily complicated expression, _eg_ `1+2` or `f(x) + 7*y`.  It
typically contains other `Expr` nodes.  `Expr` nodes can be variables
or constants or other `Expr` nodes.

```go
// Compile an Expr node
func (c *compiler) Expr(expr ast.Expr) {
    c.SetLineno(expr)
    switch node := expr.(type) {
    // ...snip...
    case *ast.BinOp:
        // Left  Expr
        // Op    OperatorNumber
        // Right Expr
        c.Expr(node.Left)
        c.Expr(node.Right)
        var op vm.OpCode
        switch node.Op {
        case ast.Add:
            op = vm.BINARY_ADD
        case ast.Sub:
            op = vm.BINARY_SUBTRACT
        case ast.Mult:
            op = vm.BINARY_MULTIPLY
        case ast.Div:
            op = vm.BINARY_TRUE_DIVIDE
        // ...snip
        }
        c.Op(op) // compile the op code
    case *ast.Name:
        // Id  Identifier
        // Ctx ExprContext
        c.NameOp(string(node.Id), node.Ctx) // compile a load of the name
    // ...and lots more
    }
}
```

Let's see how we might compile the expression `a+b` which would be represented by this AST:

```go
Expression(
    body=BinOp(
        left=noName(id='a', ctx=Load()),
        op=Add(),
        right=Name(id='b', ctx=Load()),
    )
)
```

Running that through `Expr` would first recurse to evaluate
`node.Left` (`a`) which would compile `LOAD_GLOBAL 'a'` then it would
recurse to evaluate `node.Right` (`b`) which would compile
`LOAD_GLOBAL 'b'` finally it would compile the op code `BINARY_ADD` to
add the two items on the stack and replacing them with `a+b` to
produce something like this:

```
  1           0 LOAD_GLOBAL              0 (a)
              3 LOAD_GLOBAL              1 (b)
              6 BINARY_ADD
```

You can use [Python's dis module](https://docs.python.org/3/library/dis.html) to explore bytecodes further.

## Achievement unlocked?

Now gpython can Lex, Parse, Compile and Run Python code.  Job done?

Well no unfortunately.  What makes Python great is the fact that there
are a lot of well written modules included in the standard library.
Unfortunately for gpython a great number of these are written in C
rather than Python which means porting them is really hard work.

Hence the tag line "batteries not included" :-(

It was at that point that the gpython project ground to a halt for
several years.  It contained a working Python interpreter, but very
few modules and I just couldn't bring myself to release it like that.

However after chatting to various gophers I decided to ~~throw it over
the wall~~ release it anyway, warts and all, and I did so in 2018.

Sebastien Binet very kindly offered to have it as part of the
[go-python organisation](https://github.com/go-python) so it could
live with other Go and Python crossover projects.

## Performance

Performance was never a goal of gpython, but I often get asked how
fast is it?

Here is the ubiquitous pystone benchmark

```
$ gpython examples/pystone.py 
Pystone(1.1) time for 50000 passes = 1.6536743640899658
This machine benchmarks at 30235.698808522993 pystones/second
```

vs python3.4

```
$ python3.4 examples/pystone.py 
Pystone(1.1) time for 50000 passes = 0.312686
This machine benchmarks at 159905 pystones/second
```

So gpython is 5 times slower than Python 3.4 on the pystone benchmark.

However gpython is a lot faster than Python for long number
operations - I think because the go `big.Int` implementation is faster
than the Python one.

```
$ gpython examples/pi_chudnovsky_bs.py
chudnovsky_gmpy_bs: digits 100000 time 0.6917450428009033
```

```
$ python3.4 examples/pi_chudnovsky_bs.py
chudnovsky_gmpy_bs: digits 100000 time 3.047863006591797
```

Gpython is 4.4x faster than Python on this cherry picked
example ;-)

## Gpython in your browser

go1.11 came with a web assembly backend so I decided to port gpython
to the web.  It turned out to be quite straight forward, so here you
can [try gpython in your browser](https://gpython.org/)!

There is a [go/wasm](https://github.com/golang/go/wiki/WebAssembly)
implementation and a [gopherjs](https://github.com/gopherjs/gopherjs)
implementation each with their own bugs!

## Future

I'd like to fill out the modules for gpython.  I think gpython could
re-use some of the work done by the [grumpy
project](https://github.com/grumpyhome/grumpy) to increase its coverage
here.

I'd also like to explore adding go routines and channel support to
gpython but I don't plan to introduce the global interpreter lock
(GIL).

If anyone would like to help with any of that the gpython team is
always looking for contributors - [pull requests here
please](https://github.com/go-python/gpython) :-) You can chat with
the go-python community (of which gpython is part) at
[go-python@googlegroups.com](https://groups.google.com/forum/#!forum/go-python)
or on the [Gophers Slack](https://gophers.slack.com/) in the
`#go-python` channel.

Thank you for reading and I hope you enjoyed that meander through
gpython!  If you have any questions then please contact me at
[@njcw](https://twitter.com/njcw) or nick@craig-wood.com
