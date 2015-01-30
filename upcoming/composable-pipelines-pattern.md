+++
Author = ["Samuel Lampa"]
date = "2015-01-30T00:00:00-08:00"
series = [""]
title = "Pattern for pluggable pipeline components in Go"
+++

# Patterns for composable concurrent pipelines in Go

I came into Go from the python-world, which is so prevalent in my field, bioinformatics (I work mostly on the infrastructure side though, but do poke around with general applied bioinformatics stuff from time to time, out of sheer curiosity).

## Generator function in Python

In python I had just learned how to write composable lazy-evauated pipelines of string processing operations, using it's generator syntax, which by some simple testing showed to be both vastly less memory consuming, as well as a little faster, than their eager counterparts - and so I was hooked!

The Generator functionality in python basically means that you create a function that instead of returning a single value, such as a list of items, will return a generator object, which you can iterate over. What is special about a generator compared to other *iterables* such as lists though, is that the generator function will start evaluating itself and yield the objects one by one only when you start iterating over it.

So, say that we have a file, chr_y.fa containing a little bit of the familiar A, C, G, T DNA nucleotides from the human Y chromosome, in the ubiquotous [FASTA file format](http://en.wikipedia.org/wiki/FASTA_format):

**chr_y.fa:**
````fasta
>Homo_sapiens.GRCh37.67.dna_rm.chromosome.Y
GTGATTGTTGTTAGGTTCTTAGCTGCTTCTGAAAAATGGGGTGATAATCTTAGAAGGACT
TGCTTCATGGGATGTGGTCCATAAAACTTCCTCTGCCCCAGTTGTAGGGCAGAAGACAAT
TTCTGTTACTGTAGTTTGGCCTTTTTTTGCAGAGATTCAGACATCTGTTTACTGACCTTA
GTTAAATTGTGACACTATGCCTAAAGGAGCCTGCAAGCTTTTATTTTTGCTCACTATGAA
GTCATCATTCAATTGTAAAATTTCCTTTTTAAGTTTCAGGTTGACTTAATGTCTGTCAAA
GCACAGTCTTTGGCAATAACAAAACAAACATATGCTGAATGAAAATGTTTAAGAGATGGA
TGACTATTTACTACTAAAAGAAGAAAAATTGGAAGAGAATAAAATGAAAACATGCATCTC
CTAAACCATATGNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNN
NNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNN
NNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNN
NNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNTGCATGTTTTAG
TGATTCATACTAGGTCAGTATTATAAAACTATGCTTTGTCCTTGTAAGGGGAGGCTTAAA
````
*(The N:s mean that the nucleotides at those positions are not known, and the first line, starting with '>', is just a label, that should be skipped in our case, which will be seen in the following code examples)*

*(For the real world Human Y-Chromosome fasta file, that I have been using, use [this link](ftp://ftp.ensembl.org/pub/release-67/fasta/homo_sapiens/dna/Homo_sapiens.GRCh37.67.dna_rm.chromosome.Y.fa.gz) [68MB, gzipped])*

Then we could read the content of the file, line by line, and process it in sequential steps, using chained generators:

````python
# Create a lazy-evaluated file reader, just yielding the
# lines of the file one by one, as we later iterate on
# the generation function that it returns.
def fasta_reader(filename):
	with open(filename,'r') as infile:
		for line in infile:
			yield line

# A little base complementer, which converts each nucleotide
# in a DNA sequence into it's "complement" nucleotide (the
# one it will sit paired to, in the actual DNA double-strand
# helix molecule)
def base_complementer(line_generator):
	for line in line_generator:
		if line[0] != '>':
			translation_table = {
			  'A' : 'T',
			  'T' : 'A',
			  'C' : 'G',
			  'G' : 'C',
			  'N' : 'N',
			  '\n' : '\n'}
			for nuc in line:
				newline.append(translation_table[nuc])
			yield newline

def main():
	# Connect our super-minimal little "workflow" consisting
	# - a base complementer
	# - a file reader
	# - our print statements right here in the loop
	fa_reader = fasta_reader('chr_y.fa')
	for line in base_complementer(fa_reader):
		print line
````

So here, when we execute the main loop, in the main() function, that loop will, step by step, drive our little pipeline consisting of the three parts: a (fasta) reader, a base complementer generator funciton, and our implicit little printer at the end.

The lazy evaluation means that one item at a time will be drawn through the whole pipeline for each iteration in the main loop, without any temporary aggregates (lists or dicts) of lines building up between the components, which means the program will use more or less the minimal amount of memory possible.

## The Generator pattern in Go

Coming in to Go, I was highly intrigued by all the new much more powerful concurrency patterns made possible in this language, elaborated in intriguing blog posts such as [the one on "concurrency patterns"](http://blog.golang.org/pipelines) and ["advanced concurrency patterns"](http://blog.golang.org/advanced-go-concurrency-patterns).

Still, I was even more happy to find that the simple and straight-forward generator pattern I knew from python was available in Go too.

So the above python code would read something like this in Go, using the generator pattern (leaving out some imports and const definitions, for brevity):

````go

// ...<snip>...

func fileReaderGen(filename string) chan []byte {
	fileReadChan := make(chan []byte, BUFSIZE)
	go func() {
		file, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
		} else {
				scan := bufio.NewScanner(file)
				for scan.Scan() {
					// Write to the channel we will return, while copying
					// the buffer slice (which is re-used) ...
					fileReadChan <- append([]byte(nil), scan.Bytes()...)
		}
		close(fileReadChan)
		fmt.Println("Closed file reader channel")
		}
	}()
	return fileReadChan
}

// Translation table, used in the base complementer
// function below
var baseConv = [256]byte{
	'A': 'T',
	'T': 'A',
	'C': 'G',
	'G': 'C',
	'N': 'N',
	'\n' : '\n',
}

func baseComplementGen(inChan chan []byte) chan []byte {
	returnChan := make(chan []byte, BUFSIZE)
	go func() {
		for sequence := range inChan {
			if sequence[0] != '>' {
				for pos := range sequence {
					sequence[pos] = baseConv[sequence[pos]]
				}
			}
			returnChan <- append([]byte(nil), sequence...)
		}
		close(returnChan)
		fmt.Println("Closed base complement generator channel")
	}()
	return returnChan
}

func main() {
	// Read DNA file
	inFileName := "chr_y.fa"
	fileReadChan := fileReaderGen(inFileName)
	complChan := baseComplementGen(fileReadChan)

	// Read the last channel in the "pipeline", and print
	for line := range complChan {
		if len(line) > 0 {
			fmt.Println(string(line))
		}
	}
}
````

As you can see, we basically replicate the behaviour of python generator functions (although in python it is less obvious how they work): Instead of returning a single value (list etc), we return a channel, on which we can later iterate using the "range" construct (like we do in the `main()` method above), to retrieve the elements in a lazy-evaluation fashion. Of course, in Go, we have the obvious benefit that this will now also run fully concurrently, using all of our CPU cores, if we wish, since each "generator" starts it's own go-routine (see the `go func() { ...` bits in the code)

*For reference, The generator pattern is [included in Rob Pike's Go Concurrency pattern slides](https://talks.golang.org/2012/concurrency.slide#25), as well as [listed on this site](http://www.golangpatterns.info/concurrency/generators).*

## Composability for general pipelines?

The generator pattern is truly a neat one, when you have a simple "thread-like" pipeline; Just a few processing steps operating on a stream of items from a previous function or process.

But what if we want to build up custom topologies of connected processing components with multiple (streaming) inputs and outputs? The generator patterns clearly has some limitations when going in this direction (EDIT: Or does it?!)

Are we then bound to bury those processing network topologies deep down in spaghetti-code of functions with hard-coded dependencies?

### Enter Flow-based programming

Well, to start with I have to say that the answer to that problem is solved already, in a way that is not covered in this article: You should definitely have a serious look at the [GoFlow library](https://github.com/trustmaster/goflow), which solves this problem more or less to it's core, relying on the solid foundation of the principles of [Flow-based programming](www.jpaulmorrison.com/fbp/) (FBP), invented by John P Morrison at IBM back in the 60's.

Flow based programming solves the complexity problem of complex processing network topologies in a very thorough way by suggesting the use of named in- and out-ports, channels with bounded buffers (already proveded by Go), and a separate network definition. This last thing, separating the network definition from the actual processing components, is what seems to be so crucial to arrive at truly component-oriented, modular and composable pipelines.

I personally had a great deal of fun, playing around with GoFlow, and even have an embryonic little library of proof-of-concept bioinformatics components, written for use with the framework, available at [github](https://github.com/samuell/blow). An example program, using it, can be found [here](https://gist.github.com/samuell/6164115).

### Flow-based like concepts in pure Go?

Still, it is always nice to be able to rely solely on the standard-library when you can, which lead me to start experimenting with how far one can go with Flow-based programming-like ideas without using any 3rd party framework at all.

By playing around, I have at least found that Go definitely provides a lot more flexibility to how to define and wire together components, than the with e.g. the generator functions in python.

One of the patterns from my experiements that I tended to like a lot, is one where you encapsulate concurrent processes in a struct, and define the the inputs and outputs are struct fields, of type channel (of some subsequent type).

This lets us set up of channels and do all the wiring of the processes all totally outside of the components themselves, which might lead to slightly clearer code, since you seapate the business of the components, with the business of the over-all program: How components are connected together.

### Show me the code

So, what does this look like in practice?

Code examples of this little pattern can in fact be found in a little github repo I made for this idea, called [glow](https://github.com/samuell/glow), but let's have a look at the code examples here in the post as well, to keep it integrated:

#### An example component

First let's just have a look at how a component looks. Every component has one or more "in" and "outports", consisting of struct-fields of type channel (of some type that you choose. `[]byte` arrays in this case). Then it has a run method that initializes a go-routine, and reads on the inports, and writes on the outports, as it processes incoming "data packets" (each component could add it's desired type info, although here we just use []byte, since we are working with simple string processing in ASCII format):

````go
package glow

import (
	"bufio"
	"os"
)

func NewStdInReader(outChan chan []byte) *StdInReader {
	stdInReader := new(StdInReader)
	stdInReader.Out = outChan
	stdInReader.Init()
	return stdInReader
}

type StdInReader struct {
	Out chan []byte
}

func (self *StdInReader) Init() {
	go func() {
		scan := bufio.NewScanner(os.Stdin)
		for scan.Scan() {
			self.Out <- append([]byte(nil), scan.Bytes()...)
		}
		close(self.Out)
	}()
}
````

#### Connecting components - manual way

Then, to connect such processes together, we just create a bunch of channels, a bunch of processes, and then stitch them together, and run it! This is how the more "manual" way of doing that looks:

````go
package main

import (
	"fmt"
	"github.com/samuell/glow"
)

const (
	BUFSIZE = 128 // Set a buffer size to use for channels
)

func main() {
	// Create channels / connections
	chan1 := make(chan []byte, BUFSIZE)
	chan2 := make(chan []byte, BUFSIZE)
	chan3 := make(chan int, 0)

	// Create components, connecting the channels
	stdInReader := new(glow.StdInReader)
	stdInReader.Out = chan1
	stdInReader.Init()

	baseCompler := new(glow.BaseComplementer)
	baseCompler.In = chan1
	baseCompler.Out = chan2
	baseCompler.Init()

	printer := new(glow.Printer)
	printer.In = chan2
	printer.DrivingBelt = chan3
	printer.Init()

	// Loop over the last channel, to drive the execution
	cnt := 0
	for i := range chan3 {
		cnt += i
	}
	fmt.Println("Processed ", cnt, " lines.")
}
````

#### Connecting components - using convenience methods

The above way of connecting things might well be clear and rather "self-describing", but it admittedly takes a bit of keystrokes. We can save a lot of keystrokes, and make the code shorter, and maybe more readable by using some convenience functions:

````go
package main

import (
	"fmt"
	"github.com/samuell/glow"
)

const (
	BUFSIZE = 2048 // Set a buffer size to use for channels
)

func main() {
	// Create channels / connections
	chan1 := make(chan []byte, BUFSIZE)
	chan2 := make(chan []byte, BUFSIZE)
	chan3 := make(chan int, 0)

	// Create components, connecting the channels
	glow.NewStdInReader(chan1)             // Here, chan1 is an output channel
	glow.NewBaseComplementer(chan1, chan2) // chan1 is input, chan2 is output
	glow.NewPrinter(chan2, chan3)          // chan2 is input, chan3 is output

	// Loop over the last channel, to drive the execution
	cnt := 0
	for i := range chan3 {
		cnt += i
	}
	fmt.Println("Processed ", cnt, " lines.")
}
````


Finally, to compile and run the program above, do like this:
````bash
go build basecomplement.go
cat SomeFastaFile.fa | ./basecomplement > SomeFastaFile_Basecomplemented.fa
````

### A final recap

So, what did we do above? Basically we just encapsulated concurrently running functions in structs, and made the incoming and outgoing channels used by the function into struct fields.

So, what did we get by that? - Well, certainly in technical terms not much more than the normal Go generator pattern shown earlier, but my personal gut-feeling is that the struct-based approach gives the network-defining code a bit clearer and more declaratively looking.

Then, in fact, after writing the glow proof-of-concept library, I also realized some other ways of doing this, that would be various kinds of intermediate forms between the generator pattern, and the struct based pattern above. Those might be a topic for another blog post, but the two examples above should at least show two rather different ways of achieving the same thing.

Go is of course still a rather young language, and it remains to be shown what patterns and best practices will stand the test of time, but I hope that this post can at least spark a little discussion on patterns for truly component-oriented concurrent code in Go!
