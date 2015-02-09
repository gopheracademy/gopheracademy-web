+++
Author = ["Samuel Lampa"]
date = "2015-01-30T00:00:00-08:00"
series = [""]
title = "Patterns for composable concurrent pipelines in Go"
+++

# Patterns for composable concurrent pipelines in Go

## Generator function in Python

The author of this post came into Go from the python-world, which is quite prevalent the field of bioinformatics. In python one can easily write composable lazy-evauated pipelines of string processing operations using the generator syntax built into the language, which has been shown to both use less memory and to be a bit faster than their eager counterparts.

The Generator functionality in python basically means that you create a function that, rather than returning a single data structure once (say for example a list of items), rather will return a generator object which can later be iterated over by repeadetly calling its next method, or by the shorthand `for item in ...` syntax.

What is special about a generator compared to other *iterables* in python such as lists (which also support the `for item in ...` syntax), is that the generator function will start evaluating itself and yielding its output objects one by one only after iteration has started. Thus, minimal intermediate data is created and stored in memory.

To give an example, say that we have a file, chr_y.fa containing a little bit of the familiar A, C, G, T DNA nucleotides from the human Y chromosome, in the ubiquotous [FASTA file format](http://en.wikipedia.org/wiki/FASTA_format):

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
*(The N:s mean that the nucleotides at those positions are not known, and the first line, starting with '>', is just a label, that should be skipped in our case)*

*(For the real world Human Y-Chromosome fasta file, see [this link](http://ftp.ensembl.org/pub/release-67/fasta/homo_sapiens/dna/Homo_sapiens.GRCh37.67.dna_rm.chromosome.Y.fa.gz) [68MB, gzipped])*

Then we can read the content of the file line by line and process it in sequential steps, using chained generators:

````python
# Create a lazy-evaluated file reader, just yielding the
# lines of the file one by one, as we later iterate on
# the generation function that it returns.
def fasta_reader(filename):
	with open(filename,'r') as infile:
		for line in infile:
			yield line

# A base complementer, which converts each nucleotide in a
# DNA sequence into its "complement" nucleotide (the one it
# will sit paired to in the double-strand DNA helix molecule)
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

Have a careful look at this line:

````python
for line in base_complementer(fa_reader):
	...
````

We here see how, rather than sending data into `base_complementer`, we instead send a generator object (`fa_reader`), that `base_complementer` will iterate over, as soon as we start iterating over the the latter.

So when executing the main loop in `main()``, it will step by step drive our little pipeline consisting of three parts; a (FASTA) reader, a base complementer generator function and a print-statement at the end.

Again, the lazy evaluation means that one item at a time will be drawn through the whole pipeline for each iteration in the main loop, without any temporary aggregates (lists or dicts) of lines building up between the components, which means the program needs to use very little memory during the process.

## The Generator pattern in Go

Coming into Go, the author was highly intrigued by all the powerful concurrency patterns made possible in this language, elaborated in blog posts such as [the one on "concurrency patterns"](http://blog.golang.org/pipelines) and ["advanced concurrency patterns"](http://blog.golang.org/advanced-go-concurrency-patterns).

Even more interestingly, the simple and straight-forward generator pattern in python was easy to implement in Go too, as shown in [Rob Pike's Go Concurrency pattern slides](https://talks.golang.org/2012/concurrency.slide#25), and also [listed on this site](http://www.golangpatterns.info/concurrency/generators).

The above python code would read something like this in Go, using the generator pattern (leaving out some imports and const definitions, for brevity):

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

As you can see, we basically replicate the behaviour of python generator functions (although in python it is less obvious how they work).

Just like in python, instead of returning a single value (such as a list), we return something that can be iterated over in a lazy-evaluated way.

Whereas in python this was a generator object, in Go we instead return a channel, which can similarly be iterated over using the `for item := range ...` construct to retrieve the elements lazily (like in the `main()` method above), similar to the `for item in ...` syntax in python.

But of course in Go we have the obvious benefit that this chain of "generators" will also run fully concurrently using possibly all of our CPU cores since each "generator" starts its own go-routine (see the `go func() { ...` bits in the code).

## Composability for general pipelines?

The generator pattern is neat when we have a simple thread-like pipelines with processing steps operating one after each other, similar to a chain of piped programs in unix and linux.

But what if we want to build up more complex topologies of connected processing components, with multiple (streaming) inputs and outputs in each component (something more similar to a directed acyclic graph)?

It seems that code written with the generator patterns rather quickly can get a little hard to follow when going into this direction. Most importantly because it will not be as visible what kind of data are returned (or lazily streamed) on those channels, since the returned channels don't have any name exposed from the component to the outer world.

This begs the quesiton whether there is any pattern that suits this job better than the generator pattern?

### Enter Flow-based programming

To start with, the answer to that problem is solved already in a way that is not covered in this article: You should all definitely have a serious look at the [GoFlow library](https://github.com/trustmaster/goflow), which solves this problem more or less at its core, relying on the solid foundation of the principles of [Flow-based programming](www.jpaulmorrison.com/fbp/) (FBP), invented by John P Morrison at IBM back in the 60's.

Flow based programming solves the complexity problem of complex processing network topologies in a very thorough way by suggesting the use of named in- and out-ports, channels with bounded buffers (already proveded by Go), and network definition separated from the implementation of the processes. This last thing, separating the network definition from the actual processing components, is what seems to be crucial to arrive at truly component-oriented, modular and composable pipelines.

The author of this post had a great deal of fun playing around with GoFlow, and even has published an embryonic library of proof-of-concept bioinformatics components written for use with the framework, available at [github](https://github.com/samuell/blow). An example program using it can be found [here](https://gist.github.com/samuell/6164115).

### Flow-based like concepts in pure Go?

Still, for some problems it is nice to be able to rely solely on the standard-library, which lead the author to start experimenting with how far one can go with Flow-based programming inspired ideas without using any 3rd party framework at all - that is, finding out the most flow-based programming-like pattern one can implement in pure Go.

By playing around, the first thing that became clear is that Go provides a lot more flexibility to how to define and wire together components than the with e.g. the generator functions in python.

One of the patterns arising from this experimentation that the author tends to like a lot is one where concurrent processes are encapsulated into structs, and inputs and outputs defined are struct fields of type channel (of some subsequent type).

This pattern lets us do all the wiring of the processes using the struct fields, which, by having clear names coded into the processes, and exposed to the network-defining code, can in the authors opinion make the network definition code clearer and more intuitive.

This pattern is the subject for this post, and is presented in more detail below.


### Show me the code

So, what does this pattern look like in practice?

Code examples of this little pattern can be found in a github repo that the author made for this idea, called [glow](https://github.com/samuell/glow), but let's have a look at the code examples here in the post as well:

#### An example component

First let's just have a look at how a component looks, when using this pattern. Every component has one or more "in-" and "out-ports", consisting of struct fields of type channel (of some subsequent type - `[]byte` arrays in this case).

Then it has an `Init()` method that initializes a go-routine so that it is ready to start reading on the inport(s) and write on the outport(s) as soon as the first in-data arrive:

````go
package glow

import (
	"bufio"
	"os"
)

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

Here, since we did have just one output, (and no input), we have simply named the outport to "Out", but we could of course give it a more descriptive name if we wish.

Notice also inside the go-routine (the `go func() {...}` bit) above, how even the internal component code becomes a bit clearer by using fields of the struct, so that we know immediately where to look up the fields definition!

#### Connecting components - manual way

To connect such processes together - here in the more "manual" way - we create a bunch of channels, a bunch of processes and finally stitch them together and run it:

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

The above way of connecting things might well be clear and rather "self-describing", but it perhaps looks a bit ugly to manually create all those channels, find out some name for them (which easily becomes rather synthetic) before plugging them into an outport and inport.

Why not create a convenience function for each outport that returns a ready-make channel of the right type?

So, for our "Out" struct field / outport, let's create an accompanying `OutChan()` method, that does what we described above. The updated component would look like so:

````go
package glow

import (
	"bufio"
	"os"
	)

type StdInReader struct {
	Out chan []byte
}

func (self *StdInReader) OutChan() chan []byte {
	self.Out = make(chan []byte, 16)
	return self.Out
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

Now we can just plug the returned channel from this `OutChan()` function into an in-port of another component by just assigning the output of it into the "inport" struct field (See the two lines looking like `<StructFoo>.In = <StructBar>.OutChan()` below!).

````go
package main

import (
	"fmt"
	"github.com/samuell/glow"
	)

func main() {
	// Create channels / connections
	fileReader := new(glow.FileReader)
	baseComplementer := new(glow.BaseComplementer)
	printer := new(glow.Printer)

	// Connect components (THIS IS WHERE THE NETWORK IS DEFINED!)
	baseComplementer.In = fileReader.OutChan()
	printer.In = baseComplementer.OutChan()

	// Initialize / set up go-routines
	fileReader.Init()
	baseComplementer.Init()
	printer.Init()

	// The InFilePath channel has to be created manually
	fileReader.InFilePath = make(chan string)
	fileReader.InFilePath <- "test.fa"

	// Loop over the last channel, to drive the execution
	cnt := 0
	for i := range printer.DrivingBeltChan() {
		cnt += i
	}
	fmt.Println("Processed ", cnt, " lines.")
}
````

Isn't this rather nice? Look again at those lines that did the actual wiring of the network:

````go
// Connect components (THIS IS WHERE THE NETWORK IS DEFINED!)
baseComplementer.In = fileReader.OutChan()
printer.In = baseComplementer.OutChan()
````

Do you see how the syntax now looks exactly like single-value assignments, while in pracitce it is not sending any actual data yet, but "just" setting up a "channel-powered dataflow network",  through which we can later stream data indefinitely, from the network's input to it's output. In the authors opinion, this is neat indeed.

Finally, to compile and run the program above, just execute it by piping some FASTA file content, like this:
````bash
go build basecomplement.go
cat SomeFastaFile.fa | ./basecomplement > SomeFastaFile_Basecomplemented.fa
````

### A recap

So, what did we do above?

* We encapsulated concurrently running functions (go-routines) in structs.
* We made the function's incoming and outgoing channels into struct fields.
* We also created convenience functions for the "out-ports", that returns ready-made channels so that we can use the familiar single-assignment syntax to plug those channels into "inport" struct fields, to set up our data flow network using single-assignment syntax.

So, what did we get by that? - Well, certainly in technical terms not much more than the normal Go generator pattern shown earlier, but the author's opinion is that the struct-based approach makes the network-definition code a bit clearer and more intuitive to work with.

After writing the glow proof-of-concept library, the author also realized various intermediate forms between the generator pattern and the struct based pattern above that would also accomplish the same thing. Those might be a topic for another blog post, but the two examples above should at least show two rather different ways of achieving the same thing.

It is the hope that this post can at least spark some discussion and further exploration of patterns for truly component-oriented concurrent code in Go!
