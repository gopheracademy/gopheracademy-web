+++
author = ["Jiahua Chen"]
date = "2015-12-18T04:07:33-05:00"
title = "Writing SSH server in Go"
series = ["Advent 2015"]
+++

When I'm working on [Gogs](https://gogs.io) project, there is a need of builtin
SSH server, which allows users to preform Git-only operations through key-based
authentication.

All available resources on the web are all minimal examples and does not fit this
specific requirement. Therefore, I think it's worth sharing my experiences to make
your life easier in case you just ran into same problem as mine.

The code structure is pretty much same to the examples you can find on the web.

1. Start a SSH listening host.
2. Accept new requests and validate their public key with database.
3. Preform Git operations.
4. The most important part, return a status if no error occurs.

OK, before we get started, just note that code are not supposed to be copy-paste
and just work. It will make this post too long if involves all the details.

### Prepare to start a SSH server

The server must have a private key in order to start a SSH server. This is for the
purpose of preventing [Man-in-the-middle attack](https://en.wikipedia.org/wiki/Man-in-the-middle_attack).

This key does not need to be server-wide, just keep it somewhere but not in temporary
directory because users will add this key to their `known_hosts` file.

```go
// Listen starts a SSH server listens on given port.
func Listen(port int) {
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			pkey, err := models.SearchPublicKeyByContent(strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))))
			if err != nil {
                // handle error
				return nil, err
			}
			return &ssh.Permissions{Extensions: map[string]string{"key-id": com.ToStr(pkey.ID)}}, nil
		},
	}

	keyPath := filepath.Join(setting.AppDataPath, "ssh/gogs.rsa")
	if !com.IsExist(keyPath) {
		os.MkdirAll(filepath.Dir(keyPath), os.ModePerm)
		_, stderr, err := com.ExecCmd("ssh-keygen", "-f", keyPath, "-t", "rsa", "-N", "")
		if err != nil {
			panic(fmt.Sprintf("Fail to generate private key: %v - %s", err, stderr))
		}
		log.Trace("New private key is generateed: %s", keyPath)
	}

	privateBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		panic("Fail to load private key")
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		panic("Fail to parse private key")
	}
	config.AddHostKey(private)

	go listen(config, port)
}
```

This piece of code does three things:

1. Setup a callback for validating public key from database.

    Function `ssh.MarshalAuthorizedKey` will return a string format of user's public
    key with a line break, so we want to remove that by calling `strings.TrimSpace`,
    and then search in the database.

    After search, if we return any kind of error, it will produce `Permission denied`
    prompt on user side. If no error is returned, you can carry an instance of type
    `*ssh.Permissions` to the corresponding request handler.

    In this case, we need to set which key ID is this request corresponding to in `Extensions`.

2. Create a private key when there is no one exists.

    This is done by calling a command `ssh-keygen -f keypath -t rsa -N ""`.

3. Load private key and start listening on given port.

### Start listening and accepting new requests

Like normal HTTP server, an SSH server needs to listen on a specific port as well.

The pattern is very similar:

```go
func listen(config *ssh.ServerConfig, port int) {
	listener, err := net.Listen("tcp", "0.0.0.0:"+com.ToStr(port))
	if err != nil {
		panic(err)
	}
	for {
		// Once a ServerConfig has been configured, connections can be accepted.
		conn, err := listener.Accept()
		if err != nil {
            // handle error
			continue
		}
		// Before use, a handshake must be performed on the incoming net.Conn.
		sConn, chans, reqs, err := ssh.NewServerConn(conn, config)
		if err != nil {
            // handle error
			continue
		}

		// The incoming Request channel must be serviced.
		go ssh.DiscardRequests(reqs)
		go handleServerConn(sConn.Permissions.Extensions["key-id"], chans)
	}
}
```

1. Accept requests inside a infinite `for` loop.
2. Preform handshakes for new SSH connections.
3. Discard all irrelevant incoming request but serve the one you really need to care.

    At this point, you can see we use `Extensions` to pass the user's public key ID
    in the database.

### Handle connections

Finally, we're going to really serve the SSH requests.

```go
func handleServerConn(keyID string, chans <-chan ssh.NewChannel) {
	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			newChan.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		ch, reqs, err := newChan.Accept()
		if err != nil {
            // handle error
			continue
		}

		go func(in <-chan *ssh.Request) {
			defer ch.Close()
			for req := range in {
				payload := cleanCommand(string(req.Payload))
				switch req.Type {
				case "exec":
					cmdName := strings.TrimLeft(payload, "'()")

					args := []string{"serv", "key-" + keyID, "--config=" + setting.CustomConf}
					cmd := exec.Command(setting.AppPath, args...)

					stdout, err := cmd.StdoutPipe()
					if err != nil {
						// handle error
						return
					}
					stderr, err := cmd.StderrPipe()
					if err != nil {
						// handle error
						return
					}
					input, err := cmd.StdinPipe()
					if err != nil {
						// handle error
						return
					}

					if err = cmd.Start(); err != nil {
						// handle error
						return
					}

					go io.Copy(input, ch)
					io.Copy(ch, stdout)
					io.Copy(ch.Stderr(), stderr)

					if err = cmd.Wait(); err != nil {
						// handle error
						return
					}

					ch.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
					return
				default:
				}
			}
		}(reqs)
	}
}
```

It is possible to have more than one channel inside one connection, so we need to loop
over all of them.

Then, we need to make sure that it is a `session` type channel, otherwise that's useless
for performing Git operations (or other operations in general).

Next step, we need to accept requests from current channel, and serve them in separate
goroutines so the connection won't be blocked.

Finally, we're getting into the most interesting part.

1. There could be more than one request from single channel, we need to handle each
of them.
2. The payload comes from request somehow is not always in a clean format, so we
have to preform a clean operation to remove unless characters:

    ```go
func cleanCommand(cmd string) string {
	i := strings.Index(cmd, "git")
	if i == -1 {
		return cmd
	}
	return cmd[i:]
}
    ```

3. Check the type of request, the `exec` type is what we're looking for.
4. Clean payload again for strange characters, and call a specific command that
handles Git operations.
5. We need to get all of three pipelines before actually start executing the command:
`StdoutPipe`, `StderrPipe` and `StdinPipe`.
6. Note that we have to put input pipeline in a goroutine because Git needs to write
content after it receives information from server.

**The most most most important thing at the end, is you must must must send a
`exit-status` back to Git client side**, otherwise, it just hangs forever.

This is the problem I'd been stuck for six months until someday someone somehow mentioned.

You can find complete code at [SSH module](https://github.com/gogits/gogs/blob/master/modules%2Fssh%2Fssh.go) file. Hope it helps you as well.
