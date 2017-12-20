+++
author = ["Marko Mudrinić"]
date = "2017-12-27T00:00:00-08:00"
title = "Using Go Templates"
series = ["Advent 2017"]
+++

Go templates are powerful method to customize output however you want, whatever you're creating a web page, sending e-mail, working with [Buffalo](https://github.com/gobuffalo/buffalo), [Go-Hugo](https://gohugo.io), or just using some CLI such as [kubectl](https://kubernetes.io/docs/reference/kubectl/overview/).

There're two packages operating with templates — [`text/template`](https://golang.org/pkg/text/template/) and [`html/template`](https://golang.org/pkg/html/template/). Both provide the same interface, however the `html/template` package is used to generate HTML output safe against code injection.

In this article we're going to take a quick look on how to use them package, as well as how to integrate them with your application.

### Actions

Before we learn how to implement it, let's take a look at template's syntax. Templates are provided to the appropriate functions either as string or as "raw string". **Actions** represents the data evaluations, functions or control loops. They're delimited by `{{ }}`. Other, non delimited parts are left untouched.

#### Data evaluations

Usually, when using templates, you'll bind them to some data structure (e.g. `struct`) from which you'll obtain data. To obtain data from a `struct`, you can use the `{{ .FieldName }}` action, which will replace it with `FieldName` value of given struct, on parse time. The struct is given to the `Execute` function, which we'll cover later.

There's also `{{.}}` action that you can use to refer to value of non-struct types.

#### Conditions

You can also use `if` loops in templates. For example, you can check is `FieldName` non-empty, and if it's, print its value: `{{if .FieldName}} Value of FieldName is {{ .FieldName }} {{end}}`. 

`else` and `else if` are also supported: `{{if .FieldName}} // action {{ else }} // action 2 {{ end }}`.

#### Loops

Using the `range` action you can loop through a slice. A range actions is defined using the `{{range .Member}} ... {{end}}` template.

If your slice is non-struct type, you can refer to the value using the `{{ . }}` action. In case of structs, you can refer to the value using the `{{ .Member }}` action, as already explained.

#### Functions, Pipelines and Variables

Actions have several built-in functions that're used along with pipelines to additionally parse output. Pipelines are annotated with `|` and default behavior is sending data from left side to the function on right side.

Functions are used to escape the action's result. There're several functions available by default such as, `html` which returns HTML escaped output, safe against code injection or `js` which returns JavaScript escaped output.

Using the `with` action, you can define variables that are available in that `with` block: `{{ with $x := <^>result-of-some-action<^> }} {{ $x }} {{ end }}`.

Throughput the article, we're going to cover more complex actions, such as reading from array instead of struct.

## Parsing Templates

The three most important and most frequently used functions are:

* `New` — allocates new, undefinied template,
* `Parse` — parses given template string and return parsed template,
* `Execute` — applies parsed template to the data structure and write result to the given writer.

The following code shows above-mentioned functions in the action:

```go
package main

import (
	"os"
	"text/template"
)

type Todo struct {
	Name        string
	Description string
}

func main() {
	td := Todo{"Test templates", "Let's test an template to see the magic."}

  t, err := template.New("todos").Parse("You have task named \"{{ .Name}}\" with description: \"{{ .Description}}\"")
	if err != nil {
		panic(err)
	}
	err = t.Execute(os.Stdout, td)
	if err != nil {
		panic(err)
	}
}
```

The result is the following message printed in your terminal:

```
You have task named "Test templates" with description: "Let's test an template to see the magic."
```

You can reuse the same template, without needing to create or parse it again by providing the struct you want to use to the `Execute` function again:

```go
// code omitted beacuse of brevity
...

tdNew := Todo{"Go", "Contribute to any Go project"}
err = t.Execute(os.Stdout, tdNew)
}
```

The result is like the previous one, just with new data:

```
You have task named "Go" with description: "Contribute to any Go project"
```

As you can see, templates provide powerful way to customize textual output. Beside manipulating textual output, you can also manipulate HTML output using the `html/template` package.

### Verifying Templates

`template` packages provide the `Must` functions, used to verify is template correct on parsing time. `Must` function provides the same result as if we manually checked for the error, like in the previous example. 

This approach saves you typing, but if you encounter an error, your application will panic. For advanced error handling, it's easier to use above solution instead of `Must` function.

The `Must` function takes an template and error as arguments. It's common to provide `New` function as an argument to it:

```go
t := template.Must(template.New("todos").Parse("You have task named \"{{ .Name}}\" with description: \"{{ .Description}}\""))
```

Througput the article we're going to use this function so we can omit explicitly error checking.

Once we know how what Template interface provides, we can use it in our application. Next section of the article will cover some practical use cases, such as creating web pages, sending e-mails or implementing it with your CLI.

## Implementing Templates

In this part of the article we're going to take a look how you can use the magic of templates. Let's start by creating a simple HTML page, containing an to-do list.

#### Creating Web Pages using Templates

The `html/template` package allows you to provide template file, e.g. in the form of an HTML file, to make implementing both the front-end and back-end part easier.

The following data structure represents a To-Do list. The root element has the user's name and list, which is represented as array of struct containing taks's name and status.

```go
type entry struct {
  Name string
  Done bool
}

type ToDo struct {
  User string
  List []entry
}
```

This simple HTML page will be used to display user's name and its To-Do list. For this example, we're going to use `range` action to loop through tasks slice, `with` action to easier get data from slice and an condition checking is task already done. In case task is done, `Yes` will be written in the appropriate field, otherwise `No` will be written.

```html
<!DOCTYPE html>
<html>
  <head>
    <title>Go To-Do list</title>
  </head>
  <body>
    <p>
      To-Do list for user: {{ .User }} 
    </p>
    <table>
      	<tr>
          <td>Task</td>
          <td>Done</td>
    	</tr>
      	{{ with .List }}
			{{ range . }}
      			<tr>
              		<td>{{ .Name }}</td>
              		<td>{{ if .Done }}Yes{{ else }}No{{ end }}</td>
      			</tr>
			{{ end }} 
      	{{ end }}
    </table>
  </body>
</html>
```

Just like earlier, we're going to parse template and then apply it to the struct containing our data. Instead of the `Parse` function, the `ParseFile` is going to be used. Also, for code brevity, we'll write parsed data to standard output (your terminal) instead to an HTTP Writer.

```go
package main

import (
	"html/template"
	"os"
)

type entry struct {
	Name string
	Done bool
}

type ToDo struct {
	User string
	List []entry
}

func main() {
	// Parse data -- omited for brevity

	// Files are provided as a slice of strings.
	paths := []string{
		"todo.tmpl",
	}

    t := template.Must(template.New("html-tmpl").ParseFiles(paths...))
	err = t.Execute(os.Stdout, todos)
	if err != nil {
		panic(err)
	}
}
```

This time, we're using `html/template` instead of `text/template`, but as they provide the same interface, we're using same functions to parse the template. That output would be same even if you used `text/template`, but this output is safe against code injection.

This code generates code such as the below one:

```html
<!DOCTYPE html>
<html>
  <head>
    <title>Go To-Do list</title>
  </head>
  <body>
    <p>
      To-Do list for user: gopher 
    </p>
    <table>
      	<tr>
          <td>Task</td>
          <td>Done</td>
    	</tr>
      			<tr>
              		<td>GopherAcademy Article</td>
              		<td>Yes</td>
      			</tr>			
      			<tr>
              		<td>Merge PRs</td>
              		<td>No</td>
      			</tr>
    </table>
  </body>
</html>
```

#### Parsing Multiple Files

Sometimes, this approach is not suitable if you have many files, or you're dynamically adding new ones or removing old ones

Beside `ParseFiles` function, there's `ParseGlob` function which takes glob as an argument and than parses all files that matches the glob.

```go
// ...
t := template.Must(template.New("html-tmpl").ParseGlob("*.tmpl"))
err = t.Execute(os.Stdout, todos)
if err != nil {
	panic(err)
}
// ...
```

#### Use cases

## Conclusion

