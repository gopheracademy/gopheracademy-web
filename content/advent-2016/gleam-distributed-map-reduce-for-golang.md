+++
author = ["Chris Lu"]
date = "2016-12-19T12:11:25-08:00"
series = ["Advent 2016"]
title = "Gleam: Distributed Map Reduce for Golang"

+++

After developing [Glow](https://github.com/chrislusf/glow) last year,
I came to realize the two limitations of Go for distributed data processing.

First, generics are needed. Of course, we can use reflection. But it is 
noticeably slower, to the point that I do not want to show the performance
numbers. Second, dynamic remote code execution is also
needed if we want to dynamically adjust the execution plan. We could pre-build
all the execution DAGs first and choose one of them during run time. But it
is very limiting.

As everyone else here, I enjoyed the beauty of Go. How to make it work for
big data?

Then I found LuaJIT. It seems an ideal complement to Go. For many who do
not know LuaJIT, its performance is on the same level as C, Java, Go. Being
a scripting language, it perfectly resolves the two limitations above:
generics and dynamic remote code execution. Lua was created as a small and
embeddable language. Go should embrace it. (BTW, LuaJIT's FFI is really
easy to call an external C library function, much simpler than CGO.)

Additionally, Unix Pipe tools, e.g., "sort", "tr", "uniq", etc, 
are also gems for data processing. But being single threaded, they lack 
the power to tackle big data. Go should be able to help to scale them up,
to distributedly run these tools.

With [Gleam](https://github.com/chrislusf/gleam), we can combine pure Go, 
LuaJIT, and Unix Pipes, all the 3 powerful weapons together.

Let's understand Gleam architecture of how it works. 
Then I will cover 3 examples:

1. Sort a 1GB file in both standalone and distributed modes, and distributed
Unix sort mode, with performance comparison.
2. Sort a 10GB file in On-Disk mode.
3. Implement joining of CSV files.


# Architecture

Gleam code defines the execution data flow, via simple Map(), Reduce() 
operations. By default, the flow is executed locally. 

For any Lua code, a separate LuaJIT process is started and data is streamed
through it.

The flow can also be executed in a distributed cluster. The Gleam cluster
has (currently) one master and multiple agents. The master's only job is
to collect resource statuses from agents. And agents needs to report their 
statuses to the master.

The agents also manage intermediate data. The data can be streamed via memory,
or optionally go through disk for better scalability and robustness.

When ran in a distributed cluster, the Gleam code becomes a driver of 
the whole flow's execution. It requests resources from the master, and then
contacts the agents to schedule jobs on them.

One feature I like for Gleam is that the master and agents are very efficient.
They took about 8~1MB memory and almost no resource usages when idle. So
you can install it anywhere, e.g., on or close to the source database  
so the data be processed locally.

# Installation

First, follow https://github.com/chrislusf/gleam/wiki/Installation

You just need to install LuaJIT and add one modified MessagePack lua file
to the Lua library load path.

For distributed mode, you need to setup the Gleam cluster. It's actually 
super simple. Just need to run
```sh
// on the master machine
$ gleam master
// on the agent machines
$ gleam agent --port=45327 --master="localhost:45326"
$ gleam agent --port=45328 --master="localhost:45326"
...
```
See https://github.com/chrislusf/gleam/wiki/Gleam-Cluster-Setup

# Example 1: Sorting 1GB file

## Input Data
The data input is 1GB gensort generated text data file.
http://www.ordinal.com/gensort.html

```sh
$ gensort -a 10737418 record_1Gb_input.txt
```

Before coding anything, let's use Unix sort to see how long it takes.
On my Mac(2.8GHz Intel Core i7, 16GB DDR3, APPLE SSD SM0512G),

```sh
$ time sort -k 1 ~/Desktop/record_1Gb_input.txt > result.txt

real	4m57.111s
user	4m55.379s
sys	0m1.309s
```
Peak Memory used: 1.46 GB, CPU is about 100%.

## Gleam Standalone mode

The complete source code is here. It has these steps:

1. read the input txt file line by line
- extract the integer part as a string from each line (LuaJIT)
- hash the key and distribute data into 4 partitions
- sort the key, first locally, then merge then into one partition
- print out data to stdout.

There could be better algorithms. For example, instead of randomly hashing the
key to partitions, we can just distribute the lines by the first few bits 
to 4 partitions, local sort, and we can have 4 already sorted partitions.
But it loses the generality of the sorting code.

```go
package main

import (
	"os"

	"github.com/chrislusf/gleam/flow"
)

func main() {
	flow.New().TextFile(
	  "/Users/chris/Desktop/record_1Gb_input.txt",
	).Map(`
	  function(line)
	    return string.sub(line, 1, 10), string.sub(line, 13)
	  end
	`).Partition(4).Sort().Fprintf(os.Stdout, "%s  %s\n").Run()
}
```

For Lua newbies, you may notice how similar Lua is to Go. Lua is 
a pretty simple language. If you feel Go is easy, 
[Lua is simpler](http://tylerneylon.com/a/learn-lua/).
For Gleam, you do not need to understand the advanced parts, 
such as table, metatable.


Result:
```sh
$ time ./gleam_sort > result1.txt

real	2m55.412s
user	7m26.631s
sys	1m3.288s

```
Peak Memory used: 5.29 GB, CPU is about 425%.

The code used more CPU and memory than basic Unix sort. The actual sorting
time is shorter, from 4m57s reduced to 2m55s, 59% of the baseline.

## Gleam Distributed mode with Unix Sort

Next example needs a Gleam cluster. See 
https://github.com/chrislusf/gleam/wiki/Gleam-Cluster-Setup for instructions.

This piece of code does the following:

1. read the input txt file line by line
2. extract the integer part as a string from each line (LuaJIT)
3. hash the key and distribute data into 4 partitions
4. sort the key locally for each partition (Unix sort command)
5. merge the 4 sorted partitions into one partition
6. print out data to stdout.

```go
package main

import (
	"os"

	"github.com/chrislusf/gleam/distributed"
	"github.com/chrislusf/gleam/flow"
)

func main() {
	flow.New().TextFile(
	  "/Users/chris/Desktop/record_1Gb_input.txt",
	).Map(`
	  function(line)
	    return string.sub(line, 1, 10), string.sub(line, 13)
	  end
	`).Partition(4).Pipe(`
	  sort -k 1
	`).MergeSortedTo(1).Fprintf(os.Stdout, "%s  %s\n").Run(distributed.Option())
}
```

Result
```sh
$ time ./gleam_sort > result2.txt
2016/11/25 17:25:22   taskGroup:Map.0-ScatterPartitions.0 : ScatterPartitions (5 MB)
2016/11/25 17:25:22 localhost:45326 allocated 1 executors with 5 MB memory.
2016/11/25 17:25:22   taskGroup:CollectPartitions.0-Pipe.0 : CollectPartitions (3 MB)
2016/11/25 17:25:22 localhost:45326 allocated 1 executors with 3 MB memory.
2016/11/25 17:25:22   taskGroup:CollectPartitions.2-Pipe.2 : CollectPartitions (3 MB)
2016/11/25 17:25:22   taskGroup:CollectPartitions.3-Pipe.3 : CollectPartitions (3 MB)
2016/11/25 17:25:22   taskGroup:CollectPartitions.1-Pipe.1 : CollectPartitions (3 MB)
2016/11/25 17:25:22 localhost:45326 allocated 3 executors with 9 MB memory.
2016/11/25 17:25:22   taskGroup:MergeSortedTo.0 : MergeSortedTo (5 MB)
2016/11/25 17:25:22 localhost:45326 allocated 1 executors with 5 MB memory.

real	2m54.228s
user	0m17.761s
sys	0m6.534s
```

There are 4 separated "sort" processes running, each peaked at 100% CPU and 50MB.
The total time is about the same as the standalone Gleam sort code.

The logs are just scheduling information, showing the tasks to do,
the amount of memory needed, the resources allocated, etc. Next example will
show a way to adjust the required memory.

The data flowing between each dataset are usually in MessagePack format.
But the data flowing to and from Pipe() are tab-separated lines, 
when each line has multiple fields. So there is a small amount of time 
for the extra conversion.

As you can see, as long as any program can work as Unix Pipes, Gleam
can make them work distributedly.

## Gleam Distributed mode with Gleam Sort

This piece of code takes these steps:

1. read the input txt file line by line
2. extract the integer part as a string from each line (LuaJIT)
3. hash the key and distribute data into 4 partitions
4. sort the key locally for each partition, then merge into one partition
5. print out data to stdout.

I added an optional hint to the flow, ```Hint(flow.TotalSize(1024))```, to
indicate the data size is 1024MB. When actually scheduling tasks on agents,
the driver program will ask Gleam master for corresponding resources, and 
only agents with enough resources can get the job. So with this hint, 
Gleam automatically allocates the memory.

```go
package main

import (
	"os"

	"github.com/chrislusf/gleam/distributed"
	"github.com/chrislusf/gleam/flow"
)

func main() {
	flow.New().TextFile(
	  "/Users/chris/Desktop/record_1Gb_input.txt",
	).Hint(flow.TotalSize(1024)).Map(`
	  function(line)
	    return string.sub(line, 1, 10), string.sub(line, 13)
	  end
	`).Partition(4).Sort().Fprintf(os.Stdout, "%s  %s\n").Run(distributed.Option())
}
```
Result:
```sh
$ time ./gleam_sort > result3.txt
2016/11/25 17:54:42   taskGroup:Map.0-ScatterPartitions.0 : ScatterPartitions (5 MB)
2016/11/25 17:54:42 localhost:45326 allocated 1 executors with 5 MB memory.
2016/11/25 17:54:42   taskGroup:CollectPartitions.0-LocalSort.0 : CollectPartitions (3 MB)
2016/11/25 17:54:42   taskGroup:CollectPartitions.0-LocalSort.0 : LocalSort (768 MB)
2016/11/25 17:54:42 localhost:45326 allocated 1 executors with 771 MB memory.
2016/11/25 17:54:42   taskGroup:CollectPartitions.2-LocalSort.2 : CollectPartitions (3 MB)
2016/11/25 17:54:42   taskGroup:CollectPartitions.2-LocalSort.2 : LocalSort (768 MB)
2016/11/25 17:54:42   taskGroup:CollectPartitions.1-LocalSort.1 : CollectPartitions (3 MB)
2016/11/25 17:54:42   taskGroup:CollectPartitions.1-LocalSort.1 : LocalSort (768 MB)
2016/11/25 17:54:42   taskGroup:CollectPartitions.3-LocalSort.3 : CollectPartitions (3 MB)
2016/11/25 17:54:42   taskGroup:CollectPartitions.3-LocalSort.3 : LocalSort (768 MB)
2016/11/25 17:54:42 localhost:45326 allocated 3 executors with 2313 MB memory.
2016/11/25 17:54:42   taskGroup:MergeSortedTo.0 : MergeSortedTo (5 MB)
2016/11/25 17:54:42 localhost:45326 allocated 1 executors with 5 MB memory.

real	1m55.188s
user	1m6.554s
sys	0m18.362s

```

Each of the 4 sorting executors peaked at about 868 MB, at 100% CPU.

The code used more CPU and memory than basic Unix sort. Instead of on-disk
merge sort, the Gleam sort is just all in-memory Timsort.
The actual sorting time is reduced from 4m57s to 1m55s, 39% of the baseline.

# Example 2: Sort large data with On-Disk mode.

The above examples all stream through pipes, or fit in the memory.
But sometimes the data size is more than the sum of all the machines 
in the cluster. In this case, we will need to fallback to OnDisk mode.

In this example, it hints the data size is 10GB, 10 times as the above example.
The number partitions is also changed from 4 to 40. And we add an OnDisk()
function to wrap the ```Partition(40).Sort()``` steps, to have them run
in on-disk mode.

```go
package main

import (
	"os"

	"github.com/chrislusf/gleam/distributed"
	"github.com/chrislusf/gleam/flow"
)

func main() {
	flow.New().TextFile(
	  "/Users/chris/Desktop/record_10GB_input.txt",
	).Hint(flow.TotalSize(10240)).Map(`
	  function(line)
	    return string.sub(line, 1, 10), string.sub(line, 13)
	  end
	`.OnDisk(func(d *flow.Dataset) *flow.Dataset {
		return d.Partition(40).Sort()
	}).Fprintf(os.Stdout, "%s  %s\n").Run(distributed.Option())
}
```
Result:
```sh
$ time ./gleam_sort > result4.txt
...
real	96m1.932s
user	14m3.423s
sys	3m12.283s

```

The numbers may not look that impressive. If sorting the same 1GB data 
in memory, it takes about 2 minutes. But this on-disk mode took about 
100 minutes for 10GB data.

It is due to the intensive disk IO. All the data within the
OnDisk() function will need to write to disk, so that the computer 
can process other fractions of data.

This is also why we want to use in-memory mode if possible. And, we still need
the capability to use on-disk mode when data is too big.

This OnDisk() mode also allows robust retries if anything breaks.

It is slower, but it will get the job done.

# Example 3: Joining CSV files
Data joining is another common use case for data processing. Here is a simple
example to illustrate how it works.

Assume there are file "a.csv" has fields "a1, a2, a3, a4, a5",
and file "b.csv" has fields "b1, b2, b3". 

We want to join the rows where a1 = b2. 

And the output format should be "a1, a4, b3".

```go
package main

import (
    "os"

    . "github.com/chrislusf/gleam/flow"
    "github.com/chrislusf/gleam/plugins/csv"
)

func main() {

    f := New()
    a := f.ReadFile(csv.New("a.csv")).Select(Field(1,4)) // a1, a4
    b := f.ReadFile(csv.New("b.csv")).Select(Field(2,3)) // b2, b3

    a.Join(b).Fprintf(os.Stdout, "%s,%s,%s\n").Run()  // a1, a4, b3

}

```
In the example, the CSV files are on local disk. This is the simplest form. 
The CSV files can also be read from HDFS or S3.

CSV is one of the storage format supported. There are different adapters 
for different data sources, e.g. Gleam can read from Cassandra in parallel.
More adapters are planned.

# Summary

Gleam is a simple and powerful distributed map reduce system.
By combining high performance LuaJIT and Go's powerful system programming,
Gleam can dynamically schedule jobs to run on remote computers via agents, 
intelligently allocate resources, and process the data in parallel.

Gleam is being actively worked on. Some of the goals are:

1. Add better fault tolerant error handling.
2. Add more data sources.
3. Later add a SQL layer on top of it.

Lots of work to be done. Welcome any kind of contribution!
