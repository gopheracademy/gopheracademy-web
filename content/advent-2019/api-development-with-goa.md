+++
title = "API Development in Go, using Goa"
date = "2019-12-29T00:00:00+00:00"
series = ["Advent 2019"]
author = ["Gleidson Nascimento"]
+++


API development is a hot topic nowadays. There's a huge number of ways you can 
develop and ship an API and big companies have developed massive solutions to 
help you bootstrap an application quickly.

Yet, most of those options lack a key feature: Development Lifecycle Management. 
So, developers spend few cycles creating robust APIs but end up struggling with 
the expected organic evolution of their code and the implications that a small 
change in the API has in the source code. 

But in 2016, Raphael Simon ([@rgsimon](https://twitter.com/rgsimon)) created 
[Goa](https://goa.design), a framework for API development in golang with a 
lifecycle that puts API design first. In Goa, your API definition is not only 
described as code but also is the source from which server code, client code, 
and documentation is derived. This means that every part of your code is 
described in your API definition using a golang DSL, then generated using the 
goa cli and implemented separately from your application source code. 

And, in my opinion, that's the reason why Goa shines: It's a solution with a 
well defined development lifecycle contract that relies on best practices when 
generating code (like, for example, splitting different domains and concerns in 
layers, so transport aspects do not interfere with business aspects of 
application), following a clean architecture pattern where composable modules 
are generated for the transport, endpoint, and business logic layers in your application. 

Some of the features of Goa, as defined in the [official website](https://goa.design), are:

- Composability: The package, code generation algorithms, and generated code are 
all more modular.
- Transport-agnostic. The decoupling of transport layer from the actual service 
implementation means that the same service can expose endpoints accessible via 
multiple transports such as HTTP and/or gRPC. 
- Separation of concerns: The actual service implementation is isolated from the 
transport code. 
- Use of Go standard library types: Makes it easier to interface with external code.

In this article, I will create an application and walk you through the stages of 
API development lifecycle. The application manages details about clients, such 
as name, address, phone number, social media, etc. In the end, we will attempt 
to extend it and add new features to exercise its development lifecycle.

So, let's get started! 👍

## Preparing your development area

Our first step is to initiate the repository and enable go modules support:

```bash
mkdir -p clients/design
cd clients
go mod init clients
```

At the end, your repo structure should be like below:

```bash
$ tree
.
├── design
└── go.mod
```

## Designing your API

The source of truth for your API is your design definition. As per the 
documentation, "Goa lets you think about your APIs independently of any 
implementation concern and then review that design with all stakeholders before 
writing the implementation". This means that every element of the API is defined 
here first before the actual application code is generated. But enough talking! 
Open the file `clients/design/design.go` and add the content below:

```go
/*
This is the design file. It contains the API specification, methods, inputs
and outputs using Goa DSL code. The objective is to use this as a single
source of truth for the entire API source code.
*/
package design

import (
	. "goa.design/goa/v3/dsl"
)

// Main API declaration
var _ = API("clients", func() {
	Title("An api for clients")
	Description("This api manages clients with CRUD operations")
	Server("clients", func() {
		Host("localhost", func() {
			URI("http://localhost:8080/api/v1")
		})
	})
})

// Client Service declaration with two methods and Swagger API specification file
var _ = Service("client", func() {
	Description("The Client service allows access to client members")
	Method("add", func() {
		Payload(func() {
			Field(1, "ClientID", String, "Client ID")
			Field(2, "ClientName", String, "Client ID")
			Required("ClientID", "ClientName")
		})
		Result(Empty)
		Error("not_found", NotFound, "Client not found")
		HTTP(func() {
			POST("/api/v1/client/{ClientID}")
			Response(StatusCreated)
		})
	})

	Method("get", func() {
		Payload(func() {
			Field(1, "ClientID", String, "Client ID")
			Required("ClientID")
		})
		Result(ClientManagement)
		Error("not_found", NotFound, "Client not found")
		HTTP(func() {
			GET("/api/v1/client/{ClientID}")
			Response(StatusOK)
		})
	})
	
	Method("show", func() {
		Result(CollectionOf(ClientManagement))
		HTTP(func() {
			GET("/api/v1/client")
			Response(StatusOK)
		})
	})
	Files("/openapi.json", "./gen/http/openapi.json")
})

// ClientManagement is a custom ResultType used to configure views for our custom type
var ClientManagement = ResultType("application/vnd.client", func() {
	Description("A ClientManagement type describes a Client of company.")
	Reference(Client)
	TypeName("ClientManagement")

	Attributes(func() {
		Attribute("ClientID", String, "ID is the unique id of the Client.", func() {
			Example("ABCDEF12356890")
		})
		Field(2, "ClientName")
	})

	View("default", func() {
		Attribute("ClientID")
		Attribute("ClientName")
	})

	Required("ClientID")
})

// Client is the custom type for clients in our database
var Client = Type("Client", func() {
	Description("Client describes a customer of company.")
	Attribute("ClientID", String, "ID is the unique id of the Client Member.", func() {
		Example("ABCDEF12356890")
	})
	Attribute("ClientName", String, "Name of the Client", func() {
		Example("John Doe Limited")
	})
	Required("ClientID", "ClientName")
})

// NotFound is a custom type where we add the queried field in the response
var NotFound = Type("NotFound", func() {
	Description("NotFound is the type returned when " +
		"the requested data that does not exist.")
	Attribute("message", String, "Message of error", func() {
		Example("Client ABCDEF12356890 not found")
	})
	Field(2, "id", String, "ID of missing data")
	Required("message", "id")
})

```

The first thing you can notice is that the DSL above is a set of Go functions 
that can be composed to describe a remote service API. The functions are 
composed using anonymous function arguments. in the DSL functions, we have a 
subset of functions that are not supposed to appear within other functions, 
which we call top-level DSLs. Below you have a partial set of DSL functions and 
their structure:

```bash
API                 Service          Type            ResultType
├── Title           ├── Description  ├── Extend      ├── TypeName
├── Description     ├── Docs         ├── Reference   ├── ContentType
├── Version         ├── Security     ├── ConvertTo   ├── Extend
├── Docs            ├── Error        ├── CreateFrom  ├── Reference
├── License         ├── GRPC         ├── Attribute   ├── ConvertTo
├── TermsOfService  ├── HTTP         ├── Field       ├── CreateFrom
├── Contact         ├── Method       └── Required    ├── Attributes
├── Server          │   ├── Payload                  └── View
└── HTTP            │   ├── Result
                    │   ├── Error
                    │   ├── GRPC
                    │   └── HTTP
                    └── Files
``` 

So, we have in our initial design an API top-level DSL describing our clients 
API, one Service top-level DSLs describing the principal API service, `clients` 
and serving the API swagger file, and two type top-level DSLs for describing the 
object view type used in the transport payload.

The `API` function is an optional top-level DSL which lists the global properties 
of the API such as a name, a description and also one or more Servers potentially 
exposing different sets of services. In our case, one server is enough, but you 
could for example serve different services in different tiers - development, 
test and production for example.

The `Service` function defines a group of methods, which potentially maps to a 
resource in the transport. A service may also define common error responses. The 
service methods are described using `Method`. This function defines the method 
payload (input) and result (output) types. If you omit the payload or result 
type, the built-in type Empty which maps to an empty body in HTTP is used. 

Finally, the `Type` or `ResultType` functions define User-defined types. The 
main difference between them is that a result type is a type that also defines a 
set of “views”.

In our example, we described the api and explained how it should serve, created 
a service called `clients`, and created three methods - `add` (for creating one 
client) `get` (for retrieving one client) and `show` (for listing all clients) - 
and created our own custom types, which will come handy when we integrate with a 
database - and a customized error type. 

Now that our application was described, we can generate the boilerplate code. 
The following command takes the design package import path as an argument. It 
also accepts the path to the output directory as an optional flag:

```bash
goa gen clients/design
```

The command outputs the names of the files it generates. In there, the `gen` 
directory contains the application name sub-directory which houses the 
transport-independent service code. The `http` sub-directory describes the HTTP 
transport (In there we have server and client code with the logic to encode and 
decode requests and responses and the CLI code to build HTTP requests from the 
command line). It also contains the Open API 2.0 specification files in both 
json and yaml formats. 

You can copy the content of swagger file and paste it on any online Swagger 
editor (like the one at [swagger.io website](https://editor.swagger.io/)) for 
visualizing your API specification documentation - They support both YAML and 
JSON formats.

We are now ready for our next step in the development lifecycle!

## Implementing your API

After your boilerplate code was created, it's time to add some business logic to 
it. At this point, this is how your code should look like:

```bash
$ tree
.
├── design
│   └── design.go
├── gen
│   ├── client
│   │   ├── client.go
│   │   ├── endpoints.go
│   │   ├── service.go
│   │   └── views
│   │       └── view.go
│   └── http
│       ├── cli
│       │   └── my_clients_api
│       │       └── cli.go
│       ├── client
│       │   ├── client
│       │   │   ├── cli.go
│       │   │   ├── client.go
│       │   │   ├── encode_decode.go
│       │   │   ├── paths.go
│       │   │   └── types.go
│       │   └── server
│       │       ├── encode_decode.go
│       │       ├── paths.go
│       │       ├── server.go
│       │       └── types.go
│       ├── openapi.json
│       └── openapi.yaml
├── go.mod
└── go.sum
```

Where, every file above is maintained and updated by Goa whenever we execute the 
cli. Thus, as the architecture evolves, your design will follow the evolution 
and your source code will too. To implement the application, we execute the 
command below - that will generate a basic implementation of the service along 
with buildable server files that spins up goroutines to start a HTTP server and 
client files that can make requests to that server:

```bash
goa example clients/design
```

This will generate a cmd folder with both server and client buildable sources. 
And there will be your application and those are the files you should maintain 
yourself after Goa firstly generates them ([Goa documentation](https://goa.design/learn/getting-started/) 
makes clear that *"This command generates a starting point for the service to 
help bootstrap development - in particular it is NOT meant to be re-run when the 
design changes"* ). 

Now, your code will look like:

```bash
$ tree
.
├── client.go
├── cmd
│   ├── my_clients_api
│   │   ├── http.go
│   │   └── main.go
│   └── my_clients_api-cli
│       ├── http.go
│       └── main.go
├── design
│   └── design.go
├── gen
│   ├── client
│   │   ├── client.go
│   │   ├── endpoints.go
│   │   ├── service.go
│   │   └── views
│   │       └── view.go
│   └── http
│       ├── cli
│       │   └── my_clients_api
│       │       └── cli.go
│       ├── client
│       │   ├── client
│       │   │   ├── cli.go
│       │   │   ├── client.go
│       │   │   ├── encode_decode.go
│       │   │   ├── paths.go
│       │   │   └── types.go
│       │   └── server
│       │       ├── encode_decode.go
│       │       ├── paths.go
│       │       ├── server.go
│       │       └── types.go
│       ├── openapi.json
│       └── openapi.yaml
├── go.mod
└── go.sum
```

Where `client.go` is an example file with a dummy implementation of both `get` 
and `show` methods. Let's add some business logic to it!

For simplicity, we will use Sqlite instead of an in-memory database and Gorm as 
our ORM. Create the file `sqlite.go` and add the content below - that will add 
database logic to create records on the database, and list one and/or many rows 
from database:

```go
package clients

import (
	"clients/gen/client"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)
var db *gorm.DB
var err error
type Client *client.ClientManagement

// InitDB is the function that starts a database file and table structures
// if not created then returns db object for next functions
func InitDB() *gorm.DB {
	// Opening file
	db, err := gorm.Open("sqlite3", "./data.db")
	// Display SQL queries
	db.LogMode(true)

	// Error
	if err != nil {
		panic(err)
	}
	// Creating the table if it doesn't exist
	var TableStruct = client.ClientManagement{}
	if !db.HasTable(TableStruct) {
		db.CreateTable(TableStruct)
		db.Set("gorm:table_options", "ENGINE=InnoDB").CreateTable(TableStruct)
	}

	return db
}

// GetClient retrieves one client by its ID
func GetClient(clientID string) (client.ClientManagement, error) {
	db := InitDB()
	defer db.Close()

	var clients client.ClientManagement
	db.Where("client_id = ?", clientID).First(&clients)
	return clients, err
}

// CreateClient created a client row in DB
func CreateClient(client Client) error {
	db := InitDB()
	defer db.Close()
	err := db.Create(&client).Error
	return err
}

// ListClients retrieves the clients stored in Database
func ListClients() (client.ClientManagementCollection, error) {
	db := InitDB()
	defer db.Close()
	var clients client.ClientManagementCollection
	err := db.Find(&clients).Error
	return clients, err
}
```

Then, we edit client.go to update all methods in client Service, implementing 
the database calls and constructing the API responses:

```go
// Add implements add.
func (s *clientsrvc) Add(ctx context.Context, 
	p *client.AddPayload) (err error) {
	s.logger.Print("client.add started")
	newClient := client.ClientManagement{
		ClientID: p.ClientID,
		ClientName: p.ClientName,
	}
	err = CreateClient(&newClient)
	if err != nil {
		s.logger.Print("An error occurred...")
		s.logger.Print(err)
		return
	}
	s.logger.Print("client.add completed")
	return
}

// Get implements get.
func (s *clientsrvc) Get(ctx context.Context, 
	p *client.GetPayload) (res *client.ClientManagement, err error) {
	s.logger.Print("client.get started")
	result, err := GetClient(p.ClientID)
	if err != nil {
		s.logger.Print("An error occurred...")
		s.logger.Print(err)
		return
	}
	s.logger.Print("client.get completed")
	return &result, err
}

// Show implements show.
func (s *clientsrvc) Show(ctx context.Context) (res client.ClientManagementCollection, 
	err error) {
	s.logger.Print("client.show started")
	res, err = ListClients()
	if err != nil {
		s.logger.Print("An error occurred...")
		s.logger.Print(err)
		return
	}
	s.logger.Print("client.show completed")
	return
}
```

And the first cut of our application is ready to be compiled. Run the following 
command to create server and client binaries:

```bash
go build ./cmd/clients 
go build ./cmd/clients-cli
```

To run the server, just run `./clients`. Leave it running for now. You should 
see it running successfully, like the following:

```bash
$ ./clients
[clients] 00:00:01 HTTP "Add" mounted on POST /api/v1/client/{ClientID}
[clients] 00:00:01 HTTP "Get" mounted on GET /api/v1/client/{ClientID}
[clients] 00:00:01 HTTP "Show" mounted on GET /api/v1/client
[clients] 00:00:01 HTTP "./gen/http/openapi.json" mounted on GET /openapi.json
[clients] 00:00:01 HTTP server listening on "localhost:8080"
```

We are ready to perform some testing in our application! Let's try out all 
methods using the cli:

```bash
$ ./clients-cli client add --body '{"ClientName": "Cool Company"}' \ 
--client-id "1"
$ ./clients-cli client get --client-id "1"
{
    "ClientID": "1",
    "ClientName": "Cool Company"

}
$ ./clients-cli client show               
[
    {
        "ClientID": "1",
        "ClientName": "Cool Company"
    }
]

```

If you get any error, check the server logs to ensure that the sqlite ORM logic 
is good and you are not facing any db errors such as database not initialized or 
queries returning no rows.


## Extending your API

The framework supports the development of plugins to extend your API and add 
more features easily. Goa has a [repository](https://github.com/goadesign/plugins) for plugins created by the community.
As I explained earlier, as part of the development lifecycle, we can rely on the 
toolset to extend our application, by going back to the design definition, 
updating it and refreshing our generated code. Let's showcase how plugins can 
help on that by adding CORS and authentication to the API. 
 
Update the file `clients/design/design.go` to the content below:
 
```go
/*
This is the design file. It contains the API specification, methods, inputs
and outputs using Goa DSL code. The objective is to use this as a single
source of truth for the entire API source code.
*/
package design

import (
	. "goa.design/goa/v3/dsl"
	cors "goa.design/plugins/v3/cors/dsl"
)

// Main API declaration
var _ = API("clients", func() {
	Title("An api for clients")
	Description("This api manages clients with CRUD operations")
	cors.Origin("/.*localhost.*/", func() {
		cors.Headers("X-Authorization", "X-Time", "X-Api-Version",
			"Content-Type", "Origin", "Authorization")
		cors.Methods("GET", "POST", "OPTIONS")
		cors.Expose("Content-Type", "Origin")
		cors.MaxAge(100)
		cors.Credentials()
	})
	Server("clients", func() {
		Host("localhost", func() {
			URI("http://localhost:8080/api/v1")
		})
	})
})

// Client Service declaration with two methods and Swagger API specification file
var _ = Service("client", func() {
	Description("The Client service allows access to client members")
	Error("unauthorized", String, "Credentials are invalid")
	HTTP(func() {
		Response("unauthorized", StatusUnauthorized)
	})
	Method("add", func() {
		Payload(func() {
			TokenField(1, "token", String, func() {
				Description("JWT used for authentication")
			})
			Field(2, "ClientID", String, "Client ID")
			Field(3, "ClientName", String, "Client ID")
			Field(4, "ContactName", String, "Contact Name")
			Field(5, "ContactEmail", String, "Contact Email")
			Field(6, "ContactMobile", Int, "Contact Mobile Number")
			Required("token", 
				"ClientID", "ClientName", "ContactName", 
				"ContactEmail", "ContactMobile")
		})
		Security(JWTAuth, func() {
			Scope("api:write")
		})
		Result(Empty)
		Error("invalid-scopes", String, "Token scopes are invalid")
		Error("not_found", NotFound, "Client not found")
		HTTP(func() {
			POST("/api/v1/client/{ClientID}")
			Header("token:X-Authorization")
			Response("invalid-scopes", StatusForbidden)
			Response(StatusCreated)
		})
	})

	Method("get", func() {
		Payload(func() {
			TokenField(1, "token", String, func() {
				Description("JWT used for authentication")
			})
			Field(2, "ClientID", String, "Client ID")
			Required("token", "ClientID")
		})
		Security(JWTAuth, func() {
			Scope("api:read")
		})
		Result(ClientManagement)
		Error("invalid-scopes", String, "Token scopes are invalid")
		Error("not_found", NotFound, "Client not found")
		HTTP(func() {
			GET("/api/v1/client/{ClientID}")
			Header("token:X-Authorization")
			Response("invalid-scopes", StatusForbidden)
			Response(StatusOK)
		})
	})
	
	Method("show", func() {
		Payload(func() {
			TokenField(1, "token", String, func() {
				Description("JWT used for authentication")
			})
			Required("token")
		})
		Security(JWTAuth, func() {
			Scope("api:read")
		})
		Result(CollectionOf(ClientManagement))
		Error("invalid-scopes", String, "Token scopes are invalid")
		HTTP(func() {
			GET("/api/v1/client")
			Header("token:X-Authorization")
			Response("invalid-scopes", StatusForbidden)
			Response(StatusOK)
		})
	})
	Files("/openapi.json", "./gen/http/openapi.json")
})

// ClientManagement is a custom ResultType used to 
// configure views for our custom type
var ClientManagement = ResultType("application/vnd.client", func() {
	Description("A ClientManagement type describes a Client of company.")
	Reference(Client)
	TypeName("ClientManagement")

	Attributes(func() {
		Attribute("ClientID", String, "ID is the unique id of the Client.", func() {
			Example("ABCDEF12356890")
		})
		Field(2, "ClientName")
		Attribute("ContactName", String, "Name of the Contact.", func() {
			Example("John Doe")
		})
		Field(4, "ContactEmail")
		Field(5, "ContactMobile")
	})

	View("default", func() {
		Attribute("ClientID")
		Attribute("ClientName")
		Attribute("ContactName")
		Attribute("ContactEmail")
		Attribute("ContactMobile")
	})

	Required("ClientID")
})

// Client is the custom type for clients in our database
var Client = Type("Client", func() {
	Description("Client describes a customer of company.")
	Attribute("ClientID", String, "ID is the unique id of the Client Member.", func() {
		Example("ABCDEF12356890")
	})
	Attribute("ClientName", String, "Name of the Client", func() {
		Example("John Doe Limited")
	})
	Attribute("ContactName", String, "Name of the Client Contact.", func() {
		Example("John Doe")
	})
	Attribute("ContactEmail", String, "Email of the Client Contact", func() {
		Example("john.doe@johndoe.com")
	})
	Attribute("ContactMobile", Int, "Mobile number of the Client Contact", func() {
		Example(12365474235)
	})
	Required("ClientID", "ClientName", "ContactName", "ContactEmail", "ContactMobile")
})

// NotFound is a custom type where we add the queried field in the response
var NotFound = Type("NotFound", func() {
	Description("NotFound is the type returned " +
		"when the requested data that does not exist.")
	Attribute("message", String, "Message of error", func() {
		Example("Client ABCDEF12356890 not found")
	})
	Field(2, "id", String, "ID of missing data")
	Required("message", "id")
})

// Creds is a custom type for replying Tokens
var Creds = Type("Creds", func() {
	Field(1, "jwt", String, "JWT token", func() {
		Example("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." +
			"eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9" +
			"lIiwiYWRtaW4iOnRydWV9.TJVA95OrM7E2cBab30RMHrHD" +
			"cEfxjoYZgeFONFh7HgQ")
	})
	Required("jwt")
})

// JWTAuth is the JWTSecurity DSL function for adding JWT support in the API
var JWTAuth = JWTSecurity("jwt", func() {
	Description(`Secures endpoint by requiring a valid 
JWT token retrieved via the signin endpoint. Supports 
scopes "api:read" and "api:write".`)
	Scope("api:read", "Read-only access")
	Scope("api:write", "Read and write access")
})

// BasicAuth is the BasicAuth DSL function for 
// adding basic auth support in the API
var BasicAuth = BasicAuthSecurity("basic", func() {
	Description("Basic authentication used to " +
		"authenticate security principal during signin")
	Scope("api:read", "Read-only access")
})

// Signin Service is the service used to authenticate users and assign JWT tokens for their sessions
var _ = Service("signin", func() {
	Description("The Signin service authenticates users and validate tokens")
	Error("unauthorized", String, "Credentials are invalid")
	HTTP(func() {
		Response("unauthorized", StatusUnauthorized)
	})
	Method("authenticate", func() {
		Description("Creates a valid JWT")
		Security(BasicAuth)
		Payload(func() {
			Description("Credentials used to authenticate to retrieve JWT token")
			UsernameField(1, "username", 
				String, "Username used to perform signin", func() {
				Example("user")
			})
			PasswordField(2, "password", 
				String, "Password used to perform signin", func() {
				Example("password")
			})
			Required("username", "password")
		})
		Result(Creds)
		HTTP(func() {
			POST("/signin/authenticate")
			Response(StatusOK)
		})
	})
})
```

You can notice the two major differences in the new design: we defined a 
security scope in the `client` service, so we can validate if a user is 
authorized to invoke the service, and we defined a second service called 
`signin`, which we will use to authenticate users and generate JWT tokens which 
the `client` service will use to authorize calls. But we have also added more 
fields to our custom client Type. This is often a common case when developing an 
API, a need to reshape or restructure our data. 

On design, these changes may sound simple, but reflecting about them, there's a 
lot of minimal features required to achieve what is described on design. Take, 
for example, the architectural schematics for Authentication and Authorization 
using our API methods:

```bash
# Authenticate
+--------------------------------+      user, pass        +------------------+
|                                +------------------------>                  |
|   POST /signin/authenticate    |       JWT Token        |  signin Service  |
|                                <------------------------+                  |
+--------------------------------+                        ++-----------------+
                                                           |
                                                           | Validate creds
                                                           | Set API scope
                                                           - Define expiry


# Authorize
+--------------------------------+        JWT Token        +--------------------+
|                                +------------------------>+                    |
| GET /api/v1/client/{client_id} |   Client Type Response  |   client Service   |
|                                <-------------------------+                    |
+--------------------------------+                         ++-------------------+
                                                            |
                                                            | Validate token
                                                            | Validate scope
                                                            - Validate expiry
```

Those are all new features that our code doesn't have yet. Again, this is where 
Goa adds more value to your development efforts! Let's implement these features 
at the transport side by regenerating again the source code with the command below:

```bash
goa gen clients/design
```

At this point, if you happen to be using git, you will notice the presence of 
new files with others showing as updated. This is because Goa seamlessly 
refreshed the boilerplace code accordingly without our intervention. 

Now, we need to implement the service side code. In a real-world application, 
you would be updating the application yourself manually after updating your 
source to reflect all the design changes. This is the way that Goa recommends we 
proceed, but for brevity, I will be deleting and regenerating the example 
application to get us there faster. Run the commands below to delete the example application and regenerate it:

```bash
rm -rf cmd client.go
goa example clients/design
```

With that, your code should look like the following:

```bash
$ tree
.
├── client.go
├── cmd
│   ├── clients
│   │   ├── http.go
│   │   └── main.go
│   └── clients-cli
│       ├── http.go
│       └── main.go
├── design
│   └── design.go
├── gen
│   ├── client
│   │   ├── client.go
│   │   ├── endpoints.go
│   │   ├── service.go
│   │   └── views
│   │       └── view.go
│   ├── http
│   │   ├── cli
│   │   │   └── clients
│   │   │       └── cli.go
│   │   ├── client
│   │   │   ├── client
│   │   │   │   ├── cli.go
│   │   │   │   ├── client.go
│   │   │   │   ├── encode_decode.go
│   │   │   │   ├── paths.go
│   │   │   │   └── types.go
│   │   │   └── server
│   │   │       ├── encode_decode.go
│   │   │       ├── paths.go
│   │   │       ├── server.go
│   │   │       └── types.go
│   │   ├── openapi.json
│   │   ├── openapi.yaml
│   │   └── signin
│   │       ├── client
│   │       │   ├── cli.go
│   │       │   ├── client.go
│   │       │   ├── encode_decode.go
│   │       │   ├── paths.go
│   │       │   └── types.go
│   │       └── server
│   │           ├── encode_decode.go
│   │           ├── paths.go
│   │           ├── server.go
│   │           └── types.go
│   └── signin
│       ├── client.go
│       ├── endpoints.go
│       └── service.go
├── go.mod
├── go.sum
└── signin.go
```

We can see one new files in our example application: `signin.go`, which contains 
the signin Service logic. However, we can see that `client.go` was also updated 
with a JWTAuth function for validating tokens. This matches what we have written 
in the design, so every call to any method in client will be intercepted for 
token validation and forwarded only if authorized by a valid token and a correct scope.

Therefore, we will update the methods in our signin Service inside `signin.go` 
in order to add the logic to generate the Tokens the API will create for 
Authenticated users. Copy and paste the following contect into `signin.go`:

```go
package clients

import (
	signin "clients/gen/signin"
	"context"
	"log"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"goa.design/goa/v3/security"
)

// signin service example implementation.
// The example methods log the requests and return zero values.
type signinsrvc struct {
	logger *log.Logger
}

// NewSignin returns the signin service implementation.
func NewSignin(logger *log.Logger) signin.Service {
	return &signinsrvc{logger}
}

// BasicAuth implements the authorization logic for service "signin" for the
// "basic" security scheme.
func (s *signinsrvc) BasicAuth(ctx context.Context,
	user, pass string, scheme *security.BasicScheme) (context.Context,
	error) {

	if user != "gopher" && pass != "academy" {
		return ctx, signin.
			Unauthorized("invalid username and password combination")
	}

	return ctx, nil
}

// Creates a valid JWT
func (s *signinsrvc) Authenticate(ctx context.Context, 
	p *signin.AuthenticatePayload) (res *signin.Creds, 
	err error) {

	// create JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"nbf":    time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
		"iat":    time.Now().Unix(),
		"exp":    time.Now().Add(time.Duration(9) * time.Minute).Unix(),
		"scopes": []string{"api:read", "api:write"},
	})

	s.logger.Printf("user '%s' logged in", p.Username)

	// note that if "SignedString" returns an error then it is returned as
	// an internal error to the client
	t, err := token.SignedString(Key)
	if err != nil {
		return nil, err
	}

	res = &signin.Creds{
		JWT: t,
	}

	return
}
```

Finally, because we added more fields to our custom type, we need to update the 
Add method on client Service in `client.go` to reflect such changes. Copy and 
paste the following to update your `client.go`:

```go
package clients

import (
	client "clients/gen/client"
	"context"
	"log"

	jwt "github.com/dgrijalva/jwt-go"
	"goa.design/goa/v3/security"
)

var (
	// Key is the key used in JWT authentication
	Key = []byte("secret")
)

// client service example implementation.
// The example methods log the requests and return zero values.
type clientsrvc struct {
	logger *log.Logger
}

// NewClient returns the client service implementation.
func NewClient(logger *log.Logger) client.Service {
	return &clientsrvc{logger}
}

// JWTAuth implements the authorization logic for service "client" for the
// "jwt" security scheme.
func (s *clientsrvc) JWTAuth(ctx context.Context, 
	token string, scheme *security.JWTScheme) (context.Context, 
	error) {
	
	claims := make(jwt.MapClaims)

	// authorize request
	// 1. parse JWT token, token key is hardcoded to "secret" in this example
	_, err := jwt.ParseWithClaims(token, 
		claims, func(_ *jwt.Token) (interface{}, 
		error) { return Key, nil })
	if err != nil {
		s.logger.Print("Unable to obtain claim from token, it's invalid")
		return ctx, client.Unauthorized("invalid token")
	}

	s.logger.Print("claims retrieved, validating against scope")
	s.logger.Print(claims)

	// 2. validate provided "scopes" claim
	if claims["scopes"] == nil {
		s.logger.Print("Unable to get scope since the scope is empty")
		return ctx, client.InvalidScopes("invalid scopes in token")
	}
	scopes, ok := claims["scopes"].([]interface{})
	if !ok {
		s.logger.Print("An error ocurred when retrieving the scopes")
		s.logger.Print(ok)
		return ctx, client.InvalidScopes("invalid scopes in token")
	}
	scopesInToken := make([]string, len(scopes))
	for _, scp := range scopes {
		scopesInToken = append(scopesInToken, scp.(string))
	}
	if err := scheme.Validate(scopesInToken); err != nil {
		s.logger.Print("Unable to parse token, check error below")
		return ctx, client.InvalidScopes(err.Error())
	}
	return ctx, nil

}

// Add implements add.
func (s *clientsrvc) Add(ctx context.Context, 
	p *client.AddPayload) (err error) {
	s.logger.Print("client.add started")
	newClient := client.ClientManagement{
		ClientID: p.ClientID,
		ClientName: p.ClientName,
		ContactName: p.ContactName,
		ContactEmail: p.ContactEmail,
		ContactMobile: p.ContactMobile,
	}
	err = CreateClient(&newClient)
	if err != nil {
		s.logger.Print("An error occurred...")
		s.logger.Print(err)
		return
	}
	s.logger.Print("client.add completed")
	return
}

// Get implements get.
func (s *clientsrvc) Get(ctx context.Context, 
	p *client.GetPayload) (res *client.ClientManagement, 
	err error) {
	s.logger.Print("client.get started")
	result, err := GetClient(p.ClientID)
	if err != nil {
		s.logger.Print("An error occurred...")
		s.logger.Print(err)
		return
	}
	s.logger.Print("client.get completed")
	return &result, err
}

// Show implements show.
func (s *clientsrvc) Show(ctx context.Context, 
	p *client.ShowPayload) (res client.ClientManagementCollection, 
	err error) {
	s.logger.Print("client.show started")
	res, err = ListClients()
	if err != nil {
		s.logger.Print("An error occurred...")
		s.logger.Print(err)
		return
	}
	s.logger.Print("client.show completed")
	return
}
```

And that's it! Let's recompile the application and test it again. Run the 
commands below to remove the old binaries and compile fresh ones:

```bash
rm -f clients clients-cli
go build ./cmd/clients 
go build ./cmd/clients-cli
```

Run `./clients` again and leave it running. You should see it running 
successfully - however, this time with the new methods implemented:

```bash
$ ./clients
[clients] 00:00:01 HTTP "Add" mounted on POST /api/v1/client/{ClientID}
[clients] 00:00:01 HTTP "Get" mounted on GET /api/v1/client/{ClientID}
[clients] 00:00:01 HTTP "Show" mounted on GET /api/v1/client
[clients] 00:00:01 HTTP "CORS" mounted on OPTIONS /api/v1/client/{ClientID}
[clients] 00:00:01 HTTP "CORS" mounted on OPTIONS /api/v1/client
[clients] 00:00:01 HTTP "CORS" mounted on OPTIONS /openapi.json
[clients] 00:00:01 HTTP "./gen/http/openapi.json" mounted on GET /openapi.json
[clients] 00:00:01 HTTP "Authenticate" mounted on POST /signin/authenticate
[clients] 00:00:01 HTTP "CORS" mounted on OPTIONS /signin/authenticate
[clients] 00:00:01 HTTP server listening on "localhost:8080"
```

To test, let's execute all API methods using the cli - notice that we're using 
the hardcoded credentials:

```bash
$ ./clients-cli signin authenticate \
--username "gopher" --password "academy"
{
    "JWT": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.\
eyJleHAiOjE1NzcyMTQxMjEsImlhdCI6MTU3NzIxMzU4 \
MSwibmJmIjoxNDQ0NDc4NDAwLCJzY29wZXMiOlsiY \
XBpOnJlYWQiLCJhcGk6d3JpdGUiXX0.\
tva_E3xbzur_W56pjzIll_pdFmnwmF083TKemSHQkSw"
}
$ ./clients-cli client add --body \
'{"ClientName": "Cool Company", \
"ContactName": "Jane Masters", \
"ContactEmail": "jane.masters@cool.co", \
"ContactMobile": 13426547654 }' \
--client-id "1" --token "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.\
eyJleHAiOjE1NzcyMTQxMjEsImlhdCI6MTU3NzIxMzU4MSwibmJmI\
joxNDQ0NDc4NDAwLCJzY29wZXMiOlsiYXBpOnJlYWQiLCJhcGk6d3JpdGUiXX0.\
tva_E3xbzur_W56pjzIll_pdFmnwmF083TKemSHQkSw" 
$ ./clients-cli client get --client-id "1"  \
--token "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.\
eyJleHAiOjE1NzcyMTQxMjEsImlhdCI6MTU3NzIxMzU4MSwibmJmI\
joxNDQ0NDc4NDAwLCJzY29wZXMiOlsiYXBpOnJlYWQiLCJhcGk6d3JpdGUiXX0.\
tva_E3xbzur_W56pjzIll_pdFmnwmF083TKemSHQkSw"
{
    "ClientID": "1",
    "ClientName": "Cool Company",
    "ContactName": "Jane Masters",
    "ContactEmail": "jane.masters@cool.co",
    "ContactMobile": 13426547654
}
$ ./clients-cli client show  \
--token "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.\
eyJleHAiOjE1NzcyMTQxMjEsImlhdCI6MTU3NzIxMzU4MSwibmJmI\
joxNDQ0NDc4NDAwLCJzY29wZXMiOlsiYXBpOnJlYWQiLCJhcGk6d3JpdGUiXX0.\
tva_E3xbzur_W56pjzIll_pdFmnwmF083TKemSHQkSw"
[
    {
        "ClientID": "1",
        "ClientName": "Cool Company",
        "ContactName": "Jane Masters",
        "ContactEmail": "jane.masters@cool.co",
        "ContactMobile": 13426547654
    }
]
```

And there we go! 🎉 We have a minimalist application, with proper authentication, 
scope authorization and room for evolutionary grow! After this, you could 
develop your own authentication strategy, using cloud services or any other 
identity provider of your choice. You could also create plugins for your 
preferred database, messaging system or even integrate with other APIs easily. 
Check out [Goa's github project](https://github.com/goadesign) for more plugins, 
examples (showing specific capabilities of the framework) and other useful resources.

---

That's it for today, I hope you have enjoyed playing with Goa and reading this 
article. If you have any feedback about the content, feel free to get in touch with me on [GitHub (slaterx)](https://github.com/slaterx), [Twitter (@slaterx)](https://twitter.com/slaterx) or [LinkedIn (Gleidson Nascimento)](https://linkedin.com/in/gleidsonnascimento). 
Also, we hang out in [#goa channel on Gophers Slack](https://invite.slack.golangbridge.org/), so come on over and say hi to us! 👋