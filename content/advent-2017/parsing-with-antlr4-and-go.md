+++
author = ["Andrew Brampton"]
date = "2017-12-10"
linktitle = "Parsing with Antlr4 and Go"
series = ["Advent 2017"]
title = "Parsing with Antlr4 and Go"
+++

## What is Antlr4?

[ANTLR](http://www.antlr.org) (ANother Tool for Language Recognition),
is an Adaptive [LL(\*)](https://en.wikipedia.org/wiki/LL_parser)
[parser generator](https://en.wikipedia.org/wiki/Parser_generator). In
layman's terms, Antlr, creates parsers in a number of languages (Go,
Java, C, C#, Javascript), that can process text or binary input. The
generated parser provides a callback interface to parse the input in an
event-driven manner, which can be used as-is, or used to build parse
trees (a data structure representing the input).

Antlr is used by a number of popular projects, e.g Hive and Pig use it
to parse Hadoop queries, Oracle and NetBeans uses it for their IDEs, and
Twitter even uses it to understand search queries. Support was recently
added so that Antlr4 can be used to generate parsers in pure Go. This
article will explain some of the benefits of Antlr, and walk us through
a simple example.

## Why use it?

It is possible to [hand write a
parser](https://blog.gopheracademy.com/advent-2014/parsers-lexers/), but
this process can be complex, error prone, and hard to change. Instead
there are many [parser generators](https://en.wikipedia.org/wiki/Compari
son_of_parser_generators) that take a grammar expressed in an domain-
specific way, and generates code to parse that language. Popular parser
generates include [bison](https://www.gnu.org/software/bison/) and
[yacc](http://dinosaur.compilertools.net/yacc/). In fact, there is a
version of yacc, goyacc, which is written in Go and was part of the main
go repo until it was moved to
[golang.org/x/tools](https://godoc.org/golang.org/x/tools/cmd/goyacc)
last year.

### So why use Antlr over these?

  * Antlr has a [suite of tools](http://www.antlr.org/tools.html), and
    [GUIs](http://tunnelvisionlabs.com/products/demo/antlrworks), that
    makes writing and debugging grammars easy.
  
  * It uses a simple [EBNF](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form)
    syntax to define the grammar, instead of a bespoke configuration
    language.

  * Antlr is an [adaptive](http://www.antlr.org/papers/allstar-techreport.pdf)
    [LL(\*) parser](https://en.wikipedia.org/wiki/LL_parser) whereas
    most other parser generators (e.g Bison and Yacc) are
    [LALR](https://en.wikipedia.org/wiki/LALR_parser). The difference
    between LL(\*) and LALR is out of scope for this article, but
    simply LALR works bottom-up, and LL(\*) works top-down. This
    has a bearing on how the grammar is written, making some languages
    easier or harder to express.

  * The generated code for a LL(\*) parser is more understandable than a
    LALR parser. This is because LALR parsers are commonly table driven,
    whereas LL(\*) parsers encode the logic in its control flow, making
    it more comprehensible.

  * Finally Antlr is agnostic to the target language. A single grammar
    can be used to generate parsers in Java, Go, C, etc. Unlike
    Bison/Yacc which typically embeds target language code into the
    grammar, making it harder to port.

## Installing Antlr4

Antlr4 is a Java 1.7 application, that generates the Go code needed to
parse your language. During development Java is needed, but once the
parser is built only Go and the [Antlr
library](https://godoc.org/github.com/antlr/antlr4/runtime/Go/antlr) is
required. The Antlr site has
[documentation](https://github.com/antlr/antlr4/blob/master/doc/getting-
started.md) on how to install this on multiple platforms, but in brief,
you can do the following:

```shell
$ wget http://www.antlr.org/download/antlr-4.7-complete.jar
$ alias antlr='java -jar $PWD/antlr-4.7-complete.jar'
```

The `antlr` command is now available in your shell. If you prefer, the
.jar file can be placed into a `~/bin` directory, and the alias can be
stored in your `~/.bash_profile`.

## Classic calculator example

Let's start with the “hello world” for parsers, the calculator example.
We want to build a parser that handles simple mathematical expressions
such as `1 + 2 * 3`. The focus of this article is on how to use Go with
Antlr4, so the syntax of the Antlr language won’t be explained in
detail, but the Antlr site has [compressive documentation](https://githu
b.com/antlr/antlr4/blob/master/doc/grammars.md).

As we go along, the [source is available to all
examples](https://github.com/bramp/goadvent-antlr).

```
// Calc.g4
grammar Calc;

// Tokens
MUL: '*';
DIV: '/';
ADD: '+';
SUB: '-';
NUMBER: [0-9]+;
WHITESPACE: [ \r\n\t]+ -> skip;

// Rules
start : expression EOF;

expression
   : expression op=('*'|'/') expression # MulDiv
   | expression op=('+'|'-') expression # AddSub
   | NUMBER                             # Number
   ;
```

The above is a simple grammar split into two sections, *tokens*, and
*rules*. The tokens are terminal symbols in the grammar, that is, they
are made up of nothing but literal characters. Whereas rules are non-
terminal states made up of tokens and/or other rules.

By convention this grammar must be saved with a filename that matches
the name of the grammar, in this case “Calc.g4” . To process this file,
and generate the Go parser, we run the antlr command like so:

```shell
$ antlr -Dlanguage=Go -o parser Calc.g4 
```

This will generate a set of Go files in the “parser” package and
subdirectory. It is possible to place the generated code in a different
package by using the `-package <name>` argument. This is useful if your
project has multiple parsers, or you just want a more descriptive
package name for the parser. The generated files will look like the
following:

```shell
$ tree
├── Calc.g4
└── parser
    ├── calc_lexer.go
    ├── calc_parser.go
    ├── calc_base_listener.go
    └── calc_listener.go
```

The generated files consist of three main components, the Lexer, Parser,
and Listener.

The Lexer takes arbitrary input and returns a stream of tokens. For
input such as `1 + 2 * 3`, the Lexer would return the following tokens:
`NUMBER (1), ADD (+), NUMBER (2), MUL (*), NUMBER (3), EOF`.

The Parser uses the Lexer’s output and applies the Grammar’s rules.
Building higher level constructs, such as expressions that can be used
to calculate the result.

The Listener then allows us to make use of the the parsed input. As
mentioned earlier, yacc requires language specific code to be embedded
with the grammar. However, Antlr separates this concern, allowing the
grammar to be agnostic to the target programming language. It does this
through use of listeners, which effectively allows hooks to be placed
before and after every rule is encountered in the parsed input.

## Using the Lexer

Let's move onto an example of using this generated code, starting with
the Lexer.

```go
// example1.go
package main

import (
	"fmt"
	"github.com/antlr/antlr4/runtime/Go/antlr"

	"./parser"
)

func main() {
	// Setup the input
	is := antlr.NewInputStream("1 + 2 * 3")

	// Create the Lexer
	lexer := parser.NewCalcLexer(is)

	// Read all tokens
	for {
		t := lexer.NextToken()
		if t.GetTokenType() == antlr.TokenEOF {
			break
		}
		fmt.Printf("%s (%q)\n",
			lexer.SymbolicNames[t.GetTokenType()], t.GetText())
	}
}
```

To begin with, the generated parser is imported from the local
subdirectory `import "./parser"`. Next the Lexer is created with some
input:

```go
	// Setup the input
	is := antlr.NewInputStream("1 + 2 * 3")

	// Create the Lexer
	lexer := parser.NewCalcLexer(is)
```

In this example the input is a simple string, `"1 + 2 * 3"` but there
are other [`antlr.InputStream`](https://godoc.org/github.com/antlr/antlr
4/runtime/Go/antlr#InputStream)s, for example, the [`antlr.FileStream`](
https://godoc.org/github.com/antlr/antlr4/runtime/Go/antlr#FileStream)
type can read directly from a file. The `InputStream` is then passed to
a newly created Lexer. Note the name of the Lexer is `CalcLexer` which
matches the grammar’s name defined in the Calc.g4.

The lexer is then used to consume all the tokens from the input,
printing them one by one. This wouldn’t normally be necessary but we do
this for demonstrative purposes.

```go
 	for {
		t := lexer.NextToken()
		if t.GetTokenType() == antlr.TokenEOF {
			break
		}
		fmt.Printf("%s (%q)\n",
			lexer.SymbolicNames[t.GetTokenType()], t.GetText())
	}
```

Each token has two main components, the TokenType, and the Text. The
TokenType is a simple integer representing the type of token, while the
Text is literally the text that made up this token. All the TokenTypes
are defined at the end of calc_lexer.go, with their string names stored
in the SymbolicNames slice:

```go
// calc_lexer.go
const (
	CalcLexerMUL        = 1
	CalcLexerDIV        = 2
	CalcLexerADD        = 3
	CalcLexerSUB        = 4
	CalcLexerNUMBER     = 5
	CalcLexerWHITESPACE = 6
)
```

You may also note, that the Whitespace token is not printed, even though
the input clearly had whitespace. This is because the grammar was
designed to skip (i.e. discard) the whitespace `WHITESPACE: [ \r\n\t]+
-> skip;`.

## Using the Parser

The Lexer on its own is not very useful, so the example can be modified
to also use the Parser and Listener:

```go
// example2.go
package main

import (
	"./parser"
	"github.com/antlr/antlr4/runtime/Go/antlr"
)

type calcListener struct {
	*parser.BaseCalcListener
}

func main() {
	// Setup the input
	is := antlr.NewInputStream("1 + 2 * 3")

	// Create the Lexer
	lexer := parser.NewCalcLexer(is)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	// Create the Parser
	p := parser.NewCalcParser(stream)

	// Finally parse the expression
	antlr.ParseTreeWalkerDefault.Walk(&calcListener{}, p.Start())
}
```

This is very similar to before, but instead of manually iterating over
the tokens, the lexer is used to create a [`CommonTokenStream`](https://
godoc.org/github.com/antlr/antlr4/runtime/Go/antlr#CommonTokenStream),
which in turn is used to create a new `CalcParser`. This `CalcParser` is
then “walked”, which is antlr’s event-driven API for receiving the
results of parsing the rules.

Note, the [`Walk`](https://godoc.org/github.com/antlr/antlr4/runtime/Go/
antlr#ParseTreeWalker.Walk) function does not return anything. Some may
have expected a parsed form of the expression to be returned, such as
some kind of [AST](https://en.wikipedia.org/wiki/Abstract_syntax_tree)
(abstract syntax tree), but instead the Listener receives event as the
parsing occurs. This is similar in concept to
[SAX](https://en.wikipedia.org/wiki/Simple_API_for_XML) style parsers
for XML. Event-based parsing can sometimes be harder to use, but it has
many advantages. For example, the parser can be very memory efficient as
previously parsed rules can be discarded once they are no longer needed.
The parser can also be aborted early if the programmer wishes to.

But so far, this example doesn’t do anything beyond ensuring the input
can be parsed without error. To add logic, we must extend the
`calcListener` type. The `calcListener` has an embedded
`BaseCalcListener`, which is a helper type, that provides empty methods
for all those defined in in the `CalcListener` interface. That interface
looks like:

```go
// parser/calc_listener.go
// CalcListener is a complete listener for a parse tree produced by CalcParser.
type CalcListener interface {
	antlr.ParseTreeListener

	// EnterStart is called when entering the start production.
	EnterStart(c *StartContext)

	// EnterNumber is called when entering the Number production.
	EnterNumber(c *NumberContext)

	// EnterMulDiv is called when entering the MulDiv production.
	EnterMulDiv(c *MulDivContext)

	// EnterAddSub is called when entering the AddSub production.
	EnterAddSub(c *AddSubContext)

	// ExitStart is called when exiting the start production.
	ExitStart(c *StartContext)

	// ExitNumber is called when exiting the Number production.
	ExitNumber(c *NumberContext)

	// ExitMulDiv is called when exiting the MulDiv production.
	ExitMulDiv(c *MulDivContext)

	// ExitAddSub is called when exiting the AddSub production.
	ExitAddSub(c *AddSubContext)
}
```

There is an Enter and Exit function for each rule found in the grammar.
As the input is walked, the Parser calls the appropriate function on the
listener, to indicate when the rule starts and finishes being evaluated.
## Adding the logic

A simple calculator can be constructed from this event driven parser by
using a stack of values. Every time a number is found, it is added to a
stack. Everytime an expression (add/multiple/etc) is found, the last two
numbers on the stack are popped, and the appropriate operation is
carried out. The result is then placed back on the stack.

Take the expression `1 + 2 * 3`,  the result could be either `(1 + 2) *
3 = 9`, or `1 + (2 * 3) = 7`. Those that recall the [order of
operations](https://en.wikipedia.org/wiki/Order_of_operations), will
know that multiplication should always be carried out before addition,
thus the correct result is 7. However, without the parentheses there
could be some ambiguity on how this should be parsed. Luckily the
ambiguity is resolved by the grammar. The precedence of multiplication
over addition was subtly implied within Calc.g4, by placing the `MulDiv`
expressed before the `AddSub` expression.

<div style="text-align:center;">
	<img src="/postimages/advent-2017/parse-tree.svg">
</div>

The code for a listener that implements this stack of value
implementation is relatively simple:

```go
type calcListener struct {
	*parser.BaseCalcListener

	stack []int
}

func (l *calcListener) push(i int) {
	l.stack = append(l.stack, i)
}

func (l *calcListener) pop() int {
	if len(l.stack) < 1 {
		panic("stack is empty unable to pop")
	}

	// Get the last value from the stack.
	result := l.stack[len(l.stack)-1]

	// Remove the last element from the stack.
	l.stack = l.stack[:len(l.stack)-1]

	return result
}

func (l *calcListener) ExitMulDiv(c *parser.MulDivContext) {
	right, left := l.pop(), l.pop()

	switch c.GetOp().GetTokenType() {
	case parser.CalcParserMUL:
		l.push(left * right)
	case parser.CalcParserDIV:
		l.push(left / right)
	default:
		panic(fmt.Sprintf("unexpected op: %s", c.GetOp().GetText()))
	}
}

func (l *calcListener) ExitAddSub(c *parser.AddSubContext) {
	right, left := l.pop(), l.pop()

	switch c.GetOp().GetTokenType() {
	case parser.CalcParserADD:
		l.push(left + right)
	case parser.CalcParserSUB:
		l.push(left - right)
	default:
		panic(fmt.Sprintf("unexpected op: %s", c.GetOp().GetText()))
	}
}

func (l *calcListener) ExitNumber(c *parser.NumberContext) {
	i, err := strconv.Atoi(c.GetText())
	if err != nil {
		panic(err.Error())
	}

	l.push(i)
}

```

Finally this listener would be used like so:

```go
// calc takes a string expression and returns the evaluated result.
func calc(input string) int {
	// Setup the input
	is := antlr.NewInputStream(input)

	// Create the Lexer
	lexer := parser.NewCalcLexer(is)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	// Create the Parser
	p := parser.NewCalcParser(stream)

	// Finally parse the expression (by walking the tree)
	var listener calcListener
	antlr.ParseTreeWalkerDefault.Walk(&listener, p.Start())

	return listener.pop()
}
```

Following the algorithm, the parsing of `1 + 2 * 3` would work like so.

1. The numbers 2 and 3 would be visited first (and placed on the stack),
2. Then the MulDiv expression would be visited, taking the values 2 and
   3, multiplying them, and placing the result, 6, back on the stack.
3. Then the number 1 would visited and pushed onto the stack.
4. Finally AddSub would be visited, popping the 1 and the 6 from the
   stack, placing the result 7 back.

The order the rules are visited is completely driven by the Parser, and
thus the grammar.

## More grammars

Learning how to write a grammar may be daunting, but there are many
resources for help. The author of Antlr, [Terence
Parr](http://parrt.cs.usfca.edu/), has [published a
book](https://pragprog.com/book/tpantlr2/the-definitive-antlr-4-reference),
with some of the content freely available on [antlr.org](http://antlr.org).

If you don’t want to write your own grammar, there are many [pre-written
grammars available](https://github.com/antlr/grammars-v4). Including
grammars for CSS, HTML, SQL, etc, as well many popular programming
languages. To make it easier, I have [generated
parsers](https://github.com/bramp/antlr4-grammars) for all those
available grammars, making them as easy to use just by importing.

A quick example of using one of the pre-generated grammars:

```go
import (
	"bramp.net/antlr4/json" // The parser

	"github.com/antlr/antlr4/runtime/Go/antlr"
)

type exampleListener struct {
	// https://godoc.org/bramp.net/antlr4/json#BaseJSONListener
	*json.BaseJSONListener
}

func main() {
	// Setup the input
	is := antlr.NewInputStream(`
		{
			"example": "json",
			"with": ["an", "array"]
		}`)


	// Create the JSON Lexer
	lexer := json.NewJSONLexer(is)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	// Create the JSON Parser
	p := json.NewJSONParser(stream)

	// Finally walk the tree
	antlr.ParseTreeWalkerDefault.Walk(&exampleListener{}, p.Json())
}
```

## Conclusion

Hopefully this article has given you a taste of how to use Go and Antlr.
The examples for this article are [found here](https://github.com/bramp/goadvent-antlr),
and the [godoc for the antlr library is here](https://godoc.org/github.com/antlr/antlr4/runtime/Go/antlr)
which explains the various InputStream, Lexer, Parser, etc interfaces.

If you have any questions or comments, please reach out to me at
[@TheBramp](https://www.twitter.com/TheBramp) or visit my website and
blog, [bramp.net](https://bramp.net) for more articles.
