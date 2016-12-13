+++
author = ["Ernesto Jimenez"]
date = "2016-12-16T00:00:00+00:00"
series = ["Advent 2016"]
title = "Writing good unit tests for SOLID Go"
+++

Dave Cheney covered how interfaces are used to design good Go code in his [SOLID Go Design talk][solid-go-talk] and [blog post][solid-go-post].

In this blog post, we are going to focus on some tips on how to write unit tests for that beautiful SOLID code.

The primary objectives for our test builds are to:

- **Increase the confidence in our code**. Otherwise, the test code is just dead weight.
- **Be fast**. Who likes waiting 15 minutes for the tests to pass?
- **Be stable**. Tests should never fail randomly, and a small change should never break unrelated tests.
- **Be short**. It must be the shortest possible checks required to increase our confidence in the code.

Most of us have had to suffer test suites which fell short of achieving these objectives. Objects with dependencies are, in many cases, the most problematic.

## Starting with an example

When we implement a SOLID design, as a rule of thumb our **structs will depend on interfaces instead of structs**.

Let's look at a simplified example of Go code that would use interfaces for its dependencies.

We are building a struct to manage users in our application. We will call it `UserManager`.

```go
// UseManager
type UserManager struct {
    notifier UserNotifier // Used to schedule emails
    store    UserStore    // Used to persist users
}

// SignUp creates user account pending activation and sends the activation email.
// It will return an error if creating the user failed  due to invalid details or problems with the UserStore.
func (um *UserManager) SignUp(ctx context.Context, user User) (*User, error) {
    // [...]
}

// [...]
```

As we can see, `UserManager` is dependent on the following two interfaces:

```go
// UserNotifier specifies the methods to schedule emails to be sent to certain user
type UserNotifier interface {
    RequestActivation(ctx context.Context, id string) error
    RecoverPassword(ctx context.Context, id string) error
}

// UserStore specifies the methods required to manage user accounts
type UserStore interface {
    Find(ctx context.Context, id string) (*User, error)
    Update(context.Context, User) error
    Create(context.Context, User) (*User, error)
}
```

Following the SOLID principles, we will create a function to initialise the `UserManager` injecting the dependencies:

```go
func NewUserManager(store UserStore, notifier UserNotifier) *UserManager {
    return &UserManager{store: store, notifier: notifier}
}
```

Once we have everything wired, the implementation of the UserManager methods could be something like this:

```go
func (um *UserManager) SignUp(ctx context.Context, user User) (*User, error) {
    u, err := um.store.Create(ctx, user)
    if err != nil {
        return nil, err
    }
    um.notifier.RequestActivation(ctx, u.ID)
    return u, nil
}
```

## What should we test?

When testing an object, it receiving and sending messages:

- **Incoming messages** refer to calls to methods on the tested object.
- **Outgoing messages** refers to method calls the tested object does ot it's dependencies.

Following with our example, if we were to test the `SignUp` method, `SignUp` would be the incoming message while the calls to `um.store.Create` and `um.notifier.RequestActivation` would be outgoing messages.

Furthermore, a message can be a Query or a Command:

- Query messages return data without changing anything. e.g.: `UserStore.Find(id string) (*User, error)` would return a user without making any changes in the store.
- Command messages modify data without returning any data. e.g.: `UserStore.Update(User) error` would make changes in the store without returning any new data.
- Some commands might return some data, but we should be careful, cautious messages that return data and make modifications. We must ensure the changes are never hidden side-effects but required business logic. e.g.: `UserStore.Create(User) (*User, error)` will add a user to the store and must return the information about such user so we can get the ID of the user.

This classification will help us guide what we need to test based on the types of messages affected:

- **Incoming queries**: send the message and assert the response.
- **Incoming commands**: send the message and assert the public changes. e.g.: call `UserStore.Delete` on an existing user id
- **Outgoing queries**: nothing to assert.
- **Outgoing commands**: assert the message sent.

So, how does this apply to our example?

## A bad test

Many people would opt to go straight into developing an integration test for `SignUp`.

```go
import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestSignUpSuccessful(t *testing.T) {
    // 1. Prepare the object with the right context
    us := newTestUserStore() // user store pointing to a local empty DB
    un := newTestUserNotifier() // user notifier pointing to a local queue
    um := NewUserManager(us, un)

    // 2. Send incoming command-query
    u, err := us.SignUp(User{Email: "ernesto@slackline.io", Password: "secret"})

    // 3. Since it was a query, asssert result
    assert.NoError(t, err)
    assert.NotEmpty(t, u.ID)
    assert.Equal(t, "ernesto@slackline.io", u.email)

    // 4. Since it was a command, assert public side effects
    stored, err = us.Find(u.ID)
    assert.NoError(t, err)
    assert.Equal(t, u.ID, stored.ID)
    assert.Equal(t, "ernesto@slackline.io", stored.email)

    // 5. testing outgoing messages??
}
```

Integration tests provide you end to end checks. But are much more expensive than unit tests:

- Running them is slower since you have to call external services. e.g.: in this case, we have to create/clear our test database for each test.
- You might not be able to run tests in parallel. e.g.: if you were clearing the same database for each test, running them in parallel could result in false failures due to race conditions preparing the database.
- With outgoing queries, you need to provision test data before each test. e.g.: if we were to test a `UserManager.Find` method; we would need to add an entry to the `UserStore` before calling `UserManager.Find`.
- With outgoing commands, you must assert the side effects on your dependency instead of the outgoing commands. e.g.: if you wanted to test `um.notifier.RequestActivation`, you would have to read the queue to make sure `u.ID` got queued up.
- If your method depends on external services such as third party APIs, they can take seconds to respond and could have downtime. Which would

Luckily, when since we follow a SOLID design, our dependencies are defined as interfaces, so we can implement a unit test.

Let's look at that next:

### Developing a unit test

Since our dependencies are defined as interfaces, we can use mocks to assert outgoing messages.

```go
import (
    "testing"

    "github.com/stretchr/testify/assert"
)

// Generate mocks for our dependencies using `go generate`

//go:generate goautomock -template=testify UserStore
//go:generate goautomock -template=testify UserNotifier

func TestSignUpSuccessful(t *testing.T) {
    user := User{Email: "ernesto@slackline.io", Password: "secret"}
    returned := &User{ID: "new-user", Email: "ernesto@slackline.io", Password: "secret"}

    // 1. Mock outgoing messages, asserting outgoing command and returning stubbed result for outgoing query
    us := NewUserStoreMock()
    us.On("Create", ctx, user).Return(returned, nil).Once()
    un := NewUserNotifierMock()
    un.On("RequestActivation", ctx, "new-user").Return(nil).Once()
    um := NewUserManager(us, un)

    // 2. Send incoming command-query
    u, err := us.SignUp(user)

    // 3. Since it was a query, assert the result
    assert.NoError(t, err)
    assert.Equal(t, returned, u)

    // 4. Assert outgoing command messages were sent properly
    un.AssertExpectations(t)
    us.AssertExpectations(t)
}
```

Main differences:

- Tests only focus on specifying dependencies and assertions, no need to fiddle creating/clearing databases.
- It will be fast, since everything will be done in the same process.
- Tests can run in parallel, since we have removed the race conditions.
- It is easy to mock any dependency, including calls to third party APIs.
- We are using [github.com/ernesto-jimenez/goautomock][goautomock] to automatically generate the mocks based on the interface using `go generate`. Zero boilerplate required.

As you can see, the benefits are many, specially for dependencies that are harder than databases to setup/teardown.

## Should we still create integration tests?

Definitely, but we just have to be aware of the testing pyramid:

- Start with a foundation of unit tests since they are the cheapest ones to create, run and maintain.
- Add other types of testing on top of the unit tests: integration, end to end, ui, manual... The most expensive a kind of test is, the higher up in the pyramid it should be.

### Have any questions or feedback?

I would love to gather your thoughts and some ideas for future posts.

Do you have any questions or feedback? send me a line to [ernesto@slackline.io](mailto:ernesto@slackline.io).

[solid-go-talk]: https://www.youtube.com/watch?v=zzAdEt3xZ1M
[solid-go-post]: https://dave.cheney.net/2016/08/20/solid-go-design
[goautomock]: https://github.com/ernesto-jimenez/goautomock
