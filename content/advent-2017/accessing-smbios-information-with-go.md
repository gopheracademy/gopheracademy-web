+++
author = ["Matt Layher"]
date = "2017-12-19T08:00:00+00:00"
title = "Accessing SMBIOS information with Go"
series = ["Advent 2017"]
+++

While speaking with coworkers recently, one of them posed a question:

> How can we discover the capacity of each memory DIMM in a machine?

Some veteran Linux users may be familiar with the
[`dmidecode`](https://linux.die.net/man/8/dmidecode) utility, which
can access SMBIOS/DMI information exposed by hardware.  This utility can expose
a huge amount of information about the hardware and BIOS software on a machine.
But how does it work under the hood?

This post will explain how to access this information, and demonstrate the
[open source `go-smbios` package](https://github.com/digitalocean/go-smbios)
that can be used to retrieve and leverage this information in Go.

## Introduction to SMBIOS/DMI

[System Management BIOS](https://en.wikipedia.org/wiki/System_Management_BIOS),
or SMBIOS, is a specification that defines data structures that can be used to
access information exposed by hardware and the system BIOS in a standardized way.

SMBIOS is often confused with
[Desktop Management Interface](https://en.wikipedia.org/wiki/Desktop_Management_Interface),
or DMI, but it is essentially an evolution of the original DMI specification.
This is where the Linux `dmidecode` utility's name originates.

What kind of information is exposed by this interface?  We can use the `dmidecode`
utility to take a peek at some of this information.

```
$ sudo dmidecode | head -n 12
# dmidecode 3.0
Getting SMBIOS data from sysfs.
SMBIOS 2.7 present.
69 structures occupying 3435 bytes.
Table at 0x000E0FC0.

Handle 0x0000, DMI type 4, 42 bytes
Processor Information
        Socket Designation: CPU Socket - U3E1
        Type: Central Processor
        Family: Core i7
        Manufacturer: Intel(R) Corporation
```

The utility exposes quite a lot of information, but even from this small sample,
we can note several important features:

- the version of SMBIOS present on the machine
- how many SMBIOS structures are available, and how many bytes they occupy
- the memory address of the SMBIOS structures table
- a structure with a type, length, and handle field, and some information

SMBIOS has dozens of different structures, and each can encode a variety of data.

## Retrieving SMBIOS information with Go

SMBIOS information consists of two crucial pieces: an "entry point" structure,
and a table of data structures which carry SMBIOS information.

On modern Linux machines, the entry point structure and table can be found using
two special files in sysfs:

```
$ ls /sys/firmware/dmi/tables/
DMI  smbios_entry_point
```

While this is certainly convenient, the standard approach on other UNIX-like
operating systems is to directly scan system memory for a magic string, using
[`/dev/mem`](http://man7.org/linux/man-pages/man4/mem.4.html).

The basic algorithm is:

- start scanning for the magic prefix "_SM" at memory address `0x000f0000`
- iterate one "paragraph" (16 bytes) of memory at a time until we either find an entry point or reach the memory address `0x000fffff`
- determine if the entry point is the 32-bit or 64-bit variety, and decode it
- use information from the entry point to find the address and size of the structures table

## Discovering and decoding SMBIOS entry points in Go

In simplified Go code (please always check your errors), discovering the entry point
looks something like:

```go
// Open /dev/mem and seek to the starting memory address.
const start, end = 0x000f0000, 0x000fffff

mem, _ := os.Open("/dev/mem")
_, _ = mem.Seek(start, io.SeekStart)

// Iterate one "paragraph" of memory at a time until we either find the entry point
// or reach the end bound.
const paragraph = 16
b := make([]byte, paragraph)

var addr int
for addr = start; addr < end; addr += paragraph {
	_, _ = io.ReadFull(mem, b)

	// Both the 32-bit and 64-bit entry point have a similar prefix.
	if bytes.HasPrefix(b, []byte("_SM")) {
		return addr, nil
	}
}
``` 

Now that we've discovered the location of the entry point in memory, we can
begin decoding the entry point structure.  Depending on your machine, you may
encounter a 32-bit or 64-bit SMBIOS entry point.

```go
// Prevent unbounded reads since this structure should be small.
b, _ := ioutil.ReadAll(io.LimitReader(mem, 64))
if l := len(b); l < 4 {
	return nil, fmt.Errorf("too few bytes for SMBIOS entry point magic: %d", l)
}

// Did we find a 32-bit entry point, or 64-bit entry point?
switch {
case bytes.HasPrefix(b, []byte("_SM_")):
	return parse32(b)
case bytes.HasPrefix(b, []byte("_SM3_")):
	return parse64(b)
}
```

I'll spare you the details of each entry point structure, but they contain some
key information exposed by the `dmidecode` utility, as discussed previously:

- the version of SMBIOS present on the machine
- how many SMBIOS structures are available, and how many bytes they occupy
- the memory address of the SMBIOS structures table

With this information, we can finally begin decoding the structures table.

## Decoding the SMBIOS structure table in Go

Each SMBIOS structure contains a header that indicates:

- the type of the structure (BIOS information, memory information, etc.)
- the length of the structure in bytes
- a "handle" that can be used to point to related information in another structure

Following the header, each structure contains a "formatted" section that carries
arbitrary bytes, and optionally, zero or more "strings" that the formatted section
can point to.  This data must be decoded in a manner specific to each structure
type, as laid out in the
[SMBIOS specification](https://www.dmtf.org/sites/default/files/standards/documents/DSP0134_3.1.1.pdf).

To start: let's jump to the location of the table and begin decoding it.  The
structures stream continues until a special "end of table" structure is reached.

```go
// Seek to the address of the structures table, and set up a decoder.
_, _ = mem.Seek(addr, io.SeekStart)
dec := newDecoder(mem)

var ss []*Structure
for {
	s, _ := dec.next()
	ss = append(ss, s)

	// End-of-table structure indicates end of stream.
	if s.Header.Type == typeEndOfTable {
		break
	}
}
```

Within our `decoder.next` method, we must deal with each structure's header,
formatted section, and zero or more strings:

```go
// Decode the header structure.
h, _ := dec.parseHeader()

// Length of formatted section is length specified by header, minus
// the length of the header itself.
l := int(h.Length) - headerLen
fb, _ := dec.parseFormatted(l)

// Strings may or may not be present; only advance the decoder
// if they are.
ss, _ := dec.parseStrings()

return &Structure{
	Header:    *h,
	Formatted: fb,
	Strings:   ss,
}
```

This process continues until the end of table structure is reached, or an
EOF is returned by the stream.

## Decoding memory DIMM information from SMBIOS structures in Go

As previously mentioned, the formatted section and strings in an SMBIOS structure
can be used to retrieve information stored in a specific format.  The
[SMBIOS specification](https://www.dmtf.org/sites/default/files/standards/documents/DSP0134_3.1.1.pdf)
can be used as a reference for the format of individual structures.

With that said, let's take a look back at our problem statement:

> How can we discover the capacity of each memory DIMM in a machine?

The structure we want to decode is the "Memory Device (Type 17)" structure.
One of these structures appears in the SMBIOS stream, per DIMM slot on the
motherboard.

```go
// Only look at memory devices.
if s.Header.Type != 17 {
	continue
}

// Formatted section contains a variety of data, but only parse the DIMM size.
size := int(binary.LittleEndian.Uint16(s.Formatted[8:10]))
// String 0 is the DIMM slot's identifier.
name := s.Strings[0]

// If 0, no DIMM present in this slot.
if size == 0 {
	fmt.Printf("[% 3s] empty\n", name)
	continue
}

// An extended uint32 DIMM size field appears if 0x7fff is present in size.
if size == 0x7fff {
	size = int(binary.LittleEndian.Uint32(s.Formatted[24:28]))
}

// Size units depend on MSB.  Little endian MSB for uint16 is in second byte.
// 0 means megabytes, 1 means kilobytes.
unit := "KB"
if s.Formatted[9]&0x80 == 0 {
	unit = "MB"
}

fmt.Printf("[% 3s] DIMM: %d %s\n", name, size, unit)
```

Now that we've put this all together, we can see the results from two of my
Linux machines at home:

```
desktop $ sudo ./lsdimms
SMBIOS 2.7.0
[ChannelA-DIMM0] DIMM: 4096 MB
[ChannelA-DIMM1] DIMM: 4096 MB
[ChannelB-DIMM0] DIMM: 4096 MB
[ChannelB-DIMM1] DIMM: 4096 MB
```
```
server $ sudo ./lsdimms
SMBIOS 3.0.0
[DIMM 0] empty
[DIMM 1] DIMM: 16384 MB
[DIMM 0] empty
[DIMM 1] DIMM: 16384 MB
```

There are dozens of other structure type available, but with this information,
we can now see the exact configuration and capacity of memory DIMMs in my machines.

## Summary

As you can see, a great deal of useful information about your machine can be
exposed using SMBIOS.  Check out the `dmidecode` utility to see what kind of
information is available!  If you'd like to incorporate this data in your Go
programs, I recommend that you check out the 
[`go-smbios` package](https://github.com/digitalocean/go-smbios).

This package handles the nitty-gritty details of exposing SMBIOS data from a
variety of operating systems (most UNIX-like systems, but I'd love to add
macOS and Windows support!).  At this time, it doesn't contain any code for
decoding specific structure types, but this is something I'd love to incorporate
in a higher-level package in the future!  If you'd like to collaborate, please
reach out!

Finally, if you ever find yourself working on a text or binary format parser,
I highly encourage trying out [go-fuzz](https://github.com/dvyukov/go-fuzz) to
discover any potential crashes in your parser.  `go-fuzz` is an invaluable tool,
and liberal use of it will save you many headaches down the road.  For a great
introduction to `go-fuzz`, check out
[Damian Gryski's detailed walkthrough](https://medium.com/@dgryski/go-fuzz-github-com-arolek-ase-3c74d5a3150c).

If you have any questions, feel free to contact me: "mdlayher" on
[Gophers Slack](https://gophers.slack.com/)!  You can also find me on both
[GitHub](https://github.com/mdlayher) and [Twitter](https://twitter.com/mdlayher)
with the same username.

## Links

- [dmidecode](https://linux.die.net/man/8/dmidecode)
- [Package go-smbios](https://github.com/digitalocean/go-smbios)
- [System Management BIOS](https://en.wikipedia.org/wiki/System_Management_BIOS)
- [Desktop Management Interface](https://en.wikipedia.org/wiki/Desktop_Management_Interface)
- [/dev/mem](http://man7.org/linux/man-pages/man4/mem.4.html)
- [SMBIOS 3.1.1 specification](https://www.dmtf.org/sites/default/files/standards/documents/DSP0134_3.1.1.pdf)