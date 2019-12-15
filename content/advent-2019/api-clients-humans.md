+++
author = ["Josh Michielsen"]
title = "API Clients for Humans"
linktitle = "API Clients for Humans)"
date = 2019-12-14T00:00:00Z
+++

# API Clients for Humans

Most developers, at one point or another, have either built a web API or have been a consumer of one. An API client is a package that provides a set of tools that can be used to develop software that consumes a specific API. These API clients, sometimes also referred to as a Client SDK, make it easier for consumers to integrate with your service. 

API clients are themselves also APIs, and as such it is important to consider the user experience when designing and building them. This post discusses a variety of best practices for building API clients with a focus on delivering a great user experience. Topics that will be covered include object and method design, error handling, and configuration.

## Client Initialisation & Configuration

Lets start by looking at a very basic API client for a web API. This API allows us to do basic CRUD operations on users and groups. The below example shows a client that allows us to create a new user:

```go
package myclient

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type Client struct {
	Client *http.Client
}

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (c *Client) CreateUser(name string) (*User, error) {
	data, err := json.Marshal(map[string]string{
		"name": name,
	})
	if err != nil {
		return nil, err
	}

	resp, err := c.Client.Post("https://api.exmaple.com/users", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var user User
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
```

Out client has very little configuration requirements and can easily be instantiated - simply requiring the user to pass a `http.Client`. Lets look at a brief example of a user consuming your package:

```go
package main

import (
  "net/http"
  "log"
  
  "example.com/myclient"
)

func main() {
  client := myclient.Client{
    Client: &http.Client{},
  }
  
  _, err := client.CreateUser("Boaty McBoatface")
  if err != nil {
    log.Fatalln(err)
  }
}
```

Pretty simple right? However, there are two issues with this approach. 

1. Right now our user only has to provide a `http.Client` to use our client, but we may want to add additional options as we increase it's complexity. Currently we have no way of providing sane defaults for our client package.
2. As we begin to add more options to our client, this is going to cause breaking changes to our existing users.

To deal with the first issue we should provide users a way to create an instance of our client without requiring them to "manually" create the object. We can do this by providing a `NewClient()` function:

```go
package myclient

...

func NewClient() *Client {
  return &Client{
    BaseURL: "api.example.com",
    Client:  &http.Client{},
  }
}
```

As you can see - this function now allows us to set default values on our client (such as a `BaseURL`). However, in addition to still having the second issue above to deal with, we've introduced a new issue - how do our consumer change the defaults if they want to?

There are a few ways to solve this, but the one I want to focus on is something called "Functional Configuration". To keep this post from getting too long I'm not going to take you through all the alternatives, and their potential issues - rather I will point you towards [this great post by Dave Cheney](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis).

Lets take a look at an example of our client with functional options implemeted:

```go
package myclient

import (
	"net/http"
	"time"
)

type Client struct {
	APIKey     string
	BaseURL    string
	httpClient *http.Client
}

func NewClient(apiKey string, opts ...func(*Client) error) (*Client, error) {
	client := &Client{
		APIKey:     apiKey,
		BaseURL:    "api.example.com",
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	for _, opt := range opts {
		err := opt(client)
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

// WithHTTPClient allows users of our API client to override the default HTTP Client!
func WithHTTPClient(client *http.Client) func(*Client) error {
  return func(c *Client) error {
    c.httpClient = client
    return nil
  }
}
```

We now have a mechanism to provide consumers of our package with sane defaults, while also allowing them to override those default values.

## Service Objects

In a lot of cases your REST API is going to end up with endpoints that pertain to different resources. For example a banking API might have users, accounts, payments, etc. Each of these resources will support difference HTTP methods (GET, POST, PUT, etc). An API client that supports all these methods against those resources can quickly become difficult to manage, with a large number of possible functions available to consumers:

```
- client.GetUser()
- client.CreateUser()
- client.ListAccounts()
- client.GetAccounts()
...
```

Service objects are a pattern for separating these resources that make it easier for consumers of your client package to discover and utilise the features of your client. Lets look at a basic example of what service objects look like (note: for the sake of brevity this example doesn't follow best practices for error handling, and lacks imports):

_client.go_

```go
package myclient

type Client struct {
  httpClient *http.Client
  
  Users      *UserService
  Accounts   *AccountService
}

func NewClient() *Client {
  c := &Client{
    httpClient: &http.Client{},
  }
  
  c.Users = &UserService{client: c}
  c.Accounts = &AccountService{client: c}
  
  return c
}
```

_users.go_

```go
package myclient

type UserService struct {
  client *Client
}

type User struct {
  ID   int `json:"id"`
  Name string `json"name"`
}

func (u *UserService) Get(name string) *User {
  resp, _ := u.client.httpClient.Get("api.example.com/users/" + name)
  
  defer resp.Body.Close()

	var user User
	_ = json.NewDecoder(resp.Body).Decode(&user)
	
  return &user
}
```

A consumer using our client would now call `client.Users.Get("Boaty McBoatface")` rather than `client.GetUser("Boaty McBoatface")`. When initialising the various "services" within our package we provide the original client object, which gives our services access to both the configuration of the client (e.g. `BaseURL`) and the `http.Client` so we can reuse the same client for each outgoing call (note: `http.Client` is safe for concurrency).

Some popular client packages that utilise this pattern are [twilio-go](https://github.com/kevinburke/twilio-go) and [github-go](https://github.com/google/go-github).

## Error Handling

A fundemental aspect of writing idiomatic Go is to return errors back to the caller. To make error handling easier for your consumers you should consider creating custom error types for common errors. 

For example, if your API returns a `404`, rather than returning `fmt.Errorf("API returned an error: %v", resp.StatusCode)` create and return a custom error type such as `ErrUserNotFoundError`. Provided you document this response, your users can then check the error type with `error.Is()` or by casting the error to the custom type. This provides a more consistent experience for your consumers.

## Conclusion

This post has attempted to provide some simple ways you can enhance the user experience for consumers of your API clients. This is by no means an exhaustive list, but will hopefully provide you with a good starting place the next time you sit down to write an API client package.