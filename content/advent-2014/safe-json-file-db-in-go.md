+++
author = ["Ben Schwartz"]
date = "2014-12-14T00:00:00-08:00"
title = "Using a JSON File as a Database Safely in Go"
series = ["Advent 2014"]
+++


There are definitely problems with using a json file as a database, but sometimes the simplicity
of no extra dependencies makes it an attractive option. The two biggest problems are performance 
and managing concurrent reads and writes.

We can't do much about performance, but with Go, managing concurrent reads and writes is a breeze! 

Below is a walk through of a method for managing file access so that a json file can safely be used as a database.

## Building a "TODO" service backed by a json file database

The general pattern I'll be implementing is: set up a channel and push read/write jobs onto it. Meanwhile, run a goroutine which will consume those jobs ensuring exclusive access to the json file. (Remember, Go's philosophy towards concurrency is: [Don't communicate by sharing memory; share memory by communicating](http://golang.org/doc/codewalk/sharemem/). This is why we're going to pipe requests to the component responsible for accessing the json db file rather then screwing around with lock files or synchronization.)

Follow along below where I will put it all together, or jump [here](https://github.com/benschw/jsondb-go) 
for the finished product: a full REST service for managing your todos, backed by a json file database.

### db.json
Here's what my database looks like. It's just a map of id => Todos. (I'm using a `uuid` for the Id.)

	{
	    "1a6e9148-ebe5-4bf0-9675-b76f9fab7b72": {
	        "id": "1a6e9148-ebe5-4bf0-9675-b76f9fab7b72",
	        "value": "Hello World"
	    },
	    "3e39df85-9851-4ce9-af0c-0dd831e3b970": {
	        "id": "3e39df85-9851-4ce9-af0c-0dd831e3b970",
	        "value": "Hello World2"
	    }
	}
### Todo
And here's the api model we'll be marshalling / unmarshalling it with:

	type Todo struct {
		Id    string `json:"id"`
		Value string `json:"value" binding:"required"`
	}

### main.go
In the entry point we set up our job channel, start our job processor so we're ready when the jobs
start rolling in, and then initialize a `TodoClient` which insulates us from the details of the job channel.

	db := "./db.json"
	
	// create channel to communicate over
	jobs := make(chan Job)

	// start watching jobs channel for work
	go ProcessJobs(jobs, db)

	// create client for submitting jobs / providing interface to db
	client := &TodoClient{Jobs: jobs}


### Job Processor
This is the the hub of our database. `ProcessJobs` is run as a goroutine so it just hangs out running in an infinite for loop waiting for work in the form of a `Job`.  A job's `Run` method is where the work happens: it takes in the database data (all of it! remember, this is never going to be performant with a ton of data, so let's just make things easy on ourselves and only operate on our database in its entirety) and returns the updated database data. The Job Processor then writes the modified database model back to disc before moving on to the next job. (There's also a shortcut in place where if `Run` returns nil, that means nothing was modified so we can skip the write.)

	type Job interface {
		ExitChan() chan error
		Run(todos map[string]Todo) (map[string]Todo, error)
	}

	func ProcessJobs(jobs chan Job, db string) {
		for {
			j := <-jobs

			// Read the database
			todos := make(map[string]Todo, 0)
			content, err := ioutil.ReadFile(db)
			if err == nil {
				if err = json.Unmarshal(content, &todos); err == nil {

					// Run the job
					todosMod, err := j.Run(todos)

					// If there were modifications, write them back to the database
					if err == nil && todosMod != nil {
						b, err := json.Marshal(todosMod)
						if err == nil {
							err = ioutil.WriteFile(db, b, 0644)
						}
					}
				}
			}

			j.ExitChan() <- err
		}
	}

### Read Todos Job
Here's one of our jobs for interacting with the database. This job simply implements the `Job` interface and adds in a "todos" channel so we can also return data. Since the job processor is in charge of accessing the db file, all the `Run` function does is pass the todos map to the `todo` response channel and return `nil` since there were no modifications.

	// Job to read all todos from the database
	type ReadTodosJob struct {
		todos    chan map[string]Todo
		exitChan chan error
	}

	func NewReadTodosJob() *ReadTodosJob {
		return &ReadTodosJob{
			todos:    make(chan map[string]Todo, 1),
			exitChan: make(chan error, 1),
		}
	}
	func (j ReadTodosJob) ExitChan() chan error {
		return j.exitChan
	}
	func (j ReadTodosJob) Run(todos map[string]Todo) (map[string]Todo, error) {
		j.todos <- todos

		return nil, nil
	}

### Todo Client
This is the piece which the rest of your application will interact with. It encapsulates the mess associated with pushing jobs and waiting for a response and signal to come through on the error channel. It also maps the raw database model into a more reasonable result (a slice in this case.)

	// client for submitting jobs and providing a repository-like interface
	type TodoClient struct {
		Jobs chan Job
	}

	func (c *TodoClient) GetTodos() ([]Todo, error) {
		arr := make([]Todo, 0)

		// Create and submit the job
		job := NewReadTodosJob()
		c.Jobs <- job

		// Wait for processing to complete
		if err := <-job.ExitChan(); err != nil {
			return arr, err
		}

		// Collect the found Todos
		todos <-job.todos

		// Convert Map to Slice
		for _, value := range todos {
			arr = append(arr, value)
		}
		return arr, nil
	}

## Exposing it to the web
At this point, you might be thinking: "The only reason we have to worry about concurrent writes is because you put the read/write operations in a goroutine. A single routine would provide safe reads and writes too."

But as soon as we turn this into a web service, all bets are off. Below I layer in an http server (using the [Gin](http://gin-gonic.github.io/gin/) framework) to utilize our `TodoClient` and illustrate the example.

### main.go
Same as before, but now there's a `/todo` endpoint for getting all todos

_The [full example](https://github.com/benschw/jsondb-go) is more built out with a POST, GET by id, PUT, and DELETE_

	db := "./db.json"

	// create channel to communicate over
	jobs := make(chan Job)

	// start watching jobs channel for work
	go ProcessJobs(jobs, db)

	// create dependencies
	client := &TodoClient{Jobs: jobs}
	handlers := &TodoHandlers{Client: client}

	// configure routes
	r := gin.Default()

	r.GET("/todo", handlers.GetTodos)

	// start web server
	r.Run(":8080")

### Handlers
And last but not least, we leverage the `TodoClient` to get some data... safely!

	type TodoHandlers struct {
		Client *TodoClient
	}

	// Get all todos as a slice
	func (h *TodoHandlers) GetTodos(c *gin.Context) {
		todos, err := h.Client.GetTodos()
		if err != nil {
			log.Print(err)
			c.JSON(500, "Internal Server Error")
			return
		}

		c.JSON(200, todos)
	}


## Final thoughts
Why is this better than managing lock files, synchronizing access, and more generally sharing the database across goroutines? Because those techniques impose limitations on your design. If instead you compose your app such that modifications to the shared resource are communicated to a single component (the `ProcessJobs` goroutine) responsible for modifying the resource, you've eliminated the need for sharing direct access to the database. Again, this is the Go concurrency philosophy: [Don't communicate by sharing memory; share memory by communicating](http://golang.org/doc/codewalk/sharemem/)

Thanks for following along! Check out the [full example](https://github.com/benschw/jsondb-go).

p.s. What led me down this path? We use [Composer](https://getcomposer.org/) for package management in our PHP stack and need to run a private package repo for tracking internal code. [Satis](https://github.com/composer/satis) is an excellent tool for building private Composer package repositories, but it is just a static site generator that gets its repository list from a json config file. I used the techniques above to create [Satis-Go](http://txt.fliglio.com/satis-go/) which exposes the config file as a REST api and performs the static generation of the package repository index when modifications are made. From there, adding in an admin ui and incorporating web-hooks for triggering re-indexing was straight forward.