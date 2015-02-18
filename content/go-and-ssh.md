+++
author = ["Jaime Pillora"]
date = "2015-01-13T09:00:00+00:00"
title = "Go and the Secure Shell protocol"
+++

# What is the Secure Shell protocol?

Many of us use [`ssh(1)`](http://www.openbsd.org/cgi-bin/man.cgi/OpenBSD-current/man1/slogin.1) everyday to administer servers around the world. The [**S**ecure **Sh**ell protocol](http://www.ietf.org/rfc/rfc4254.txt), however, allows us to do much more than open secure shells. This post will attempt to the raise awareness of SSH and suggest various use cases for SSH in the modern web ecosystem.

SSH is a transport layer security protocol that provides a secure method for data communication. On top of a single data channel, such as TCP, the SSH protocol provides us with:

1. A secure handshake using four authentication methods, providing **authenticity**
1. A two-way encrypted and verified data stream, providing **privacy** and **integrity**
1. The ability to create many logical connections, providing a clean multiplexed connection abstraction
1. Connection metadata "request" channels, terminal management and remote program execution

![ssh diagram](https://docs.google.com/drawings/d/11e0kDfg0IUm7DHz76g7Akkbi8SBIHU-9e2QrCVw-1Tw/pub?w=400&h=300)

<!--
comment on the diagram here
https://docs.google.com/drawings/d/11e0kDfg0IUm7DHz76g7Akkbi8SBIHU-9e2QrCVw-1Tw/edit?usp=sharing
-->

# What about TLS?

In this post-Snowden era, authenticity, privacy and integrity are becoming a requirement for all modern networked applications. These three requirements are also met by the popular protocol, [**T**ransport **L**ayer **S**ecurity](http://en.wikipedia.org/wiki/Transport_Layer_Security) along with its equivalent ([though less secure](https://www.openssl.org/~bodo/ssl-poodle.pdf)) predecessor, SSL. TLS and SSL are often used synonymously since TLS is simply a newer version of SSL. TLS was originally designed by Netscape in 1993 for the HTTP use case and when compared with SSH, results in two main differences:

1. TLS only supports [X.509](http://en.wikipedia.org/wiki/X.509) certificate based authentication, which relies on the global [**P**ublic **K**ey **I**nfrastructor](http://en.wikipedia.org/wiki/Public_key_infrastructure). This is most likely due to operating systems and browsers shipping with pre-installed certificates from large corporations such as Verizon and Microsoft, in effect, providing a pre-existing PKI.
1. TLS only tunnels the existing TCP connection, so in order to create multiple data channels we must create multiple TCP connections or create our own internal protocol. At the time, this was probably not seen as an issue since the HTTP paradigm is "request and response" - where the client may only request a single unique resource and the server may only respond once to each request. HTTP/1.1 slightly mitigates this problem with pipelining, though for various reasons it's rarely used.

# A new optimisation target: The network

A large set of the world's networked applications are public facing web applications which, by definition, have an HTTP based API. These applications suit the TLS use case perfectly when our clients are web browsers. However, as we scale up these web applications, supporting more and more concurrent users, we enter the field of [Distributed computing](http://en.wikipedia.org/wiki/Distributed_computing) and our clients become our other servers.

An example web application might have modules for image processing, authentication and storage, residing in a single executable on a single server. If we wish to move this example into a distributed setting, we might split out each module into its own application, each of which could reside across a large number of "worker" servers. This can vastly improve capacity as the workload can be *distributed* out amongst our servers. Although the concept is relatively easy to grasp - it's difficult to implement.

When distributing a system, the network becomes the main concern. Where a procedure call would have previously returned results in nano seconds, a **remote** procedure call now returns results in milliseconds – one million times slower. This new performance concern produces a real need for optimising the network portion of our applications. Today, we generally use HTTPS (HTTP+TLS) to secure internal network traffic, and below I propose the use of SSH in its place – for simpler key management, less reliance on HTTP and an improved connection abstraction.

---

*For those wanting to learn more about distributed systems, I recommend starting here – 
[Distributed systems theory for the distributed systems engineer](http://the-paper-trail.org/blog/distributed-systems-theory-for-the-distributed-systems-engineer/).*

---

Mobile computing is another important entrant onto the modern web ecosystem. Where it would be common for a distributed application to be low latency and high bandwidth connections, it is equally common for mobile to be the reverse. In a world where [a extra seconds of page load time can cost companies thousands of dollars per year](https://blog.kissmetrics.com/loading-time/), we should see the network as an important optimisation target.

<!--
Although SSH can only be used in native applications, the issue of load times is still relevant. This issue is alleviated somewhat by SSH, for example, we can save on TCP round trip times since we only need one connection and instead of using text based protocols (like HTTP) we can use succinct binary protocols (like MsgPack).
-->

# Simpler key management

The X.509 standard defines the operation of today's PKI. This standard is complex, and some feel that it is fundamentally flawed. In short, X.509 requires Certificate authorities; CAs have supreme power; and corruption or negligence at the root (or even intermediate) level leads to global insecurity. Although a private key infrastructure would be easier to manage than the public one, we can create a simpler system using SSH.

SSH has four methods of authentication: password, keyboard challenge (e.g. a series of questions), public key (servers maintain a list of allowed clients), and host based (client and server stores host-specific information). In the following proposed authentication system, we will use SSH's public key method.

<!--
Also note, SSH key pairs are simple binary blobs, whereas X.509 certificates contain much more information.
-->

Let's design a simple authentication system. Imagine we have a distributed system made up of services. There is always one service per host, and a central discovery service which stores host information: ID, service type, IP address and public key. Each server hard-codes the discovery service's IP address and public key. At deploy time, the deploy machine generates a key pair for the new service and securely adds the new host to the the discovery service. Private keys are never shared. When a service receives a connection for first time, it compares the source IP address and received public key with the information from the discovery service, and accepts or rejects it. And that is it.

In practice, not all use cases will match this example perfectly, aspects may be added or removed where necessary. The important lesson here is the system's simplicity.

Note that a few tasks have been left as an exercise for the reader. For example, securely deploying new hosts to the discovery service *(possibly solved by hard-coding the deployment machine's public key into the discovery service)* and scaling and providing redundancy for the discovery service *(possibly solved by a distributed consensus algorithm like [Raft](https://raftconsensus.github.io/) - though with a mostly static environment and a long cache times, lookups would be rare and distributed consensus may be overkill)*.

# HTTP vs SSH

We often choose HTTP for networked applications out of habit and this common choice most likely started due to HTTP's ubiquity. Previously, this ubiquity was important for improved compatibility with clients. Today, however, there are many cases where we also control the clients, and in these cases, we should choose the best tool for the job. Many might assume the following HTTP constructs are required: `Authentication` headers that assist with our authentication systems, URL paths that allow us to route **requests** and cache headers that prevent us from asking for the same data twice. Below, I'll attempt to dispel these assumptions.

**HTTP logins vs SSH authentication**

Although TLS provides the means to perform certificate based authentication, we commonly use `Cookie` header (session tokens) or `Authorization` header (API tokens) for authentication. These artifacts from browser logins provide a less effective authentication scheme for our server environments. Instead of reinventing the authentication wheel for every new application, we can use the built-in primitives provided by SSH.

---

*For those interested in authentication on the web, checkout [Revolutionizing Website Login and Authentication with SQRL](http://vimeo.com/112444120) and if you're interested further, you can also read [the documentation](https://www.grc.com/sqrl/sqrl.htm). It'd be great to shed more light on SQRL, although it's not the perfect solution, it's a huge improvement on today's password ecosystem.*

---

**Pull vs Push**

Since HTTP is based around the request-response message pattern, HTTP clients can only request for information from servers. That is, servers may only send data after a client has asked for it. Therefore, if the client wants real-time information, the client would have to poll for new information at a given interval. This polling is suboptimal, and servers would alternatively like to be able to push new information to the client as it arrives. This ability to push to the client introduces the publish-subscribe message pattern, where a client asks the server to be notified of new information of a particular type. Therefore, the server will send new information **as it arrives**, no polling is required.

The lack of publish-subscribe in HTTP is generally solved with WebSockets. A WebSocket is obtained by "upgrading" an HTTP connection back down to (essentially) a TCP connection, which only has one channel and includes the HTTP overhead. So, in non-browser scenarios, it's clear that using `TCP > TLS > HTTP > WebSockets` is inferior to simply using `TCP > SSH`.

**HTTP paths vs SSH channels**

When network programming with TCP, TLS or WebSockets, we often need some mechanism for multiplexing (routing) messages. We commonly solve this problem with HTTP and URL paths or with channel ID tags on each message. With SSH, however, we are given logical channels by protocol itself. If we view an SSH channel as a single request, we can see it has a URL path (channel type), optional headers (channel data) followed by the request body (the channel's connection stream). SSH has the added benefit that the connection stream is fully duplex, it's not limited to a single request-response, and SSH also has out-of-band requests which could be used for all sorts of meta data use cases (for example, we could send a `content-type` request with data `gzip+json` to change the data encoding).

**HTTP+JSON vs SSH+Binary**

This section isn't a real comparison since we can use any data encoding scheme with any transport protocol. However, it's mentioned here because we generally use JSON, even though there may be a better choice. For example, many messaging libraries use faster binary encoding schemes since message encoding makes up such a large portion of their runtime. So, if your application performs a lot of message encoding, please consider using [Gob](http://golang.org/pkg/encoding/gob/), [MsgPack](https://github.com/ugorji/go), [Cap'nProto](http://kentonv.github.io/capnproto/), etc.

# A small SSH daemon written in Go

Upon learning about Go's [crypto/ssh package](http://golang.org/x/crypto/ssh), I thought I'd try my hand at writing a primitive [`sshd(8)`](http://www.openbsd.org/cgi-bin/man.cgi/OpenBSD-current/man8/sshd.8?query=sshd&sec=8) using password authentication. Below, I'll go through the code and comment on my findings.

First, let's create our SSH server configuration. In the latest version of crypto/ssh (after Go 1.3), the SSH server type has been removed in favour of an SSH connection type. We upgrade a `net.Conn` into an `ssh.ServerConn` by passing it along with a `ssh.ServerConfig` to `ssh.NewServerConn`.

In this example, we'll only support password authentication and we'll hard-code the username and password to be `foo` and `bar`

``` go
config := &ssh.ServerConfig{
	PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
		if c.User() == "foo" && string(pass) == "bar" {
			return nil, nil
		}
		return nil, fmt.Errorf("password rejected for %q", c.User())
	},
}
```

We'll add the server's private key into the `config`. If we don't have one yet, we can generate a key pair with `ssh-keygen -t rsa -C "server@company.com"`. Keep the private key safe at all costs!

``` go
privateBytes, err := ioutil.ReadFile("id_rsa")
if err != nil {
	log.Fatal("Failed to load private key (./id_rsa)")
}

private, err := ssh.ParsePrivateKey(privateBytes)
if err != nil {
	log.Fatal("Failed to parse private key")
}

config.AddHostKey(private)
```

We'll bind a `net.Server` and `Accept()` incoming `net.Conn`s

``` go
listener, err := net.Listen("tcp", "0.0.0.0:2200")
if err != nil {
	log.Fatalf("Failed to listen on 2200 (%s)", err)
}

log.Print("Listening on 2200...")
for {
	conn, err := listener.Accept()
	if err != nil {
		log.Printf("Failed to accept incoming connection (%s)", err)
		continue
	}

```

Once accepted, we'll upgrade each `net.Conn` into an `ssh.ServerConn`

``` go
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("Failed to handshake (%s)", err)
		continue
	}

	// Discard all global out-of-band Requests
	go ssh.DiscardRequests(reqs)
	// Accept all channels
	go handleChannels(chans)
}
```

To prevent the program from blocking when we receive a new channel request, we'll pass it straight off to `handleChannel`


``` go
func handleChannels(chans <-chan ssh.NewChannel) {
	for newChannel := range chans {
		go handleChannel(newChannel)
	}
}
```

Before we accept this new channel, we'll ensure it's a terminal session by confirming the channel type is `session`

``` go
func handleChannel(newChannel ssh.NewChannel) {
	if t := newChannel.ChannelType(); t != "session" {
		newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
		return
	}
```

Upon accepting the new channel, we'll receive the channel's `connection` and `requests` queue. As shown on the diagram above, the TCP connection and each of the channel `connection`s are all `net.Conn`s. In this particular case, the `connection` is a duplex data stream to the user's terminal screen.

``` go
	connection, requests, err := newChannel.Accept()
	if err != nil {
		log.Printf("Could not accept channel (%s)", err)
		return
	}
```

Next, we'll fire up `bash` for this session and prepare a `close` function to be called at the end of the session or in the event an error

``` go
	bash := exec.Command("bash")

	// Prepare teardown function
	close := func() {
		connection.Close()
		_, err := bash.Process.Wait()
		if err != nil {
			log.Printf("Failed to exit bash (%s)", err)
		}
		log.Printf("Session closed")
	}
```

At this point, we could simply pipe the inputs and outputs of `bash` to the terminal screen, though this would fail since the `bash` command knows that it's not running inside a [tty](http://en.wikipedia.org/wiki/Teleprinter). In order to trick `bash` into thinking that it is, we'll need to wrap the command in a [pty](http://en.wikipedia.org/wiki/Pseudo_terminal) (*since we don't have real teletypes anymore, our software terminals emulate them - yielding pseudo teletypes*)

``` go
	bashf, err := pty.Start(bash)
	if err != nil {
		log.Printf("Could not start pty (%s)", err)
		close()
		return
	}
```

Now, when we pipe `bash` and the `connection` together, `bash` correctly assumes it may use [terminal commands](http://wiki.bash-hackers.org/scripting/terminalcodes) to control our `ssh(1)` client. Also, we're using `sync.Once` to ensure our `close` function is only called once.

``` go
	var once sync.Once
	go func() {
		io.Copy(connection, bashf)
		once.Do(close)
	}()
	go func() {
		io.Copy(bashf, connection)
		once.Do(close)
	}()
```

Finally, we **must** reply positively to the `shell` and `pty-req` channel requests in order to instruct `ssh(1)` that it has access to a shell and that it is also a pseudo-teletype. We will also listen for `window-change` events and pass on these updates to `bashf` (our `bash`-wrapped pty).

*As a side note, since channel types and request types are strings, this should tell us they are dynamic. So, in our own applications, we could re-purpose these protocol constructs to suit our own needs as necessary.*

``` go
	go func() {
		for req := range requests {
			switch req.Type {
			case "shell":
				// We only accept the default shell
				// (i.e. no command in the Payload)
				if len(req.Payload) == 0 {
					req.Reply(true, nil)
				}
			case "pty-req":
				termLen := req.Payload[3]
				w, h := parseDims(req.Payload[termLen+4:])
				SetWinsize(bashf.Fd(), w, h)
				req.Reply(true, nil)
			case "window-change":
				w, h := parseDims(req.Payload)
				SetWinsize(bashf.Fd(), w, h)
			}
		}
	}()
}
```

And that's it. The remaining minor portions of this example have been omitted,
such as comments, imports and helper functions.
You can find the complete version of this example here:

**[https://github.com/jpillora/go-and-ssh](https://github.com/jpillora/go-and-ssh/blob/master/sshd/server.go)**

When it's ready, run the server in one window 
and connect to it in another. We should see:

``` plain
$ go run sshd.go
2014/12/26 13:05:55 Listening on 2200...
```

``` plain
$ ssh foo@localhost -p 2200
foo@localhost's password:
bash-4.3$
```

``` plain
2014/12/26 13:06:28 New SSH connection from [::1]:50261 (SSH-2.0-OpenSSH_6.2)
2014/12/26 13:06:28 Creating pty...
```

``` plain
bash-4.3$ date
Fri 26 Dec 2014 13:06:30 AEDT
bash-4.3$ top
# ... fullscreen view of top - try it and see!
```

# A terminal version of Tron written in Go

Last year, I wrote a terminal version of the classic arcade game [Tron](http://www.classicgamesarcade.com/game/21670/tron-game.html) (Light Cycles) over `telnet`. After learning about SSH however, I decided to change the backend to SSH for improved client compatibility and for the various terminal commands built into SSH. You can learn more about it and give it a try here:

**https://github.com/jpillora/ssh-tron**

[![tron](https://rawgit.com/jpillora/ssh-tron/master/demo.gif)](https://github.com/jpillora/ssh-tron)

# Conclusion

HTTP has served us well since 1991, however the web has come a long way since then and there is much room for improvement. [SPDY](http://en.wikipedia.org/wiki/SPDY) is one such improvement which sits in-between TCP and HTTP. SDPY has been used as the basis for [HTTP/2](http://en.wikipedia.org/wiki/HTTP/2) (though some still [aren't happy](https://queue.acm.org/detail.cfm?id=2716278) with it). I'm looking forward to [QUIC](http://en.wikipedia.org/wiki/QUIC) as a superior alternative to TCP, TLS and SSH ([*QUIC: next generation multiplexed transport over UDP*](https://www.youtube.com/watch?v=hQZ-0mXFmk8)). For now, though, if you're stuck with HTTP (writing for browser clients), I would recommend Gzip+JSON over HTTPS for a nice balance of compatibility and network performance. Finally, if your new project targets an internal server environment, native desktop clients, or native mobile clients, give [crypto/ssh](https://golang.org/x/crypto/ssh) a try.

I'll end with a disclaimer: although this post advocates for SSH over TLS+HTTP, the benefits described may not always outweigh the benefits of HTTP. **Beware of premature optimisation**. **Profile accordingly**.

Please send corrections and suggestions to [this issues page](https://github.com/jpillora/gopheracademy-web/issues). You can follow me on [Github](https://github.com/jpillora) and [Twitter](https://twitter.com/jpillora).
