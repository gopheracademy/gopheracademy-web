+++
author = ["Ben Schwartz"]
date = "2014-12-22T00:00:00-08:00"
title = "Testing Microservices in Go"
series = ["Advent 2014"]
+++

This post is about testing microservices and why they should be tested differently from many types of software. Microservices are by their very nature simple and encapsulated behind their api. This means two things:

- As long as we don't break the http interface, there is no way to introduce regressions.
- Updates to the implementation of an endpoint are usually going to be close enough to a rewrite that tests will need to be rewritten too.

Unit testing your service's implementation details isn't very important; you can achieve more effective coverage by focusing on component testing the http api.

In this post, I'll walk through testing a [weather microservice](https://github.com/benschw/weather-go) that keeps track of a list of locations and leverages a separate service to get weather details for those locations. Consumers of this service will be written to the service's http api, so we need to make sure that it doesn't break or inadvertently change. The code which satisfies the service's http api isn't complicated and is very small, so we're going to skip unit testing it. Testing to prevent regressions at that low of a level just isn't worth it because application changes would likely necessitate test changes (and not protect against regressions) anyway.

Below, I refer to this level of testing as _component testing_, which I should probably attempt to define. There are [definitions](http://istqbexamcertification.com/what-is-component-testing/) out there but they aren't very consistent. For this article, a component test is higher level then a unit test but lower level than an integration test. It should be easy (quick) to run, but high level enough that changes in the implementation under test shouldn't require updating your test (as long as the changes don't break your microservice's api: the http api).


I'm not sure if any of that made sense and I know it's a hard sell, but maybe a walk-through will illustrate what I'm getting at.

## Weather-Go
[Weather-go](https://github.com/benschw/weather-go) is a Json REST API written in _go_ for the express purpose of illustrating patterns for testing microservices in go. It exposes a single `location` resource with CRUD operations:

- `POST /location`
- `GET /location/{id}`
- `GET /location`
- `PUT /location/{id}`
- `DELETE /location/{id}`

In addition, it contains a client library for [openweathermap.org](http://openweathermap.org/) which is used to include weather details in our `location` resource (temperature and description).

### Get it running
Weather-go uses mysql to store the locations you add, so before you can run the server or the tests, make sure you have a pair of databases (local dev & test) set up. Update the yaml configs appropriately.
	

	# create the database configured in `config.yaml`
	$ mysql -u root -p -e "CREATE DATABASE Location;"

	# create the database configured in `test.yaml`
	$ mysql -u root -p -e "CREATE DATABASE LocationTest;"

Now just grab the project, build, and you're ready to go.

	$ go get github.com/benschw/weather-go
	$ cd $GOPATH/src/github.com/benschw/weather-go
	$ go build
	$ go test ./...

	# add the `location` table
	$ ./weather-go -config ./config.yaml migrate-db

	# start the http server
	$ ./weather-go -config ./config.yaml serve


Now that we have that out of the way, time to talk about testing!

## And that's why you always write a client
Even if you don't plan on leveraging your service in another go app, it pays to write a client library. 

If you do plan on composing many go services together (having one service call another to model complex operations) then even better! In either case, I like to put the client library and the structs that serve as our api model into their own packages. That way your service can depend on the `api` package, but not know or care about the `client` package. Likewise, the client can depend only on the `api` and be imported by another app without exposing the implementation of the service.

Regardless, the reason we're even talking about clients is to support testing our service. Since we've decided to focus on testing our http api, what better way than to actually make http requests and write assertions about the responses.

For `weather-go`, I start up an http server on a random port and use the client I've written to perform tests:

### Suite Setup
I'm using [gocheck](http://gopkg.in/check.v1) which allows for things like `setup` and `teardown` functions in addition to higher level `Assert` calls. Following, I use these fixtures to boot a server to test against and to add/drop the location table.

	type TestSuite struct {
		s *LocationService
	}

	var _ = Suite(&TestSuite{})

	func (s *TestSuite) SetUpSuite(c *C) {
		...
		s.s = &LocationService{...}

		go s.s.Run()
	}

	func (s *TestSuite) SetUpTest(c *C) {
		// add the location table
		s.s.MigrateDb()
	}

	func (s *TestSuite) TearDownTest(c *C) {
		s.s.Db.DropTable(api.Location{})
	}

I've stripped out the noise, but you can see the gist of it above (or the whole thing [here](https://github.com/benschw/weather-go/blob/master/location/location_service_test.go).)

- `SetUpSuite` starts the server in a separate goroutine for us to beat up against.
- `SetUpTest` adds the location table to our test database.
- `TearDownTest` drops all the data we left in the test database so we can start over with a clean slate.

### Testing with the client library
With a running server, we can now make some real http requests and start testing that they behave the way we expect. For example, testing the `POST`:

Here is a test for the happy path. We try to add a location, and it gets added.

	// Location should be added
	func (s *TestSuite) TestAdd(c *C) {
		// given
		locClient := client.LocationClient{Host: s.host}

		// when
		created, err := locClient.AddLocation("Austin", "Texas")

		// then
		c.Assert(err, Equals, nil)
		found, _ := locClient.FindLocation(created.Id)

		c.Assert(created, DeepEquals, found)
	}

Here we test that we get a `400` (bad request) if the location we are trying to add doesn't validate.

	// Client should return ErrStatusBadRequest when entity doesn't validate
	func (s *TestSuite) TestAddBadRequest(c *C) {
		// given
		locClient := client.LocationClient{Host: s.host}

		// when
		_, err := locClient.AddLocation("", "Texas")

		// then
		c.Assert(err, Equals, rest.ErrStatusBadRequest)
	}

And finally we test that we get a `409` (conflict) if we try to `POST` an entity with an Id that already exists. (Note that our client doesn't support doing this, so we had to make the request at a lower level.)

	// Client should return ErrStatusConflict when id exists
	// (not supported by client so pulled impl into test)
	func (s *TestSuite) TestAddConflict(c *C) {
		// given
		locClient := client.LocationClient{Host: s.host}
		created, _ := locClient.AddLocation("Austin", "Texas")

		// when
		url := fmt.Sprintf("%s/location", s.host)
		r, _ := rest.MakeRequest("POST", url, created)
		err := rest.ProcessResponseEntity(r, nil, http.StatusCreated)

		// then
		c.Assert(err, Equals, rest.ErrStatusConflict)
	}

So there it is: component testing our application's http interface. If the underlying implementation changes, these tests will tell us if they've changed in a way that will impact code using our service, but we won't get bogged down in updating lower level tests that at best don't provide additional value, or at worst are brittle and cause false negatives.

(Also, take a look at the [openweather package](https://github.com/benschw/weather-go/tree/master/openweather), which I organized and tested in the same way as the location package: with a `client` and `api` sub package. The only difference is that there is no service implementation, but this way it's exposed to my app in a format I'm used to working with.)

_All the `LocationService` tests, [here](https://github.com/benschw/weather-go/blob/master/location/location_service_test.go)_

## 418: I'm a Teapot
My next point to make regarding component tests for microservices is you should only test things that you have use cases for. You don't need to validate every possible http error - only the ones you're using. Which probably means you don't need to test for "418: I'm a Teapot" or any number of other esoteric status codes.

I've found that there are seven status codes that I regularly use, and I barely, if ever, use the others. 

- http.StatusOK
- http.StatusCreated
- http.StatusConflict
- http.StatusBadRequest
- http.StatusInternalServerError
- http.StatusNotFound
- http.StatusNoContent

You don't need to constrain your service to using as few codes as possible, but make sure you're aware of which are being used and test them all. This list is your cheat sheet for what to test. Adding additional, more granular codes might make for a richer interface, but it also makes for a more brittle one.

## You Mocked me once, never do it again!
To make our tests faster and cleaner, we're going to fake the [openweather client](https://github.com/benschw/weather-go/blob/master/openweather/client/weather_client.go) calls. We can do this by creating a stub implementation that will return some generic weather data for the hard coded "Austin, Texas" query and an empty result for anything else (simulating how the client responds if a city/state isn't found.)

It would work just fine to go ahead and use the real api, but because it is communicating over the WAN with a third party service ([http://api.openweathermap.org/](http://api.openweathermap.org/)), it will make our tests a lot quicker and effective if we fake it. (We are still testing the service, but we constrain these integration tests to the openweather/client package.) Additionally, if this were one of our own services, we wouldn't want to manage setting up that service, its database, and any transitive services, etc. Not to mention, we shouldn't have to know how to set up our microservice dependencies, only how their api works; setting them up here would be a breach of encapsulation.

Writing stubs (or mocks) in go is pretty elegant. All you have to do is define an interface for the component you need to use, refer to it by the interface in your implementation, and then you can mock or stub it out for test. Key to this strategy is that even if you need to use a third party library that doesn't provide an interface, you're OK. Since in _go_ you don't need to declare when you are implementing an interface, you can add interfaces for a third party library in your application. This is even useful for integrating with your own code, because it allows you to constrain the library down to the parts you need and keep the declaration close to your implementation.

(Karl Matthias wrote an article, [Writing Testable Code in Go](http://relistan.com/writing-testable-apps-in-go/), with a particularly good explanation for why you should write your interfaces alongside the code that uses them, not the code that implements them.)

To put this into clearer terms, let's look at an example from `weather-go` where we stub the openweather client.

In our `location` package, we define an interface for the client.

	package location

	import (
		"github.com/benschw/weather-go/openweather/api"
	}

	type WeatherClient interface {
		FindForLocation(city string, state string) (api.Conditions, error)
	}

In the `LocationService`, we refer to the `openweather` client by the interface we just created. The `NewLocationService` factory method specifies to use the implementation from the `openweather` package. 

	import (
		"github.com/benschw/weather-go/openweather/client"
		...
	)

	type LocationService struct {
		...
		// this is the location.WeatherClient, 
		// not the openweather/api.WeatherClient
		WeatherClient WeatherClient 
	}

	func NewLocationService(bind string, dbStr string) (*LocationService, error) {
		s := &LocationService{}
		...
		s.WeatherClient = &client.WeatherClient{}

		return s, nil
	}

Since we built `LocationService` with the client as a field, our tests can inject a stub client. This way, if you build the service with the `NewLocationService` factory, you are wired to use the real client, but you can also define a different implementation and construct a `LocationService` with that:

	type WeatherClientStub struct {
	}

	func (c *WeatherClientStub) FindForLocation(city string, state string) (api.Conditions, error) {
		...
	}

	server := &LocationService{
		...
		WeatherClient: &WeatherClientStub{},
	}

And now, our component tests will flow through a real http server, use a real database, but be tested against a fake weather service. Everything is still snappy and self-contained (no WAN calls and MySQL is a small price to pay for simple testing).

_All the `LocationService` tests, [here](https://github.com/benschw/weather-go/blob/master/location/location_service_test.go)_

### MySQL?

You might have noticed I didn't create a stub for the database; it's a pain to do and MySQL is fast. If we did want to fake the database layer, we could wrap the database object in a `LocationRepository` structure and then stub out that. The increased complexity added to our app only buys us a little though: we would need to separately test the repository, which would probably require running a real database anyway. 

So that's why we just test with a real database.

## These aren't the Drones you're looking for

I typically have used [Drone.io](https://drone.io/) for ci because it's fast and I like that it's open source. But it also gets on my nerves because you have to configure your build steps in the ui and it doesn't support go 1.3 without [some hacking](https://gist.github.com/benschw/7b479c42f426ef07cb79). ([Installing drone](https://github.com/drone/drone) yourself is a different story: it has `.drone.yml` for build step configuration and you get to supply whatever container images you want!)

Anyway, I was excited to see that [Travis CI](https://travis-ci.org/) has added Docker support and this seemed like a great chance to try it out. The verdict? Works just as advertised and my builds started up a lot quicker than with VMs. Here's my config for `weather-go`. It creates the `LocationTest` database and tests our app against go 1.3 and "tip"

`.travis.yml`

	sudo: false
	language: go

	go:
	  - 1.3
	  - tip

	services:
	  - mysql

	before_script:
	  - mysql -e 'create database LocationTest;'

	script: 
	  - go get 
	  - go get gopkg.in/check.v1 
	  - go test ./...
	  - go build

_(for whatever reason, `sudo: false` is what makes it use Docker.)_



## So long and thanks for all the fish
So that's it. Jump to [weather-go on github](https://github.com/benschw/weather-go) or directly to the [LocationService tests](https://github.com/benschw/weather-go/blob/master/location/location_service_test.go).

Obviously you've got to be pragmatic about what you do and don't test, but with microservices you might have to retrain yourself in order to make sure you're spending your time on the right tests. When the lines of code in your application can be measured in "tens" instead of "thousands", and when your external contract is an explicit http api, you get better results by focusing on the edge.


Follow me on Twitter at [@benhschwartz](https://twitter.com/benhschwartz) and read more at [txt.fliglio.com](http://txt.fliglio.com/).
