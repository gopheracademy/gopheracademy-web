+++
author = ["Ben Johnson"]
date = "2014-12-02T00:00:00-06:00"
title = "Handwritten Parsers & Lexers in Go"
series = ["Advent 2014"]
+++

## Handwritten Parsers & Lexers in Go

In these days of web apps and REST APIs it seems that writing parsers is a dying
art. You may think parsers are a complex undertaking only reserved for
programming language designers but I'd like to dispel this idea. Over the past
few years I've written parsers for [JSON][megajson], [CSS3][css], and [database
query languages][influxql] and the more that I write parsers the more that I
love them.

[megajson]: https://github.com/benbjohnson/megajson
[css]: https://github.com/benbjohnson/css
[influxql]: https://github.com/influxdb/influxdb/tree/master/influxql


### The Basics

Let's start off with the basics: what is a lexer and what is a parser? When we
parse a language (or, technically, a "formal grammar") we do it in two phases.
First we break up series of characters into _tokens_. For a SQL-like language
these tokens may be "whitespace", "number", "SELECT", etc. This process is
called _lexing_ (or _tokenizing_ or _scanning_).

Take this simple SQL SELECT statement as an example:

```
SELECT * FROM mytable
```

When we tokenize this string we'd see it as:

```
`SELECT` • `WS` • `ASTERISK` • `WS` • `FROM` • `WS` • `STRING<"mytable">`
```

This process, called _lexical analysis_, is similar to how we break up words in
a sentence when we read. These tokens then get fed to a parser which performs
_semantic analysis_.

The parser's job is to make sense of these tokens and make sure they're in the
right order. This is similar to how we derive meaning from combining words in a
sentence. Our parser will construct an _abstract syntax tree (AST)_ from our
series of tokens and the AST is what our application will use.

In our SQL SELECT example, our AST may look like:

```go
type SelectStatement struct {
	Fields []string
	TableName string
}
```


### Parser Generators

Many people use parser generators to automatically write a parser and lexer for
them. There are many tools made to do this: [lex][lex], [yacc][yacc],
[ragel][ragel]. There's even a Go implementation of `yacc` built into the `go`
toolchain.

However, after using parser generators many times I've found them to be
problematic. First, they involve learning a new language to declare your
language format. Second, they're difficult to debug. For example, try reading
the [Ruby language's yacc file][ruby-yacc]. Eek!

After watching a talk by [Rob Pike on lexical scanning][pike] and reading the
implementation of the [`go`][go-pkg] standard library package, I realized how
much easier and simpler it is to hand write your parser and lexer. Let's walk
through the process with a simple example.

[lex]: http://en.wikipedia.org/wiki/Lex_%28software%29
[yacc]: http://en.wikipedia.org/wiki/Yacc
[ragel]: http://en.wikipedia.org/wiki/Ragel
[ruby-yacc]: https://github.com/mruby/mruby/blob/master/src/parse.y
[pike]: https://www.youtube.com/watch?v=HxaD_trXwRE
[go-pkg]: http://golang.org/pkg/go/


### Writing a Lexer in Go

#### Defining our tokens

Let's start by writing a simple parser and lexer for SQL SELECT statements.
First, we need to define what tokens we'll allow in our language. We'll only
allow a small subset of the SQL language:

```go
// Token represents a lexical token.
type Token int

const (
	// Special tokens
	ILLEGAL Token = iota
	EOF
	WS

	// Literals
	IDENT // fields, table_name

	// Misc characters
	ASTERISK // *
	COMMA    // ,

	// Keywords
	SELECT
	FROM
)
```

We'll use these tokens to represent series of characters. For example, `WS` will
represent one or more whitespace characters and `IDENT` will represent an
identifier such as a field name or a table name.

#### Defining character classes

It's useful to define functions that will let us check the type of character.
Here we'll define two functions: one to check if a character is whitespace and
one to check if the character is a letter.

```go
func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n'
}

func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}
```

It's also useful to define an "EOF" rune so that we can treat EOF like any other
character:

```go
var eof = rune(0)
```


#### Scanning our input

Next we'll want to define our `Scanner` type. This type will wrap our input
reader with a `bufio.Reader` so we can peek ahead at characters. We'll also
add helper functions for reading and unreading characters from our underlying
reader.

```go
// Scanner represents a lexical scanner.
type Scanner struct {
	r *bufio.Reader
}

// NewScanner returns a new instance of Scanner.
func NewScanner(r io.Reader) *Scanner {
	return &Scanner{r: bufio.NewReader(r)}
}

// read reads the next rune from the bufferred reader.
// Returns the rune(0) if an error occurs (or io.EOF is returned).
func (s *Scanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	return ch
}

// unread places the previously read rune back on the reader.
func (s *Scanner) unread() { _ = s.r.UnreadRune() }
```

The entry function into `Scanner` will be the `Scan()` method which return the
next token and the literal string that it represents:

```go
// Scan returns the next token and literal value.
func (s *Scanner) Scan() (tok Token, lit string) {
	// Read the next rune.
	ch := s.read()

	// If we see whitespace then consume all contiguous whitespace.
	// If we see a letter then consume as an ident or reserved word.
	if isWhitespace(ch) {
		s.unread()
		return s.scanWhitespace()
	} else if isLetter(ch) {
		s.unread()
		return s.scanIdent()
	}

	// Otherwise read the individual character.
	switch ch {
	case eof:
		return EOF, ""
	case '*':
		return ASTERISK, string(ch)
	case ',':
		return COMMA, string(ch)
	}

	return ILLEGAL, string(ch)
}
```

This entry function starts by reading the first character. If the character
is whitespace then it is consumed with all contiguous whitespace characters.
If it's a letter then it's treated as the start of an identifier or keyword.
Otherwise we'll check to see if it's one of our single character tokens.

#### Scanning contiguous characters

When we want to consume multiple characters in a row we can do this in a
simple loop. Here in `scanWhitespace()` we'll consume whitespace characters
until we hit a non-whitespace character:

```go
// scanWhitespace consumes the current rune and all contiguous whitespace.
func (s *Scanner) scanWhitespace() (tok Token, lit string) {
	// Create a buffer and read the current character into it.
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	// Read every subsequent whitespace character into the buffer.
	// Non-whitespace characters and EOF will cause the loop to exit.
	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isWhitespace(ch) {
			s.unread()
			break
		} else {
			buf.WriteRune(ch)
		}
	}

	return WS, buf.String()
}
```

The same logic can be applied to scanning our identifiers. Here in `scanIdent()`
we'll read all letters and underscores until we hit a different character:

```go
// scanIdent consumes the current rune and all contiguous ident runes.
func (s *Scanner) scanIdent() (tok Token, lit string) {
	// Create a buffer and read the current character into it.
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	// Read every subsequent ident character into the buffer.
	// Non-ident characters and EOF will cause the loop to exit.
	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isLetter(ch) && !isDigit(ch) && ch != '_' {
			s.unread()
			break
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}

	// If the string matches a keyword then return that keyword.
	switch strings.ToUpper(buf.String()) {
	case "SELECT":
		return SELECT, buf.String()
	case "FROM":
		return FROM, buf.String()
	}

	// Otherwise return as a regular identifier.
	return IDENT, buf.String()
}
```

This function also checks at the end if the literal string is a reserved word.
If so then a specialized token is returned.

### Writing a Parser in Go

#### Setting up the parser

Once we have our lexer ready, parsing a SQL statement becomes easier. First
let's define our `Parser`:

```go
// Parser represents a parser.
type Parser struct {
	s   *Scanner
	buf struct {
		tok Token  // last read token
		lit string // last read literal
		n   int    // buffer size (max=1)
	}
}

// NewParser returns a new instance of Parser.
func NewParser(r io.Reader) *Parser {
	return &Parser{s: NewScanner(r)}
}
```

Our parser simply wraps our scanner but also adds a buffer for the last read
token. We'll define helper functions for scanning and unscanning so we can use
this buffer:

```go
// scan returns the next token from the underlying scanner.
// If a token has been unscanned then read that instead.
func (p *Parser) scan() (tok Token, lit string) {
	// If we have a token on the buffer, then return it.
	if p.buf.n != 0 {
		p.buf.n = 0
		return p.buf.tok, p.buf.lit
	}

	// Otherwise read the next token from the scanner.
	tok, lit = p.s.Scan()

	// Save it to the buffer in case we unscan later.
	p.buf.tok, p.buf.lit = tok, lit

	return
}

// unscan pushes the previously read token back onto the buffer.
func (p *Parser) unscan() { p.buf.n = 1 }
```

Our parser also doesn't care about whitespace at this point so we'll define a
helper function to find the next non-whitespace token:

```go
// scanIgnoreWhitespace scans the next non-whitespace token.
func (p *Parser) scanIgnoreWhitespace() (tok Token, lit string) {
	tok, lit = p.scan()
	if tok == WS {
		tok, lit = p.scan()
	}
	return
}
```

#### Parsing the input

Our parser's entry function will be the `Parse()` method. This function will
parse the next SELECT statement from the reader. If we had multiple statements
in our reader then we could call this function repeatedly.

```go
func (p *Parser) Parse() (*SelectStatement, error)
```

Let's break this function down into small parts. First we'll define the AST
structure we want to return from our function:

```go
stmt := &SelectStatement{}
```

Then we'll make sure there's a `SELECT` token. If we don't see the token we
expect then we'll return an error to report the string we found instead.

```go
if tok, lit := p.scanIgnoreWhitespace(); tok != SELECT {
	return nil, fmt.Errorf("found %q, expected SELECT", lit)
}
```

Next we want to parse a comma-delimited list of fields. In our parser we're
just considering identifiers and an asterisk as possible fields:

```go
for {
	// Read a field.
	tok, lit := p.scanIgnoreWhitespace()
	if tok != IDENT && tok != ASTERISK {
		return nil, fmt.Errorf("found %q, expected field", lit)
	}
	stmt.Fields = append(stmt.Fields, lit)

	// If the next token is not a comma then break the loop.
	if tok, _ := p.scanIgnoreWhitespace(); tok != COMMA {
		p.unscan()
		break
	}
}
```

After our field list we want to see a `FROM` keyword:

```go
// Next we should see the "FROM" keyword.
if tok, lit := p.scanIgnoreWhitespace(); tok != FROM {
	return nil, fmt.Errorf("found %q, expected FROM", lit)
}
```

Then we want to see the name of the table we're selecting from. This should be
an identifier token:

```go
tok, lit := p.scanIgnoreWhitespace()
if tok != IDENT {
	return nil, fmt.Errorf("found %q, expected table name", lit)
}
stmt.TableName = lit
```

If we've gotten this far then we've successfully parsed a simple SQL SELECT
statement so we can return our AST structure:

```
return stmt, nil
```

Congrats! You've just built a working parser!


### Diving in deeper

You can find the full source of this example (with tests) at:

> https://github.com/benbjohnson/sql-parser

This parser example was heavily influenced by the InfluxQL parser. If you're
interested in diving deeper and understanding multiple statement parsing, 
expression parsing, or operator precedence then I encourage you to check out
the repository:

> https://github.com/influxdb/influxdb/tree/master/influxql

If you have any questions or just love chatting about parsers, please find 
me on Twitter at [@benbjohnson](https://twitter.com/benbjohnson).


