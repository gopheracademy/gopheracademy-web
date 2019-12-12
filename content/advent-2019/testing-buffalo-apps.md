+++
date = "2019-12-27T00:00:00+00:00"
author = ["Antonio Pagano"]
title = "Testing Buffalo Applications"
linktitle = "Testing Buffalo"
+++

Buffalo is a great tool to rapidly ship software solutions to the market, inspired by the famous Ruby on Rails framework, it brought Rapid Application Development to the Go Language. I say Buffalo, by far provides the best developer experience among other web development frameworks in Go, but I'm obviously biased by being part of the Buffalo core team. Buffalo's existence has lead my team at Wawandco to deliver great apps within the last 2 years, and we are still delivering. 

Don't worry, this article is not (only) about the greatness of Buffalo, with the speed that Buffalo provides comes a lot of responsibility when it comes to developing apps. As (Uncle) Bob C. Martin says:

>
> The only way to go fast, is to go well!
>
> Bob C. Martin


Which we completely agree, apart from just delivering apps, it's important to guarantee the correctness and stability of the software we're delivering, and testing (as in TDD) is a fundamental element in GOing well.

Fortunately for us, Buffalo has been built with the same thinking in mind, the framework tooling considers testing as an important activity of the software development life cycle, and running tests in Buffalo is not a complex task. It can be summarized in:

```bash
$ buffalo test 
$ buffalo test -v
$ buffalo test ./actions/... --run ActionsSuite/MyActionTest
```

However, while the framework provides the foundational aspects to do great testing, I've found that my coworkers and I struggled at some point with understanding how to test different elements of Buffalo, and deep at the 3rd of 4th why we found that it was more about understanding the Buffalo stack, the anatomy of a Buffalo app and the responsibilities of each of these parts or a Buffalo app.

In this post I plan to explain how my team and I test Buffalo apps by going through the different layers of a typical Buffalo app.

## What’s in a Buffalo app ?

Typical Buffalo app serves web requests and renders HTML as response to it, to illustrate the layers that are part of this process, let’s take a look at the following image.

![Buffalo Stack](/postimages/advent-2019/testing-buffalo-applications/buffalo-stack.png)

And now lets go one by one of these layers to explain what that layer does and show what and how do you test each of these.

## R is for routes

Routes are defined typically in the app.go App(). This function is in charge of building the app object that would be used by the main package.

In the App() function you will see instructions like `app.GET(“/home”, homeHandler)` which associate an HTTP method, a path and a Buffalo handler.

Routes define what will be served and where will it be served. So our route tests should specify where our handlers are mounted.

```go
/* in app_test.go */
// TestRoutes is in charge of testing that our routes are placed where these should.
// When starting a new action, start by adding the test here. this should be your first test.
func (as *ActionSuite) Test_Routes() {
   routes := []struct {
       method  string
       path    string
       handler string
   }{
       {"GET", "/", "app/actions.HomeHandler"},
       // Event routes
       {"GET", "/company/{company_id}/events/", "app/actions.EventsResource.List"},
       {"POST", "/company/{company_id}/events/", "app/actions.EventsResource.Create"},
       {"GET", "/company/{company_id}/events/new/", "app/actions.EventsResource.New"},
       {"PUT", "/company/{company_id}/events/{event_id}/", "app/actions.EventsResource.Update"},
       {"GET", "/company/{company_id}/events/{event_id}/edit/", "app/actions.EventsResource.Edit"},
       {"DELETE", "/company/{company_id}/events/{event_id}/", "app/actions.EventsResource.Destroy"},
       {"PUT", "/company/{company_id}/events/{event_id}/activate/", "app/actions.EventsResource.Activate"},
   }
 
   for _, routeCase := range routes {
       testName := fmt.Sprintf("Route: %v:%v %v", routeCase.method, routeCase.path, routeCase.handler)
 
       found := false
       for _, route := range as.App.Routes() {
 
           matches := route.Method == routeCase.method
           matches = matches && route.Path == routeCase.path
           matches = matches && route.HandlerName == routeCase.handler
 
           if matches {
               found = true
               break
           }
       }
 
       as.True(found, testName+" %v", "Not found")
   }
}
```

This test will expand as you add more routes into your app, and will serve as a specification in case someone changes application routes. This could also be implicit in actions tests, however by defining these explicitly in a test for the routes we make explicit the design we’ve made for the routes of our app.

As you might have noticed we’ve not entered in detail on things like What the action do? Or which middlewares would be applied to it, those are integration tests that should happen at other layer of the stack.

## Middlewares

In Buffalo a middleware typically looks like:

```go
func AuthorizeCompany(next buffalo.Handler) buffalo.Handler {
   return func(c buffalo.Context) error {
      // do something here and then execute the next handler
       return next(c)
   }
}
```

Middlewares act as proxies for handlers, once the server receives the request it passes through each of the middlewares that apply for the given route that the request is intended to. Middlewares are the place to do things like: 

- Setup common context values for a group of actions
- Authorization and Authentication (see buffalo-auth middlewares for example)
- Condition the access to a route given request context
- Redirect the user to required forms before desired action

And when testing middlewares we need to ensure those behaviors are correct, in isolation from the app adding the middleware or not. One thing my team and I do is separating middlewares in its own package (app/middleware). In there we add our custom middlewares and with each of those tests for them.

A typical middleware test looks like:

```go
// in middlewares/authorize_company_test.go
package middleware
import (
   "net/http"
   "app/models"
 
   "github.com/gobuffalo/buffalo"
   "github.com/gobuffalo/buffalo-pop/pop/popmw"
   "github.com/gobuffalo/httptest"
)
 
var (
   actionAccessed = false
   sampleAction  = func(c buffalo.Context) error {
       actionAccessed = true
       return nil
   }
)

func (ms *MiddlewareSuite) Test_Company_Middleware() {
   ms.LoadFixture("Load companies")
   app := buffalo.New(buffalo.Options{})
   app.Use(popmw.Transaction(models.DB))
 
   actionAccessed = false
   g := app.Group("/{company_id}/")
   g.Use(AuthorizeCompany)
   g.GET("/sample", sampleHandler)
 
   ht := httptest.New(app)
   res := ht.HTML("/%v/sample", 111).Get()
   ms.Equal(false, actionAccessed)
   ms.Equal(http.StatusNotFound, res.Code)
   
  // Look for loaded company in the DB
   company := models.Company{}
   err := models.DB.Last(&company)
   ms.NoError(err)
 
   res = ht.HTML("/%v/sample", company.ID).Get()
   ms.Equal(true, actionAccessed)
   ms.Equal(http.StatusOK, res.Code)
}

```


## Helpers 

Helpers are functions we use in plush views that help us abstract things that we don’t want or cannot do in the view layer. A Buffalo helper function looks like:

```go
//HelperFormatDate is a helper function intended to be available globally to simplify date formatting on templates
func HelperFormatDate(t time.Time) string {
   date := t.Format("01/02/06")
   return date
}
```

And are added to Buffalo apps on the render.go `init()` function. 

```go
func init() {
   r = render.New(render.Options{
       // HTML layout to be used for all HTML requests:
       HTMLLayout: "application.plush.html",
 
       // Box containing all of the templates:
       TemplatesBox: templatesBox,
       AssetsBox:    assetsBox,
 
       // Add template helpers here:
       Helpers: render.Helpers{
           // for non-bootstrap form helpers uncomment the lines
           // below and import "github.com/gobuffalo/helpers/forms"
           // forms.FormKey:     forms.Form,
           // forms.FormForKey:  forms.FormFor,
           "formatDate": HelperFormatDate,
       },
   })
}
```

In the (you guessed it) Helpers property of the rendering engine (r) created there.

As the helpers are pure functions that take input and outputs you can just test these functions separately. For example:

```go
import (
   "time"
)
 
func (as *ActionSuite) Test_FormatDateHelper() {
   date := "2019-12-31"
   t, _ := time.Parse("2006-01-02", date)
 
   formattedDate := HelperFormatDate(t)
   as.Equal("12/31/19", formattedDate)
}
```

This test ensures that the resulting date is formatted accordingly.

## Actions

Actions are the C in the MVC design pattern, we join our application domain with the view in the actions, to represent a correct response to the user (or external system in the case of an API).

These are also called handlers, and its tests are typically integration tests. This is because Actions are not intended to store business logic in them (Fat controller anti-pattern), hence testing Actions will imply using the model and the view layers to test the complete operation. 

Another layer that often gets in the mix is the middleware layer, when testing actions we typically test the middlewares applied to the action in app.go.

When testing Actions we should test:

- Correct status code returned.
- Correct server side HTML/JSON generated (including conditional classes).
- Correct conditional content for search, filters and sorting.
- Correct conditional content for role based view sections.

But let’s get a bit practical here, As an example let’s consider the following action:

```go
// HomeHandler is a default handler to serve up
// a home page.
func HomeHandler(c buffalo.Context) error {
   return c.Render(200, r.HTML("index.html"))
}
```

The action itself doesn’t do much, it just renders a view (index.html). index.html looks like: 

```html
<p>Welcome to your app</p>
<%= if ( curentUser.isAdmin() ) { %>
   <p>We know you're an admin, so here is <a href="<%= adminPath() %>"> link to your admin tools</a>.</p>
<% } %>
```

And uses a layout that adds a sidebar with the following partial (see partial helper):

```html
<div class="bg-light border-right" id="sidebar-wrapper">
   <div class="sidebar-heading">App</div>
   <div class="list-group">
       <a href="/" class="bg-light <%= if (homeActive()) { %>active<% } %>">Home</a>
       <a href="/company/1/events" class="bg-light <%= if (eventsActive()) { %>active<% } %>" >Events</a>
   </div>
</div>
```

A few things need to be tested here:


- The page should be rendered with a 200 (OK) status code
- The rendering of the Welcome to your app content
- The home link <a> tag should have the “active” class when rendered
- The events link <a> tag should NOT have the “active” class when rendered
- For an admin user we should render the “We know ...” content block.
- For a regular user (not admin) we should not render the “We know ...” content block.

And in order to test this we could create a method of the `ActionSuite` struct that may look like:

```go
func (as *ActionSuite) Test_HomeHandler() {
 
   tcases := []struct {
       role       string
       contains   []string
       notContain []string
   }{
       {"USER", []string{"Welcome to your app"}, []string{"We know you're an admin, so here is"}},
       {"ADMIN", []string{"Welcome to your app", "We know you're an admin, so here is"}, []string{}},
   }
 
   for _, tcase := range tcases {
 
       as.SetRole(tcase.role)
 
       res := as.HTML("/").Get()
       as.Equal(200, res.Code)
 
       as.Contains(res.Body.String(), `<a href="/" class="bg-light active">Home</a>`)
 
       for _, content := range tcase.contains {
           as.Contains(res.Body.String, content)
       }
 
       for _, content := range tcase.notContain {
           as.NotContains(res.Body.String, content)
       }
   }
}

```

This will check most of the cases from an integration point of view. However to test that the link is not active for other pages we need another case.

```go
func (as *ActionSuite) Test_OtherPage() {
   res := as.HTML("/not/home").Get()
   as.Equal(200, res.Code)
   as.NotContains(res.Body.String(), `<a href="/" class="bg-light active">Home</a>`)
   as.Contains(res.Body.String(), `<a href="/" class="bg-light">Home</a>`)
}
```

## Models

That takes us into the last and in my opinion more important of the layers, the model. In the model is where the domain (model) of our app lives, here we would have:

- Representation of the ubiquitous language of our app in structs
- Logic for computing that secret formula that our app sells in functions
- Reporting specific queries in functions that talk to our database and return structs
- Important background procedures like saving certain events in our db for later consumption.

And the Database? Yes, model in a Buffalo app usually is connected with the database, not that the database is the center of our domain model but usually our apps will save state in a persistent storage in the shape of a database.

Tests here usually imply loading data to the database, and then doing something with the data in the database. Or passing something to a function and checking that it reacts with the correct response for what we have passed. Tests here are about the correctness in relation with our business.

Assume you’re building a payroll system, on it you would possibly have a function that computes the employees next payment:

```go
type Employee struct {
   ID           uuid.UUID `db:"id"`
   Name         string    `db:"name"`
   YearlySalary float64
}
 
// NextPay computes next employee monthly payment.
func (e Employee) NextPay() float64 {
   result := e.YearlySalary / 12.0
   result -= e.totalMonthlyReductions() //Loans and so on
   result -= e.discountableTimeoffAmount() // Non-paid Timeoff to discount.
   return result
}
```

For purposes of this article I’ve simplified what it would like, but as you may see this would be a very important piece of logic in your app. Which needs to be tested.

To do so, you write a test like:

```go
func (ms *ModelSuite) Test_ComputeMonthlySalary() {
   ms.LoadFixture("Employee")
   ms.LoadFixture("EmployeeTimeoffs")
   ms.LoadFixture("EmployeeLoans")
 
   tcases := []struct {
       id     string
       result float64
   }{
       //Knowing that these UUIDs are in the Employee fixture
       {"fc043cce-85b2-4add-8105-7d84dedf02ab", 12000},
       {"5df0536d-ccb0-444a-8a9f-4e6912a929b3", 10500.12},
   }
 
   for _, tcase := range tcases {
       id := uuid.Must(uuid.FromStringOrNil("fc043cce-85b2-4add-8105-7d84dedf02ab"))
       employee := models.Employee{
           ID: id,
       },
 
       result := employee.NextPay()
       ms.Equal(tcase.result, result)
   }
}
```

Where you load employees, time-offs and loans and then check that the employee NextPay method returns the correct value.

Fixtures, in Buffalo, are a great tool when it comes to loading data. I will not cover these and other tools like Fako in this article, but I hope to plant the seed for you to start doing a lot more testing your Buffalo apps.

## Wrapping up

As I mentioned when I started this post, Buffalo comes with testing ready for you. I explained in this post some of the layers that you should be testing in a typical Buffalo app but there are other layers that I didn’t mention (Background Tasks and Emails jump to my mind immediately).

I hope you enjoyed the read, If you have questions or comments reach me in twitter at [@paganotoni](https://twitter.com/paganotoni) or find more about my company (Wawandco) at [wawand.co](http://wawand.co) or if you’re someone visual you can check our our [dribbble profile](https://dribbble.com/wawandco).
