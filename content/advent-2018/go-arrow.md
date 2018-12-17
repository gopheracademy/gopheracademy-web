+++
author = ["Sebastien Binet"]
title = "Go and Arrow: building blocks for data science"
linktitle = "go-arrow"
date = 2018-12-18T00:00:00Z
series = ["Advent 2018"]
+++

Today we will see how [Arrow](https://arrow.apache.org) could be useful for data science, or -- really -- a lot of analysis' workloads.

## Lingua franca

In [Data Science](https://en.wikipedia.org/wiki/Data_science) and in many scientific fields, the _lingua franca_ is Python.

![xkcd-gravity](/postimages/advent-2018/go-arrow/xkcd-python.png)

This means that a lot of the libraries are written in Python, with the CPU intensive parts leveraging either [NumPy](https://www.numpy.org/), [SciPy](https://www.scipy.org/) or, directly, a C/C++ library wrapped with the CPython C-API.
This also means that:

- the vast majority of the analysis pipelines are written in Python;
- it is not completely straightforward to migrate parts of these pipelines to other languages, in an adiabatic and piecewise process.

So why would anyone use Go for this?

Well... the usual reasons:

- Go is easier to deploy,
- Go is faster than pure Python,
- it is easier to write concurrent code in Go,
- Go code and results are easy to reproduce,
- Go code is more amenable to mechanical refactoring,
- Go code tends to be more robust at scale than Python code.

The real question is: do we have all the building blocks, all the libraries to write a modern, robust, efficient analysis pipeline in Go?

Most of the science-y things one might carry out with the Python scientific stack (NumPy/SciPy) can be performed with [Gonum](https://gonum.org):

- linear algebra, matrices,
- differentiation,
- statistics, probability distributions,
- integration, optimization,
- network creation,
- Fourier transformations,
- plots.

There are also a few packages that enable file-level interoperability with many useful data science oriented or scientific "file formats":

- [CSV](https://godoc.org/encoding/csv)
- [SQL](https://godoc.org/database/sql)
- [Numpy data files](https://godoc.org/github.com/sbinet/npyio)
- [HDF5](https://godoc.org/gonum.org/v1/hdf5)
- [FITS](https://godoc.org/github.com/astrogo/fitsio)
- [HDFS](https://godoc.org/github.com/colinmarc/hdfs)
- [Parquet](https://godoc.org/github.com/xitongsys/parquet-go/parquet)
- [ORC](https://godoc.org/github.com/scritchley/orc)
- [Cassandra](https://godoc.org/github.com/gocql/gocql)
- ...

Many basic ingredients are already there, even if some are still not there, yet (_e.g._ [ODE](https://en.wikipedia.org/wiki/Ordinary_differential_equation) is not implemented in [Gonum](https://gonum.org).)

But even if basic file-level interoperability is (somewhat) achieved, one still needs to implement readers, writers and converters from one file format to any another when integrating with a new analysis pipeline:

![file formats](/postimages/advent-2018/go-arrow/arrow-copy.png)

In many scenarii, this implies a large fraction of the computation is wasted on serialization and deserialization, reimplementing over and over the same features for converting from `format-1` to `format-n` or from `format-1` to `format-m`, _etc..._

What if we had a common data layer between all these analysis pipelines, a common tool that would also be efficient?

## Arrow

What is [Arrow](https://arrow.apache.org)? From the website:

```
Apache Arrow is a cross-language development platform for in-memory data.

It specifies a standardized language-independent columnar memory format for flat
and hierarchical data, organized for efficient analytic operations on modern
hardware.
It also provides computational libraries and zero-copy streaming messaging and
interprocess communication.
```

![arrow shared](/postimages/advent-2018/go-arrow/arrow-shared.png)

The idea behind Arrow is to describe and implement a cross-language mechanism for efficiently sharing data across languages and processes.
This is embodied as an `arrow::array` in C++, the reference implementation.
Languages currently supported include C, C++, Java, JavaScript, Python, and Ruby.

... and -- of course -- Go :)

[InfluxData](https://www.influxdata.com/) contributed the original Go code for Apache Arrow, as announced [here](https://www.influxdata.com/blog/influxdata-apache-arrow-go-implementation/).
Initially, the Go Arrow package implemented by [Stuart Carnie](https://twitter.com/stuartcarnie) had support for:

- primitive arrays (`{u,}int{8,16,32,64}`, `float{32,64}`, ...)
- parametric types (`timestamp`)
- memory management
- typed metadata
- SIMD math kernels (_via_ a nice automatic translation tool - `c2goasm` - to extract vectorized code produced by CLang, see the [original blog post](https://www.influxdata.com/blog/influxdata-apache-arrow-go-implementation/) for more details)

Since then, a few contributors provided implementations for:

- list arrays
- struct arrays
- time arrays
- loading `CSV` data to Arrow arrays,
- tables and records,
- tensors (_a.k.a._ n-dimensional arrays.)

But what are those arrays?

## Arrow arrays

From the Arrow perspective, an array is a sequence of values with known length, all having the same type.
Arrow arrays have support for "missing" values, or `null` slots in the array.
The number of `null` slots is stored in a bitmap, itself stored as part of the array.
Arrays with no `null` slot can choose not to allocate that bitmap.

Let us consider an array of `int32`s to make things more concrete:

```
[1, null, 2, 4, 8]
```

Such an array would look like:

```
* Length: 5, Null count: 1
* Null bitmap buffer:

|Byte 0 (validity bitmap) | Bytes 1-63            |
|-------------------------|-----------------------|
| 00011101                | 0 (padding)           |

* Value Buffer:

|Bytes 0-3 |     4-7     |  8-11 | 12-15 | 16-19 |    20-63    |
|----------|-------------|-------|-------|-------|-------------|
| 1        | unspecified |   2   |   4   |   8   | unspecified |
```

Arrow specifies the expected [memory layout](https://arrow.apache.org/docs/memory_layout.html) of arrays in a document.

How does it look like in Go?
The Go implementation for the Apache Arrow standard is documented here:

- [github.com/apache/arrow/go/arrow](https://godoc.org/github.com/apache/arrow/go/arrow): data types, metadata, schemas, ...
- [github.com/apache/arrow/go/arrow/array](https://godoc.org/github.com/apache/arrow/go/arrow/array): array types,
- [github.com/apache/arrow/go/arrow/memory](https://godoc.org/github.com/apache/arrow/go/arrow/memory): support for low-level allocation of memory,
- [github.com/apache/arrow/go/arrow/csv](https://godoc.org/github.com/apache/arrow/go/arrow/csv): support for reading CSV files into Arrow arrays,
- [github.com/apache/arrow/go/arrow/tensor](https://godoc.org/github.com/apache/arrow/go/arrow/tensor): types for n-dimensional arrays.

The main entry point is the `array.Interface`:

```go
package array

// A type which satisfies array.Interface represents
// an immutable sequence of values.
type Interface interface {
	// DataType returns the type metadata for this instance.
	DataType() arrow.DataType

	// NullN returns the number of null values in the array.
	NullN() int

	// NullBitmapBytes returns a byte slice of the validity bitmap.
	NullBitmapBytes() []byte

	// IsNull returns true if value at index is null.
	// NOTE: IsNull will panic if NullBitmapBytes is not empty and 0 > i ≥ Len.
	IsNull(i int) bool

	// IsValid returns true if value at index is not null.
	// NOTE: IsValid will panic if NullBitmapBytes is not empty and 0 > i ≥ Len.
	IsValid(i int) bool

	Data() *Data

	// Len returns the number of elements in the array.
	Len() int

	// Retain increases the reference count by 1.
	// Retain may be called simultaneously from multiple goroutines.
	Retain()

	// Release decreases the reference count by 1.
	// Release may be called simultaneously from multiple goroutines.
	// When the reference count goes to zero, the memory is freed.
	Release()
}
```

Most of the interface should be self-explanatory.
The careful reader might notice the `Retain/Release` methods that are used to manage the memory used by arrays.
Gophers might be surprised by this kind of low-level memory management, but this is needed to allow for `mmap`- or GPU-backed arrays.

Another interesting piece of this interface is the `Data()` method that returns a value of type `*array.Data`:

```go
package array

// A type which represents the memory and metadata for an Arrow array.
type Data struct {
	refCount int64
	dtype    arrow.DataType
	nulls    int
	offset   int
	length   int
	buffers  []*memory.Buffer
	kids     []*Data
}
```

The `array.Data` type is where all the semantics of an Arrow arrays are encoded, as per the [Arrow specifications](https://arrow.apache.org/docs/memory_layout.html).

The last piece to understand the memory layout of a Go Arrow array is `memory.Buffer`:

```go
package memory

type Buffer struct {
	refCount int64
	buf      []byte
	length   int
	mutable  bool
	mem      Allocator
}
```

In a nutshell, Arrow arrays -- and the `array.Interface` -- can be seen as the cousins of the [buffer protocol](https://www.python.org/dev/peps/pep-3118/) from Python which has been instrumental in (pythonic) science.

Let us come back to the `int32` array.
It is implemented like so in Go:

```go
package array

type Int32 struct {
	array
	values []int32
}

func NewInt32Data(data *Data) *Int32 { ... }

type array struct {
	refCount        int64
	data            *Data
	nullBitmapBytes []byte
}
```

Creating a new array value holding `int32`s is done with `array.NewInt32Data(data)` where `data` holds a carefully crafted `array.Data` value with the needed memory buffers and sub-buffers (for the `null` bitmap.)
As this can be a bit unwiedly, a helper type is provided to create Arrow arrays from scratch:

```go
mem := memory.NewGoAllocator()
bld := array.NewInt32Builder(mem)
defer bld.Release()

// create an array with 4 values, no null
bld.AppendValues([]int32{1, 2, 3, 4}, nil)

arr1 := bld.NewInt32Array() // materialize the array
defer arr1.Release()        // make sure we release memory, eventually.

// arr1 = [1 2 3 4]
fmt.Printf("arr1 = %v\n", arr1)

// create an array with 5 values, 1 null
bld.AppendValues(
	[]int32{1, 2, 3, 4, 5},
	[]bool{true, true, true, false, true},
)

arr2 := bld.NewInt32Array()
defer arr2.Release()

// arr2 = [1 2 3 (null) 5]
fmt.Printf("arr2 = %v\n", arr2)
```

Arrow arrays can also be sub-sliced:

```go
sli := array.NewSlice(arr2, 1, 5).(*array.Int32)
defer sli.Release()

// slice = [2 3 (null) 5]
fmt.Printf("slice = %v\n", sli)
```

Similar code can be written for arrays of `float64`, `float32`, unsigned integers, etc...
But not everything can be mapped or simply expressed in terms of these basic primitives.
What if we need a more detailed data type?
That is expressed with `List`s or `Struct`s.
Let us imagine we need to represent an array of "Person" type like:
```go
type Person struct {
	Name string
	Age  int32  // presumably, int8 should suffice.
}
```

Structured types in Arrow can be seen as entries in a database, with a specified schema.
While many databases are row oriented, data is column-oriented in Arrow:

![simd](/postimages/advent-2018/go-arrow/arrow-simd.png)

The reasoning behind organizing data along columns instead of rows is that:

- compression should work better for data of the same type, (presumably, data should be relatively similar)
- many workloads only care about a few columns to perform their analyses, the other columns can be left at rest on disk,
- one can leverage [vectorized instructions](https://en.wikipedia.org/wiki/SIMD) to carry operations.

This is the well-known _structure of arrays_ (`SoA`) _vs_ _arrays of structure_ (`AoS`) [debate](https://en.wikipedia.org/wiki/AOS_and_SOA).

With Arrow, the `Person` type defined previously could be implemented as the following structure of arrays:

```go
mem := memory.NewGoAllocator()
dtype := arrow.StructOf([]arrow.Field{
	{Name: "Name", Type: arrow.ListOf(arrow.PrimitiveTypes.Uint8)},
	{Name: "Age",  Type: arrow.PrimitiveTypes.Int32},
}...)

bld := array.NewStructBuilder(mem, dtype)
defer bld.Release()
```

A complete example is provided [here](https://godoc.org/github.com/apache/arrow/go/arrow#example-package--StructArray).

## Arrow tables

As hinted during this blog post, Arrow needs to be able to interoperate with industry provided or scientific oriented databases, to load all the interesting data into memory, as efficiently as possible.
Thus, it stands to reason for Arrow to provide tools that maps to operations usually performed on databases.
In Arrow, this is expressed as `Table`s:

```go
package array

// Table represents a logical sequence of chunked arrays.
type Table interface {
    Schema() *arrow.Schema
    NumRows() int64
    NumCols() int64
    Column(i int) *Column

    Retain()
    Release()
}
```

The `Table` interface allows to describe the data held by some database, and access it _via_ the `array.Column` type -- a representation of chunked arrays: arrays that may not be contiguous in memory.

The code below creates a pair of records `rec1` and `rec2` (each record can be seen as a set of rows in a database.)
Each record contains two columns, the first one containing `int32` values and the second one `float64` values.
Once a few entries have been added to each of the two records, a `Table` reader is created from these in-memory records.
The chunk size for this `Table` reader is set to `5`: each column will hold at most 5 entries during the iteration over the in-memory `Table`.

```go
package main

import (
	"fmt"

	"github.com/apache/arrow/go/arrow"
	"github.com/apache/arrow/go/arrow/array"
	"github.com/apache/arrow/go/arrow/memory"
)

func main() {
	mem := memory.NewGoAllocator()

	schema := arrow.NewSchema(
		[]arrow.Field{
			arrow.Field{Name: "f1-i32", Type: arrow.PrimitiveTypes.Int32},
			arrow.Field{Name: "f2-f64", Type: arrow.PrimitiveTypes.Float64},
		},
		nil, // no metadata
	)

	b := array.NewRecordBuilder(mem, schema)
	defer b.Release()

	b.Field(0).(*array.Int32Builder).AppendValues(
		[]int32{1, 2, 3, 4, 5, 6},
		nil,
	)
	b.Field(0).(*array.Int32Builder).AppendValues(
		[]int32{7, 8, 9, 10},
		[]bool{true, true, false, true},
	)
	b.Field(1).(*array.Float64Builder).AppendValues(
		[]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 
		nil,
	)

	rec1 := b.NewRecord()
	defer rec1.Release()

	b.Field(0).(*array.Int32Builder).AppendValues([]int32{
		11, 12, 13, 14, 15, 16, 17, 18, 19, 20}, 
		nil,
	)
	b.Field(1).(*array.Float64Builder).AppendValues([]float64{
		11, 12, 13, 14, 15, 16, 17, 18, 19, 20}, 
		nil,
	)

	rec2 := b.NewRecord()
	defer rec2.Release()

	tbl := array.NewTableFromRecords(schema, []array.Record{rec1, rec2})
	defer tbl.Release()

	tr := array.NewTableReader(tbl, 5)
	defer tr.Release()

	n := 0
	for tr.Next() {
		rec := tr.Record()
		for i, col := range rec.Columns() {
			fmt.Printf("rec[%d][%q]: %v\n", n, rec.ColumnName(i), col)
		}
		n++
	}
}
```

Executing the code above would result in:
```
$> go run ./table-reader.go
rec[0]["f1-i32"]: [1 2 3 4 5]
rec[0]["f2-f64"]: [1 2 3 4 5]
rec[1]["f1-i32"]: [6 7 8 (null) 10]
rec[1]["f2-f64"]: [6 7 8 9 10]
rec[2]["f1-i32"]: [11 12 13 14 15]
rec[2]["f2-f64"]: [11 12 13 14 15]
rec[3]["f1-i32"]: [16 17 18 19 20]
rec[3]["f2-f64"]: [16 17 18 19 20]
```

## CSV data

Finally, to conclude with our quick whirlwind overview of what Go Arrow can provide now, let us consider the [arrow/csv](https://godoc.org/github.com/apache/arrow/go/arrow/csv) package.
As mentioned earlier, many analysis pipelines start with ingesting [CSV](https://en.wikipedia.org/wiki/Comma-separated_values).
The Go standard library already provides a package to decode and encode data using this comma-separated values (CSV) "format".
But what the [encoding/csv](https://godoc.org/encoding/csv) package exposes is slices of `string`s, not values of a specific type.
This is left to the user of that package.

The [arrow/csv](https://godoc.org/github.com/apache/arrow/go/arrow/csv) package leverages the `arrow.Schema` and `array.Record` types to provide a typed and (eventually) scalable+optimized API.

```go
package main

import (
	"bytes"
	"fmt"

	"github.com/apache/arrow/go/arrow"
	"github.com/apache/arrow/go/arrow/csv"
)

func main() {
	f := bytes.NewBufferString(`## a simple set of data: int64;float64;string
0;0;str-0
1;1;str-1
2;2;str-2
3;3;str-3
4;4;str-4
5;5;str-5
6;6;str-6
7;7;str-7
8;8;str-8
9;9;str-9
`)

	schema := arrow.NewSchema(
		[]arrow.Field{
			{Name: "i64", Type: arrow.PrimitiveTypes.Int64},
			{Name: "f64", Type: arrow.PrimitiveTypes.Float64},
			{Name: "str", Type: arrow.BinaryTypes.String},
		},
		nil, // no metadata
	)
	r := csv.NewReader(
		f, schema,
		csv.WithComment('#'), csv.WithComma(';'),
		csv.WithChunk(3),
	)
	defer r.Release()

	n := 0
	for r.Next() {
		rec := r.Record()
		for i, col := range rec.Columns() {
			fmt.Printf("rec[%d][%q]: %v\n", i, rec.ColumnName(i), col)
		}
		n++
	}
}
```

Running the code above will result in:

```
$> go run ./read-csv.go
rec[0]["i64"]: [0 1 2]
rec[1]["f64"]: [0 1 2]
rec[2]["str"]: ["str-0" "str-1" "str-2"]
rec[0]["i64"]: [3 4 5]
rec[1]["f64"]: [3 4 5]
rec[2]["str"]: ["str-3" "str-4" "str-5"]
rec[0]["i64"]: [6 7 8]
rec[1]["f64"]: [6 7 8]
rec[2]["str"]: ["str-6" "str-7" "str-8"]
rec[0]["i64"]: [9]
rec[1]["f64"]: [9]
rec[2]["str"]: ["str-9"]
```

## Conclusions

This post has only scratched the surface of what can be done with Go Arrow and how it works under the hood.
For example, we have not talked about how the typesafe array builders and array types are generated: this is - BTW - an area where the [Go2 draft proposal for generics](https://go.googlesource.com/proposal/+/master/design/go2draft-generics-overview.md) would definitely help.

There also many features available in C++/Python Arrow that are yet to be implemented in Go Arrow.
The main remaining one is perhaps implementing the [IPC protocol](https://arrow.apache.org/docs/ipc.html) that Arrow specifies.
This would allow to create polyglot, distributed applications with an eye towards data science.

In the same vein, the Arrow `Table`, `Record` and `Schema` types could be seen as building blocks for creating a [dataframe](https://pandas.pydata.org/pandas-docs/stable/generated/pandas.DataFrame.html) package interoperable with the ones from Python, R, etc...
Finally, Go Arrow tensors could be used as an efficient vehicle to transfer data between various machine learning packages ([ONNX](https://github.com/onnx/onnx), [Gorgonia](http://gorgonia.org/).)

Gophers, follow the arrows and send your pull requests!
