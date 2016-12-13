+++
author = [
  "Carlisia Pinto",
]
title = "Go and a Package Focused Design"
linktitle = "Go and a Package Focused Design"
date = "2016-12-11T00:00:00"
series = ["Advent 2016"]
+++

Developers often tend to think about designing software in terms of using logical layers of abstractions. I have seen many Go projects with layers of abstractions that reflect grouping of all common things together such as types (model), handlers for all services (api or controllers), and even multi-purpose packages (util). These ways of organizing code are not putting Go package features to good use. With Go offering purposeful tools for designing code, its long-term success rests on our ability to make good use of these features so that we end up with software that is well designed and durable. To quote [Dave Cheney](https://twitter.com/davecheney) in his Golang UK 2016 keynote talk:

> Because if Go is going to be a language that companies invest in for the long term, the maintenance of Go programs, the ease of which they can change, will be a key factor in their decision. - Dave Cheney

Taking the time to reason about each package in a project and designing each with a narrow and specific purpose will go a long way in surfacing a design that will tend towards reusability, composability, and durability.

In a very intentional way, Go packages give us a comprehensible set of ways of creating “firewalls” within our programs so various pieces are not just in a different place in the hierarchy tree, but can be made to be completely isolated, exposing only what is needed for a minimum and clean API. Here are design features of Go packages:

* **Namespacing:** allows us to choose short and clear names for types and functions in a package. We don’t need to worry if common names have already been used in other packages because packages are self-contained. <sup>[[1]](#one)</sup>

* **Encapsulation:** through the use of exported variables and functions, we control what is accessible outside of a package. This restricted visibility allows us for the possibility of having a very intentional API at the package level, and the flexibility to change unexported code without worrying about breaking that API. <sup>[[2]](#two)</sup>

* **Internal packages:** disallows the importing of code containing the element “internal” from outside the tree rooted at the parent directory of the internal directory. Code that doesn't share this common parent root directory can't import any of the packages within the internal directory. <sup>[[3]](#three)</sup>

I would like to point out possible approaches for designing packages. By no means, however, these are the only possible arrangements that would arrive at a successful and durable design.

## The Ben Johnson Way
[Ben Johnson](https://twitter.com/benbjohnson) has illustrated a design approach that will work really well especially when modeling well-defined business domains. It reflects closely the principles of Domain-Driven Design <sup>[[4]](#four)</sup>. With his approach, you isolate the packages and define a clear domain language across the entire project. These are the four rules he proposes:

* Root package is for domain types
* Group subpackages by dependency
* Use a shared mock subpackage
* Main package ties together dependencies

He talks about approaching the design from the perspective of dependencies, and uses the project packages as adapters between the domain and the different services that need to use the domain to implement the business logic. Notice how he isolates the types from the concrete implementation of the data storage, which allows for both easily swapping data storages as well as adding new ones, without the types having to change at all. There is quite more to this approach so please see the  Standard Package Layout blog post at: https://medium.com/@benbjohnson/standard-package-layout-7cdbc8391fc1#.epus9ggex

Ben also provides a very concrete example implementation using his design approach: https://github.com/benbjohnson/wtf. Note that each pull request is annotated with commentaries and with an accompanying blog post, so be sure to browse through the open and closed PRs!


## The Bill Kennedy Way
[Bill Kennedy](https://twitter.com/goinggodotnet)’s approach for designing an application would work for projects of any size. I don’t see a reason to avoid using this approach even if your project is not expected to grow in size significantly. With this approach, you primarily have three types of package levels, depending on their purpose and reusability goal. Below is my take on what he advocates:

* **Maximum reusability (shared across projects)** -
At this level, you are looking at packages that are very specific and have the highest level of reusability. They are very decoupled and only import other packages that are strictly related to their purpose. These are standard library-like packages, they share a top level directory that lives in its own repository and contains all the packages, and should be vendored into projects that use it. Any code that needs to be shared across multiple projects should be moved here.

* **External API for your project (binaries)** -
The set of binaries for the project should go under the cmd directory. This is already a very common practice in the Go ecosystem. Dependencies imported in packages under cmd are only used by these binaries.

* **Domain (domain related and reusable within the project)** -
All code that is not in one of the two categories above go inside the internal directory. This will ensure that no code outside of the project can import any of the code in this tree structure, given the compiler guarantee that such access is not possible. This restriction is helpful since it ensures that changes to the API of internal packages will never break an external application.


But it is not enough to only isolate packages as internal packages randomly. The design should be such that these packages can be reusable components throughout the application. To allow for this flexibility, Bill proposes these rules:

* Internal packages that are at the same level of the source tree are not allowed to import each other.
* Since there can be internal subpackages, an internal outer package can import any inner package.
* If an internal package needs to access a package that is not an inner package, then an outer level package should be created that can then import that functionality from both packages.


This design enforces decoupling and allows for both discoverability and reuse. Also, if it becomes obvious that any internal package seems to be of use to other projects, it will be trivial to remove it and place it in the shared common project.


The bulk of the time spent in implementation would be spent inside the internal directory, since it is where most of the business logic should reside. Also, with Bill’s approach, types are artifacts of the API: each package needs to be reusable and cannot share the types. What the cost of possibly ending up with duplicate types buys us is reusability.


For an example of this package design in action, you may look at Ardan Labs active training material on GitHub: https://github.com/ardanlabs/gotraining/tree/master/starter-kits/http


## Domain-Driven Design
Domain-Driven Design (DDD), as the name itself suggests it, demands a lot of focus on the domain model. Modern enterprise applications that are rich with intricate behavior and business logic will benefit the most from the patterns and organization options advocated by this approach.


[Marcus Olsson](https://twitter.com/marcusolsson) has set out to port the Domain-Driven Design (DDD) Sample App to Go. Besides providing a great DDD example, that sample app in Go also provides a good reference for discussing alternative package design options for any Go application. You may study the design and implementation on Marcus’ `goddd` repo on GitHub: https://github.com/marcusolsson/goddd


Marcus also wrote a series of blog posts about the experience of porting that sample app to Go, and gave a very educational talk at Golang UK 2016 on the subject:

* Domain Driven Design in Go: https://www.citerus.se/go-ddd
* Domain Driven Design in Go, Part 2: https://www.citerus.se/part-2-domain-driven-design-in-go/
* Domain Driven Design in Go, Part 3: https://www.citerus.se/part-3-domain-driven-design-in-go/
* Building an enterprise service in Go - Golang UK Conference 2016: https://www.youtube.com/watch?v=twcDf_Y2gXY


## Conclusion
Packages are at the core of designing software in Go. They are the building blocks for writing software that is composable and durable. Organizing your thoughts, designs and architectures by the packages you need is how you will have long-term success writing applications in Go. A focus on composability, reusability and durability will drive you towards effective package design. Take a package focused approach to software development in Go and you will have greater success.

---
**References**

* <a name="one"></a>[1] The Go Blog: Package names https://blog.golang.org/package-names
* <a name="two"></a>[2] Tour of Go: Esported names https://tour.golang.org/basics/3
* <a name="three"></a>[3] Go 1.4 “Internal” Packages https://docs.google.com/document/d/1e8kOo3r51b2BWtTs_1uADIA5djfXhPT36s6eHVRIvaU/
* <a name="four"></a>[4] Domain Language website: http://domainlanguage.com/ddd/
