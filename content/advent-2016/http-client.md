+++
author = [
  "Kevin Burke",
]
date = "2016-11-06T06:40:42Z"
linktitle = "Writing an API Client in Go"
title = "Writing an API Client in Go"
series = ["Advent 2016"]

+++

Let's say you need to write a client that talks to a third party API, like the
AWS API, or the Twilio API. Go gives you a lot of tools that can help you write
a really good client, but you have to know how to take advantage of them! Keep
reading for tips that will help you write a great API client.

## Contexts and Timeouts

Generally your users will give up waiting for an answer after some amount of
time. As an extreme example, if your checkout flow takes 2 days to return a
response, your users will probably give up and buy it at Target. 2 days is
an extreme example, but there is some maximum amount of time your users (and
your code) should wait for an answer before you should give up and execute the
fallback logic (like telling users they should try again later).

Complicating this, the user that's calling your library is probably doing so
as part of a larger request; maybe they are making 10 simultaneous requests to
your client, and also making a database request and also doing some filesystem
work. Odds are they want to enforce some deadline for all of that work to
finish, and if the work isn't complete by then, everything still in progress
should be canceled.

Go's [`context` library][context] is perfect for this use case. Users can
create a Context, pass it to multiple threads, and then either cancel the work
being done in every thread, or time it out after a specific deadline.

[context]: https://golang.org/pkg/context/

Many other languages make it tricky to enforce an absolute deadline on a
HTTP request - in many languages, you compute the timeout as a duration
("3 seconds") and that [can reset any time the server sends a single
byte][byte]. You can use `context.WithDeadline` to [enforce an absolute
deadline][with-deadline] on a client request, which is really nice.

[byte]: https://kev.inburke.com/slides/reliable-http/#connect-timeout-requests
[with-deadline]: https://golang.org/pkg/context/#WithDeadline

The best practice is to pass a Context as the first parameter to every function
that can open a socket. Here are some example function signatures:

```go
client.Messages.Create(ctx context.Context, to, from string)
client.Emails.Get(ctx context.Context, sid string)
client.Notifications.List(ctx context.Context, filters url.Values)
```

All you have to do inside your library is pass the Context to the http.Request
object via `request.WithContext()`:

```go
req, err := http.NewRequest(method, url, body)
req = req.WithContext(ctx)
return client.Do(req)
```

This will let your users coordinate timeouts very precisely, as well as cancel
requests they no longer need.

## Type Parsing

[GRPC][grpc] API's are becoming more common, but most HTTP API's you'll deal
with are still returning XML or JSON data. JSON offers only a few types -
numbers, strings, booleans, and arrays/maps of those. Go offers a much wider
range of types. Consider trying to marshal those JSON/XML objects into a more
useful type for your users.

For example, the Twilio API returns phone numbers as strings - `"+14105551234"`
for example. We can parse those into PhoneNumber objects, and then provide
helpers to let users print different formats of the number (e.g. `"(410)
555-1234"`).

```go
type PhoneNumber string

func (p PhoneNumber) Local() string {
    num, err := libphonenumber.Parse(string(pn), "US")
    if err != nil {
        return string(pn)
    }
    return libphonenumber.Format(num, libphonenumber.NATIONAL)
}

// A Call represents a phone call.
type Call struct {
    To          PhoneNumber `json:"to"`
    From        PhoneNumber `json:"from"`
    ID          string      `json:"id"`
    DateCreated time.Time   `json:"date_created"`
}
```

[grpc]: http://www.grpc.io/about/

#### Nullable Values

Frequently API's written in other languages will return values that are
nullable or don't contain the right type. For example, an API may return either
`null` or a `time.Time` for a field, or may return booleans as strings, e.g.
"true" and "false". I like to borrow a type from the `database/sql` package to
handle nullable types.

```go
type NullTime struct {
    Time time.Time
    Valid bool
}
```

Callers can check whether a NullTime is Valid; if so, they can access the Time
value, otherwise it's zero.

Of course, this [needs to be marshaled][marshal-time] from JSON into the right
value, which we can accomplish by satisfying the json.Unmarshaler interface.

```go
func (nt *NullTime) UnmarshalJSON(b []byte) error {
    if string(b) == "null" {
        nt.Valid = false
        return nil
    }
    var t time.Time
    err := json.Unmarshal(b, &t)
    if err != nil {
        return err
    }
    nt.Valid = true
    nt.Time = t
    return nil
}
```

[marshal-time]: https://github.com/kevinburke/go-types/blob/master/null_time.go#L22

## User Agents

Sometimes clients have faulty logic. In these cases it's extremely useful for
the server to know which version of the client is making the request. The
server can use this to email accounts with faulty clients and ask them to
upgrade, or (gasp) return different results to different clients, if upgrading
is impossible.

I recommend including the following information in your library:

- the version number
- the name of the client
- the name/version of the HTTP or REST client you are using, if it's not just
  net/http
- the Go platform version

Here is a sample User-Agent string for my [twilio-go helper library][twilio-go]:

```
twilio-go/0.54 rest-client/0.16 (https://github.com/kevinburke/rest) go1.7.4 (darwin/amd64)
```

You can add it to outbound requests with req.Header.Add():

```go
req, err := http.NewRequest(method, url, body)
req.Header.Add("User-Agent", "twilio-go/0.54 ...")
return client.Do(req)
```

## Forward Compatibility

Users of your client library might not be able to upgrade to a newer version
(or may be worried about introducing incompatibilities by doing so). Where
possible, it's good to try to be forward compatible in your client library. For
example, if the server offers a new parameter, or changes the available types
for an existing parameter, users should be able to specify those without
needing to pull down the latest versions.

Specifying API parameters with a `url.Values` works really well for this use
case. For example:

```go
data := url.Values{}
data.Set("To", "+14105551234")
data.Set("From", "+14105556789")
client.Calls.Create(data)
```

If the server decides to allow calls to people's names instead of phone
numbers, or to the number 7, or allow multiple From values, or a new parameter,
your users are totally compatible with their existing code! They can just
change the values they set on `data` and they are good to go.

## Usage Patterns

For most client libraries, the vast majority of your users will only do one
or two things with the API. For Stripe this is charging a credit card, for
Sendgrid this is sending an email, &c, &c. Offer helpers to make common actions
really easy. For example, this type of interface is easy to scale to many
resources and many different HTTP methods:

```go
data := url.Values{}
data.Set("From", "boss@example.org")
data.Set("To", "foo@example.com")
data.Set("Subject", "TPS Cover Sheets")
data.Set("Body", "They are important!")
return client.V1.Emails.CreateResource(data)
```

But it's a little cumbersome. It might be worthwhile to add a helper function
that simplifies the interface a little bit.

```go
client.SendEmail("boss@example.org", "foo@example.com", "TPS Cover Sheets",
    "They are important! Don't forget them.")
```

## Testing

One way to test your client would be to define each resource as an interface,
then add dummy code that satisfies the interface and returns you objects. For
example:

```go
type ChargeResource interface {
    Get(string) (*Charge, error)
    Create(url.Values) (*Charge, error)
    List(url.Values) ([]*Charge, error)
}
```

Then your tests would integrate with dummy versions of each interface. The
problem here is that you're not actually testing against the HTTP response. If
you make a change to the client code that parses the HTTP response incorrectly,
you're not going to catch it.

The next option is to integrate directly with the API - pass in a valid set of
credentials and make network requests. This is the only way to tell if the API
starts returning different responses, but is slow, doesn't work on the subway,
and may be expensive, if you are testing API calls that cost money.

The third option is to save the API response, then spin up a test server
that serves that response on demand. This gets you most of the benefits of
integration with the API, but is much faster, works on the subway and won't
cost you anything to test expensive calls. Spinning up a test server is easy
and cheap in Go. Here's an example test:

```go
s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(400)
    w.Write([]byte(`{"message": "Card charge denied", "code": 10002}`))
}))
defer s.Close()
client.Base = s.URL
charge, err := client.Charge.Create(...)
if err == nil {
    t.Fatal("Expected to get error...")
}
```

This is really cheap and you should be able to run all of these tests in
parallel, which will also help speed up your test suite.

That's it! Best of luck.

[twilio-go]: https://github.com/kevinburke/twilio-go
