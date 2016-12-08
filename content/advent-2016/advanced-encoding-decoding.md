+++
linktitle = "Advanced Encoding and Decoding Techniques in Go"
date = "TBD"
author = ["Jon Calhoun"]
title = "Advanced Encoding and Decoding Techniques in Go"
series = ["Advent 2016"]
+++


# Advanced Encoding and Decoding Techniques

Go's standard library comes packed with some great encoding and decoding packages covering a wide array of encoding schemes. Everything from CSV, XML, JSON, and even gob - a Go specific encoding format - is covered, and all of these packages are incredibly easy to get started with. In fact, most of them don't require you to add any code at all; You simply plug in your data and it spits out encoded data.

Unfortunately, not all applications have the pleasure of working with data that maps one-to-one to its JSON representation. Struct tags help cover most of these use cases, but if you work with enough APIs you are bound to eventually come across a case where it just isn't enough. 

For example, you might run into an API that outputs different objects to the same key, making it a prime candidate for generics, but Go doesn't have those. Or you might be using an API that accepts and returns [Unix time](https://en.wikipedia.org/wiki/Unix_time) instead of RFC 3339; Sure, we could leave it as an int in our code, but it would be much nicer if we could work directly with the [time](https://golang.org/pkg/time/) package's [Time](https://golang.org/pkg/time/#Time) type.

In this post we are going to review a few techniques that help turn what might seem like a troublesome encoding problem into some fairly easy to follow code. We will be doing this using the [encoding/json](https://golang.org/pkg/encoding/json/) package, but it is worth noting that Go provides a `Marshaler` and `Unmarshaler` interface for most encoding types, allowing you to customize how your data is encoded and decoded in multiple encoding schemes.


## The always-faithful new type approach

The first technique we are going to examine is to create a new type and convert our data to/from that type before encoding and decoding it. While this isn't technically an encoding specific solution, it is one that works reliably, and is fairly easy to follow. It is also a basic technique that we will be building upon in the next few sections, so it is worth taking some time to look at.

Let's imagine that our application is starting off with a simple `Dog` type like below.

```go
type Dog struct {
  ID      int
  Name    string
  Breed   string
  BornAt  time.Time
}
```

By default, the [time.Time](https://golang.org/pkg/time/#Time) type will be marshaled as RFC 3339. That is, it will turn into a string that looks something like `2016-12-07T17:47:35.099008045-05:00`.

While there is nothing particularly wrong with this format, we might have a reason to want to encode and decode this field differently. For example, we might be working with an API that sends us a Unix time and expects the same in return.

Regardless of the reason, we need a way to change how this is turned into JSON, along with how it is parsed from JSON. One approach to solving this problem is to simply create a new type, let's call it the `JSONDog` type, and use that to encode and decode our JSON.

```go
type JSONDog struct {
  ID     int    `json:"id"`
  Name   string `json:"name"`
  Breed  string `json:"breed"`
  BornAt int64  `json:"born_at"`
}
```

Now if we want to encode our original `Dog` type in JSON, all we need to do is convert it into a `JSONDog` and then marshal that using the `encoding/json` package.

Starting with a constructor that takes in a `Dog` and returns a `JSONDog` we get the following code.

```go
func NewJSONDog(dog Dog) JSONDog {
  return JSONDog{
    dog.ID,
    dog.Name,
    dog.Breed,
    dog.BornAt.Unix(),
  }
}
```

Putting that together with the `encoding/json` package we get the following sample.

```go
func main() {
  dog := Dog{1, "bowser", "husky", time.Now()}
  b, err := json.Marshal(NewJSONDog(dog))
  if err != nil {
    panic(err)
  }
  fmt.Println(string(b))
}
```

Decoding a JSON object into a `Dog` works very similar to this. We first start by decoding into a `JSONDog` instance, and then we convert that back to our `Dog` type using a `Dog()` method on the `JSONDog` type.

```go
func (jd JSONDog) Dog() Dog {
  return Dog{
    jd.ID,
    jd.Name,
    jd.Breed,
    time.Unix(jd.BornAt, 0),
  }
}

func main() {
  b := []byte(`{
    "id":1,
    "name":"bowser",
    "breed":"husky",
    "born_at":1480979203}`)
  var jsonDog JSONDog
  json.Unmarshal(b, &jsonDog)
  fmt.Println(jsonDog.Dog())
}
```

*You can see the full code sample and run it on the Go Playground here: <https://play.golang.org/p/0hEhCL0ltW>*

The primary benefit to this approach is that it will always work because we can always build that translation layer. It doesn't matter if the JSON representation looks nothing like our Go code, so long as we have a way to convert between the two.

In addition to always working, the code is incredibly easy to follow. A new teammate could jump right into this code without missing a beat because there isn't any magic happening. We are just converting data between two types.

Despite the benefits of this technique, there are a few cons. The two biggest are:

1. It is easy for a developer to forget to convert a `Dog` into a `JSONDog`, and 
2. It takes a lot of extra code, especially if we have a large struct and only a couple fields need customized

Rather than throw this technique out the window, let's take a look at how to tackle both of these problems without sacrificing much code clarity.


### Implementing the `Marshaler` and `Unmarshaler` interfaces

The first problem we saw with the last approach is that it is pretty easy to forget to convert a `Dog` into a `JSONDog`, so in this section we are going to discuss how you can implement the [Marshaler](https://golang.org/pkg/encoding/json/#Marshaler) and [Unmarshaler](https://golang.org/pkg/encoding/json/#Unmarshaler) interfaces in the `encoding/json` package to make the conversion automatic.

The way these two interfaces work is pretty straight forward; When the `encoding/json` package runs into a type that implements the `Marshaler` interface, it uses that type's `MarshalJSON()` method instead of the default marshaling code to turn the object into JSON. Similarly, when decoding a JSON object it will test to see if the object implements the `Unmarshaler` interface, and if so it will use the `UnmarshalJSON()` method instead of the default unmarshaling behavior. That means that all we need to do to ensure our `Dog` type is always encoded and decoded as a `JSONDog` is to implement these two methods and do the conversion there.

Once again, we are going to start with encoding first, which means we will be implementing the `MarshalJSON() ([]byte, error)` method on our `Dog` type. 

While this might feel like a large undertaking at first, we can actually utilize a lot of our existing code to minimize what we need to write. All we really need to do in this method is return the results from calling `json.Marshal()` on a `JSONDog` representation of our current `Dog`.

```go
func (d Dog) MarshalJSON() ([]byte, error) {
  return json.Marshal(NewJSONDog(d))
}
```

Now it doesn't matter if a developer forgets to convert our `Dog` type to the `JSONDog` type; This will happen by default when the `Dog` is encoded into JSON.

The `Unmarshaler` implementation ends up being pretty similar. We are going to implement the `UnmarshalJSON([]byte) error` method, and once again we are going to utilize our existing `JSONDog` type.

```go
func (d *Dog) UnmarshalJSON(data []byte) error {
  var jd JSONDog
  if err := json.Unmarshal(data, &jd); err != nil {
    return err
  }
  *d = jd.Dog()
  return nil
}
```

Finally, we update our `main()` function to use the `Dog` type for both decoding and encoding instead of the `JSONDog` type.

```go
func main() {
  dog := Dog{1, "bowser", "husky", time.Now()}
  b, err := json.Marshal(dog)
  if err != nil {
    panic(err)
  }
  fmt.Println(string(b))

  b = []byte(`{
    "id":1,
    "name":"bowser",
    "breed":"husky",
    "born_at":1480979203}`)
  dog = Dog{}
  json.Unmarshal(b, &dog)
  fmt.Println(dog)
}
```

*You can find a working sample of the code up until this point on the Go Playground here: <https://play.golang.org/p/GR6ckydMxF>*

With about 10 lines of code we have overridden the default JSON encoding for our `Dog` type. Pretty neat, right?

Next we will tackle the other problem we had with our initial approach using embedded data and an alias type.



### Using embedded data and an alias type to reduce code

*NOTE: The term "alias" here is NOT the same as the alias proposed for Go 1.9. This is simply referring to a new type that has the same data as another type, but has its own set of methods.*

As we saw before, copying all of the data from one type to another just to customize a single field can be pretty tedious. This is amplified even further when we start to work with objects with 10 or 20 fields. Keeping both a `JSONDog` and a `Dog` type in sync is just annoying to do.

Luckily, there is another way to tackle this problem that will reduce the fields we need to customize to just those that need custom encoding and decoding. We are going to embed our `Dog` object inside of our `JSONDog` and customize the fields that need customizing. 

To get started, we first need to update the `Dog` type and add the JSON struct tags back for fields we won't be customizing, and we will tell the `encoding/json` package to ignore fields we will be customizing by using the struct tag `json:"-"` which signifies that the JSON encoder should ignore this field even though it is exported.

```go
type Dog struct {
  ID     int       `json:"id"`
  Name   string    `json:"name"`
  Breed  string    `json:"breed"`
  BornAt time.Time `json:"-"`
}
```

Next, we are going to embed the `Dog` type inside of our `JSONDog` type, and update the `NewJSONDog()` function along with the `Dog()` method on the `JSONDog` type. We will also be temporarily renaming the `Dog()` method to `ToDog()` so it doesn't collide with the embedded `Dog` object.

*Warning: This code will not work just yet, but I am showing this intermediate step to illustrate why it won't work.*

```go
func NewJSONDog(dog Dog) JSONDog {
  return JSONDog{
    dog,
    dog.BornAt.Unix(),
  }
}

type JSONDog struct {
  Dog
  BornAt int64 `json:"born_at"`
}

func (jd JSONDog) ToDog() Dog {
  return Dog{
    jd.Dog.ID,
    jd.Dog.Name,
    jd.Dog.Breed,
    time.Unix(jd.BornAt, 0),
  }
}
```

Unfortunately, this code won't work. It will compile, but if you try to run it you will run into a `fatal error: stack overflow` error. This happens when we call the `MarshalJSON()` method on the `Dog` type. When this happens, it will construct a `JSONDog`, but this object has a nested `Dog` inside of it, which will construct a new `JSONDog`, and this cycle will repeat infinitely until our application crashes. 

To avoid this, we need to create an alias dog type that doesn't have the `MarshalJSON()` and `UnmarsshalJSON()` methods.

```go
type DogAlias Dog
```

Once we have our alias type, we can update our `JSONDog` type to embed this instead of the `Dog` type. We will also need to update our `NewJSONDog()` function to convert our `Dog` into a `DogAlias`, and we can clean up our `Dog()` method on the `JSONDog` type a bit by utilizing the nested `Dog` as our return value.

```go
func NewJSONDog(dog Dog) JSONDog {
  return JSONDog{
    DogAlias(dog),
    dog.BornAt.Unix(),
  }
}

type JSONDog struct {
  DogAlias
  BornAt int64 `json:"born_at"`
}

func (jd JSONDog) Dog() Dog {
  dog := Dog(jd.DogAlias)
  dog.BornAt = time.Unix(jd.BornAt, 0)
  return dog
}
```

As you can see, the initial setup for this took around 30 lines of code, but now that we have it set up, it really doesn't matter how many fields are in the `Dog` type. Our JSON code will only grow if we increase the number of fields with custom JSON.

*You can find all of the code from this section on the Go Playground here: <https://play.golang.org/p/N0rweY-cD0>*


## Custom types for specific fields

The approach we looked at in the last section focused heavily on converting our entire object into another type before encoding and decoding it, but even with the embedded alias, we would need to repeat this code for every different type of object that has a `time.Time` field.

In this section we are going to be looking at an approach that allows us to define how we want a type to encode and decode just once, and then we will reuse that type across our application. Going back to our original example, we are again going to start with the `Dog` type with a `BornAt` field that needs custom JSON.

```go
type Dog struct {
  ID     int       `json:"id"`
  Name   string    `json:"name"`
  Breed  string    `json:"breed"`
  BornAt time.Time `json:"born_at"`
}
```

We already know that this won't work, so rather than using the `time.Time` type, we are going to create our own `Time` type, embed the `time.Time` inside of the new type, and then update our `Dog` type to use our new `Time` type.

```go
type Dog struct {
  ID     int    `json:"id"`
  Name   string `json:"name"`
  Breed  string `json:"breed"`
  BornAt Time   `json:"born_at"`
}

type Time struct {
  time.Time
}
```

Next, we are going to write a custom `MarshalJSON()` and `UnmarshalJSON()` method for our `Time` type. These will output a Unix time, and parse a Unix time respectively.

```go
func (t Time) MarshalJSON() ([]byte, error) {
  return json.Marshal(t.Time.Unix())
}

func (t *Time) UnmarshalJSON(data []byte) error {
  var i int64
  if err := json.Unmarshal(data, &i); err != nil {
    return err
  }
  t.Time = time.Unix(i, 0)
  return nil
}
```

And that's it! We can now freely use our new `Time` type in all of our structs and it will be encoded and decoded as a Unix time. On top of that, because we embedded the `time.Time` object, we can even freely use methods like `Day()` on our own `Time` type, meaning a good portion of our code won't need to be rewritten.

That does bring up one of the downsides to this approach; By using a new type, we do stand the chance of breaking some of our code that expects the `time.Time` type rather than our new `Time` type. You can update all of your code to use the new type, or you can even access the embedded `time.Time` object, so it isn't impossible to fix this, but it may require a little bit of refactoring.

Another solution to this problem is to merge this approach with the first one we looked at. That way we get the best of both worlds - our `Dog` type has a `time.Time` object, but our `JSONDog` doesn't need to worry itself with as many specifics for converting between the two types; The conversion logic is all contained within the new `Time` type.

*A full example of this can be seen on the Go Playground here: <https://play.golang.org/p/C272eojwTh>*


## Encoding and decoding generics

The last technique we are going to look at is a little different than the first two we examined, and is meant to solve an entirely different problem - the problem of dynamic types being stored in nested JSON.

For example, imagine you could get the following JSON responses from a server:

```go
{
  "data": {
    "object": "bank_account",
    "id": "ba_123",
    "routing_number": "110000000"
  }
}
```

And from the same endpoint you might also receive the following:

```go
{
  "data": {
    "object": "card",
    "id": "card_123",
    "last4": "4242"
  }
}
```

At first glance these might look like similar responses with optional data, but they are in fact completely different objects. What you can do with a bank account is different from what you can do with a card, and while I haven't shown them all here, each of these would likely have very different fields. 

One solution to this problem would be to use generics, and to set the type of the nested data when decoding the JSON. You have to use the reflect library a bit, but it *is possible* in a language like Java, and you would end up with some classes like those below.

```java
class Data<T> {
  public T t;
}
class Card {...}
class BankAccount {...}
```

Go, on the other hand, doesn't have generics, so how are we supposed to parse this JSON?

One option is to use a map with our keys being strings, but what data type should we use for our values? Even if we assume it is a nested map, what happens if the card object has an integer value, or a nested JSON object inside of it?

That really limits our options, and we are essentially stuck using the empty interface (`interface{}`).


```go
func main() {
  jsonStr := `
{
  "data": {
    "object": "card",
    "id": "card_123",
    "last4": "4242"
  }
}
`
  var m map[string]map[string]interface{}
  if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
    panic(err)
  }
  fmt.Println(m)

  b, err := json.Marshal(m)
  if err != nil {
    panic(err)
  }
  fmt.Println(string(b))
}
```

Using the empty interface type comes with its own unique set of problems, the most notable being that the empty interface literally tells us nothing about the data. If we want to know anything about the data stored at any key, we are going to need to do a type assertion, and that sucks. Thankfully, there are other ways to approach this problem! 

This approach is going to once again take advantage of the `Marshaler` and `Unmarshaler` interfaces, but this time we are going to add a bit of conditional logic to our code, and we are going to use a type that has both a pointer to a `Card`, and a pointer to a `BankAccount` in it. When we start to decode our JSON, we will first decode the `object` key to determine which of these two fields we should fill, and then we will fill the corresponding field. 

We will start by declaring our types. The `BankAccount` and `Card` types are pretty easy - we are mapping the JSON direct to a Go struct.

```go
type BankAccount struct {
  ID            string `json:"id"`
  Object        string `json:"object"`
  RoutingNumber string `json:"routing_number"`
}

type Card struct {
  ID     string `json:"id"`
  Object string `json:"object"`
  Last4  string `json:"last4"`
}
```

Next we have our `Data` type. You are welcome to name this whatever you want, and it might make sense to use something like `Source` or `CardOrBankAccount`, but that tends to vary from case to case, so I am sticking with `Data` for now.

```go
type Data struct {
  *Card
  *BankAccount
}
```

*We are using pointers here because we won't ever be initializing both of these. Instead, we will determine which one needs to be used, and initialize that one. That means in your code that does use this type, you might need to write something like `if data.Card != nil { ... }` to determine if it is a card. Alternatively, you could also store the `Object` attribute on the `Data` type itself. That choice is ultimately up to you, and it may require some minor code tweaks, but the overall approach discussed here should still work.*

Now that we have a `Data` type, we are going to update our `main()` function so it is clear how this object will map to our JSON. 

```go
func main() {
  jsonStr := `
{
  "data": {
    "object": "card",
    "id": "card_123",
    "last4": "4242"
  }
}
`
  var m map[string]Data
  if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
    panic(err)
  }
  fmt.Println(m)
  data := m["data"]
  if data.Card != nil {
    fmt.Println(data.Card)
  }
  if data.BankAccount != nil {
    fmt.Println(data.BankAccount)
  }

  b, err := json.Marshal(m)
  if err != nil {
    panic(err)
  }
  fmt.Println(string(b))
}
```

`Data` isn't meant to be the entire JSON structure, but is instead meant to represent everything stored inside the `data` key in the JSON object. This is important to note, because in our code our `Data` type has both `Card` and `BankAccount` pointers, but in the JSON these won't be nested objects. That means the first thing we need to do is write a `MarshalJSON()` method to reflect this.

```go
func (d Data) MarshalJSON() ([]byte, error) {
  if d.Card != nil {
    return json.Marshal(d.Card)
  } else if d.BankAccount != nil {
    return json.Marshal(d.BankAccount)
  } else {
    return json.Marshal(nil)
  }
}
```

This code is first checking to see if we have a `Card` or `BankAccount` object. If either is present, it will output the JSON for that corresponding object. If neither is present, it will instead output the JSON for `nil`, which is `null` in JSON. 

*The last bit of the `MarshalJSON()` method might vary in your own code. You may want to return an error in cases like this, or you may want to encode an empty map, but for this example we are using `nil`.*

`MarshalJSON()` wasn't a big departure from what we have done so far, but the `UnmarshalJSON()` method is going to be a bit different. In this method we are going to end up parsing the data twice. The first time we are going to parse using a struct that only has an `Object` field simply so we can determine what type we should be using to decode our JSON, and then we will use that type to decode the JSON.

```go
func (d *Data) UnmarshalJSON(data []byte) error {
  temp := struct {
    Object string `json:"object"`
  }{}
  if err := json.Unmarshal(data, &temp); err != nil {
    return err
  }
  if temp.Object == "card" {
    var c Card
    if err := json.Unmarshal(data, &c); err != nil {
      return err
    }
    d.Card = &c
    d.BankAccount = nil
  } else if temp.Object == "bank_account" {
    var ba BankAccount
    if err := json.Unmarshal(data, &ba); err != nil {
      return err
    }
    d.BankAccount = &ba
    d.Card = nil
  } else {
    return errors.New("Invalid object value")
  }
  return nil
}
```

There are also a few other minor things going on here, like setting the unused field to nil after decoding the data. I am doing this to ensure that our `Data` object is cleared of old data, and our encoding code doesn't run into any bugs. This works well in this case because it isn't ever really valid to have both a `Card` and a `BankAccount` in our `Data` type; Only one should ever be set.

I also define the struct with the `Object` field dynamically, but you could just as easily declare the type elsewhere and use it here. 

*You can find a runnable version of the code from this section on the Go Playground here: <https://play.golang.org/p/gLAgLQv9Et>*


## Wrapping up

While it is impossible to cover every treacherous way that someone might structure their JSON output, I hope that this post has prepared you for handling any other snags you may find along the way. Just remember, you can always start with two separate types using tools like [JSON-to-Go](https://mholt.github.io/json-to-go/) and convert data to/from each type.

I often find that starting with two types can help shed some light on better ways to handle the incoming JSON, and even if it doesn't, always remember that ["done is better than perfect"](http://lifehacker.com/5870379/done-is-better-than-perfect), and you can always come back and refactor your code when you do come up with a better solution.

