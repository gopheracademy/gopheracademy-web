+++
author = ["Ian Chiles (fortytw2)"]
date = "2015-12-24T00:00:00-08:00"
series = ["Advent 2015"]
title = "Tiny Linux OSes with Go"

+++

> Small disclaimer: This is much more _fun_ than it is _useful_

For a while now, we've been seeing new "cloud" OSes crop up, like CoreOS and
RancherOS. These two both simply marry docker+linux to create a magic docker
runtime.

You may be familiar with "micro" Docker images built on top of Alpine Linux /
BusyBox base images. These "images" general weigh in around 10-30mb. But that's
before including the ~300mb OS + actual Docker daemon that they have to run on -
seems awfully bloated to me.

So, we need to ask ourselves, "who really needs docker when we can have an
entire OS that is purpose built to run your application, and nothing else? Short answer,
most people; fun answer, no one.

So let's get started. Make sure you have a copy of `QEMU` installed locally,
as we're going to _cheat_ and use its ability to boot a raw kernel bzImage
(the compressed, bootable kernel image) to avoid having to set up a real
bootloader (not too hard, but can be a real pain).

So, to get in the right mindset, we're going to use the excellent [termboy-go](https://github.com/dobyrch/termboy-go)
as our test application, an incredibly cool Gameboy Color emulator that runs
exclusively in the Linux console. In the end, we'll have a "single purpose OS"
that just runs a Gameboy emulator.

# Linux

First off, we're going to need a Linux kernel, so grab mainline from kernel.org,
run `make menuconfig` and then `make bzImage`. The default config should work,
just ensure that EXT4 is compiled *not* as a module.

<script type="text/javascript" src="https://asciinema.org/a/4adm32wk0t626mijvkexuvtmk.js" id="asciicast-4adm32wk0t626mijvkexuvtmk" async></script>

To boot our new kernel and make sure everything works, just use QEMU. You should
get a `Kernel panic - not syncing: VFS: Unable to mount root fs on unknown-block(0,0)`
which makes complete sense, as we haven't added any sort of root FS.

<script type="text/javascript" src="https://asciinema.org/a/12i116g0kltwwxn2c8jwmsq81.js" id="asciicast-12i116g0kltwwxn2c8jwmsq81" async></script>

Using a tool within `libguestfs`, `virt-make-fs` we can create a QEMU compatible
`.qcow2` from a directory (mine is a directory named `null`) and boot with it.

<script type="text/javascript" src="https://asciinema.org/a/bfk43pq02kdq0hq4das6q70f3.js" id="asciicast-bfk43pq02kdq0hq4das6q70f3" async></script>

Looking good, but we got another kernel panic - `Kernel panic - not syncing: No
working init found. Try passing init= option to kernel. See Linux
Documentation/init.txt for guidance.`. Luckily, this one's easy to solve, as,
funnily enough, any binary can be used as the kernel `init` - Go just so happens
to make wonderful, static binaries.

# Termboy-go

As such, we need a binary to be our `init`. So let's clone, then build a static
copy of `termboy-go` with `go build -a --ldflags="-s -X -linkmode external -extldflags -static"`,
the extra LDFLAGS are needed, as `termboy` uses a bit of CGo to handle keyboard
input. You'll probably need to slightly tweak termboy-go (hardcode in the ROM
path you want to use), but this isn't particularly difficult.

Unfortunately, `termboy-go` depends on having a copy of `setfont` in $PATH. This
is part of how it manages to render pixel-perfect 2D graphics in text-mode consoles.
Luckily, you can find a copy prebuilt from [minos.io](http://s.minos.io/bifrost/x86_64/),
or rip apart a copy with `statify` or `ermine` (convert dynamic binaries to static).

Copy this `termboy-go` into the directory you used `virt-make-fs` on, along with
a Gameboy Color ROM, and the copy of `setfont` you just obtained.

After this, you should be able to just boot your new gameboy-as-a-OS ->

[Imgur](http://i.imgur.com/8yORQXQ.png?1)

[Imgur](http://i.imgur.com/4ILCwQF.png?1)

# Conclusion

Just about any Go program can be built into a static binary, making it the perfect
choice for developing embedded systems, like this gameboy-emulator system we built
today. This single-OS approach may work well for running high-performance applications
in production, if you're interested in doing so, drop me a line.

However, there are many, far more ridiculous things that can be done using Go to
tackle early-userland OS problems - I have a bit of a distributed device manager
laying around, that synchronizes `/dev/` across all devices running it... and that's
just the beginning of the things that are possible here. Go is _fantastic_ for
this use case.

Feel free to get in touch with me via email (fortytw2 at gmail) for just about
any reason, especially if you have questions about this post, or find me on
on [twitter](https://twitter.com/fortytw2)/ [github](https://github.com/fortytw2).
