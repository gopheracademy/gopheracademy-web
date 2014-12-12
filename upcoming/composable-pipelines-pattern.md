+++
Author = ["Samuel Lampa"]
date = "2014-12-12T00:00:00-08:00"
series = ["Advent 2014"]
title = "Pattern for pluggable pipeline components in Go"
+++

# A pattern for composable concurrent pipelines in Go

I came into Go from the python-world, which is so prevalent in my field, bioinformatics (I work mostly on the infrastructure side though, but do poke around with general applied bioinformatics stuff from time to time, out of sheer curiosity).

## Generator function in Python

In python I had just learned how to write composable lazy-evauated pipelines of string processing operations, using it's generator syntax, which by some simple testing showed to be both vastly less memory consuming, as well as a little faster, than their eager counterparts - and so I was hooked!

The Generator functionality in python basically means that you create a function that instead of returning a single value, such as a list of items, it will return a generator object, which you can iterate over, which is when it will start evaluating itself and yield the objects one by one, in the lazy evaluation fashion.

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
*(The N:s mean that the nucleotides at those positions are not known, and the first line, starting with '>', is just a label, that should be skipped, which will be seen in the following code examples)*

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

But then, coming in to Go, I was highly intrigued by all the new much more powerful concurrency patterns made possible in this language, elaborated in intriguing blog posts such as [the one on "concurrency patterns"](http://blog.golang.org/pipelines) and [the one on "advanced concurrency patterns"](http://blog.golang.org/advanced-go-concurrency-patterns).

Still, I was maybe even more happy to find that the simple and straight-forward generator pattern I knew from the python world, was very much possible in Go too!

So the above python code would read something like this in Go, using the generator pattern (leaving out some imports and const definitions, to improve readability):

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

As you can see, we basically replicate the behaviour of python generator functions (although in python it is less obvious how they work): Instead of returning a single value (list etc), we return a channel, on which we can later iterate using the "range" construct (like we do in the main() method above), to retrieve the elements in a lazy-evaluation fashion. Of course, in Go, we have the obvious benefit that this will now also run fully concurrently, using all of our CPU cores, if we wish!

*For reference, The generator pattern is [included in Rob Pike's Go Concurrency pattern slides](https://talks.golang.org/2012/concurrency.slide#25), as well as [listed on this site](http://www.golangpatterns.info/concurrency/generators).*

## Composability for general pipelines?

The generator pattern is truly a neat one, when you have a simple "thread-like" pipeline; Just a few processing steps operating on a stream of items from a previous function or process.

But what if we want to build up custom topologies of connected processing components with multiple (streaming) inputs and outputs? The generator patterns clearly has some limitations when going in this direction (EDIT: Or does it?!)

Are we then bound to bury those processing network topologies deep down in spaghetti-code of functions with hard-coded dependencies?

### Enter Flow-based programming

Well, to start with I have to say that the answer to that problem is solved already, in a way that is not covered in this article: You should definitely have a serious look at the [GoFlow library](https://github.com/trustmaster/goflow), which solves this problem more or less to it's core, relying on the solid foundation of the principles of [Flow-based programming](www.jpaulmorrison.com/fbp/) (FBP), invented by John P Morrison at IBM back in the 60's.

Flow based programming solves the complexity problem of complex processing network topologies in a thorough way by suggesting the use of named in- and ou-ports, channels with bounded buffers (already proveded by Go), and a separate network definition. This last thing, separating the network definition from the actual processing components, is what is so crucial to arrive at truly composable pipelines.

I personally had a great deal of fun, playing around with GoFlow, and even have an embryonic little library of proof-of-concept bioinformatics components, written for use with the framework, available at [github](https://github.com/samuell/blow). An example program, using it, can be found [here](https://gist.github.com/samuell/6164115).

### Flow-based like concepts in pure Go?

Still, looking to use Golang as a high-performance alternative to python, I was a little worried with one finding: The communication between components in GoFlow was not as fast as pure golang channels.

This surely doesn't matter for most normal use cases, where the increased flexibility and maintainability will give it it's clear benefits.

Still, for building easy-to-compose high performance bioinformatics programs in GoLang, it was worrying.

This lead me to start experimenting with how far one can go with Flow-based programming-like ideas, without using any 3rd party framework at all (this is btw territory that I have heard that GoFlow's creator, [Vladimir Sibiroc](https://github.com/trustmaster) did explore to some extent, although I haven't found any public information on the outcomes.)

The results of my findings was the following: A pattern for encapsulating concurrent processes in a struct, where the inputs and outputs are struct fields, of type channel (of some subsequent type).

This lets us set up of channels and do all the wiring of the processes, all totally outside of the components themselves.

### Show me the code

So, let's dive in to the actual code.
