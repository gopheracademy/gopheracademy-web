+++
author = ["Mat Ryer"]
date = "2016-12-26T06:00:00+00:00"
title = "A little bit of Machine Learning: Playing with Google's Prediction API"
series = ["Advent 2016"]
draft = false
+++

Before we get started, let's begin by making clear that this isn't going to be a deep dive on
TensorFlow, neural networks, inductive logic, Bayesian networks, genetic algorithms or any other sub-heading
from the [Machine Learning Wikipedia article](https://en.wikipedia.org/wiki/Machine_learning). Nor is this
really a Go-heavy article, but rather an introduction to machine learning via a simple consumption of the 
Google Prediction API.

## How the Google Prediction API works

The Google Prediction API attempts to guess answers to questions by either predicting a numeric value between 0 and 1
for that item based on similar valued examples in its training data ("regression"), or choosing a category that describes it
given a set of similar categorized items in its training data ("categorical").

The training data is used to generate a model, and it is the model that attempts to provide answers to
future questions we ask via the Google Prediction API. In this context, *questions* refers to any input into the 
model, and *answers* refers to the expected outputs.

For example, imagine we have trained a model with the following data:

```
"This is great", "Happy"
"This is awful", "Sad"
"I love this",   "Happy"
"I hate this",   "Sad"
(plus lots more correct examples)
```

We could then to ask the Google Prediction API to tell us whether each of the following phrases are either happy or sad:

* "Awful performance"
* "Great job"
* "Absolutely love this great API"

Our human brains can do this easily, but with a sensible and appropriate set of
training data, it is possible to teach a machine to do it too with impressive results.

## Training the Prediction API

Creating training data isn't easy and is often the most delicate part of the process.
Luckily Google has provided us with a set of data that we can use.

The Language Identification dataset attempts to train a model that will allow the machine to predict
which language a given sentence is, either English, Spanish, or French.

The data is in CSV format, and looks something like this:

```
"English", "This version of the simple language detection model was created on December 27, 2011..."
"French", "M. de Troisvilles, comme s'appelait encore sa famille en Gascogne, ou M. de Tréville..."
"Spanish", "En efeto, rematado ya su juicio, vino a dar en el más estraño pensamiento que..."
...
```

Have a quick [look at the entire dataset](https://cloud.google.com/prediction/docs/language_id.txt)

As you can see, there are hundreds of examples of each language. The first column is the answer, and the
second column is an example of a question that we might ask it.

Google has already trained a model with this data, so we don't need to do it - but it's as simple as creating
a Google Project, uploading a CSV file to Google Cloud Storage, and making an API request similar to:

```
POST https://www.googleapis.com/prediction/v1.6/projects/{project-id}/trainedmodels
{
    "id": "model-identifier",
    "storageDataLocation": "path/to/training-data.csv"
}
```

Once the model is ready, we can start to ask it to categorise sentences by making requests similar to:
```
POST https://www.googleapis.com/prediction/v1.6/projects/{project-id}/trainedmodels/model-identifier/predict
{
    "input": {
        "csvInstance": [
            "Please predict which language this is"
        ]
    }
}
```

Yes, it is a little weird that we're embedding CSV into JSON, and I expect this will evolve in future versions
of the API. But it is still pretty clear what's going on; we're asking the Google Prediction API to predict which
language the `"Please predict which language this is"` sentence is in.

## Building a little app in Go

We will write a little command line tool that lets us query the model from the terminal. 

As the example project for this article was being put together, it became clear that the bulk of the work
was actually in authorising the requests via OAuth. And the experience for a terminal app was less than desirable; you 
had to copy and paste a URL into a browser, log in (or create an account), click Accept to allow access to the API, 
then copy the access code from the query parameters of a failed redirect and paste it into the terminal.

OAuth is a web based authorisation protocol through and through, and short of implementing an HTTP endpoint to
receive the request, there isn't much we can do to make the experience any better. So instead, we are going to be
a little cheeky and piggy back on [Google's Try it app hosted on App Engine](http://try-prediction.appspot.com).

In your `$GOPATH/src` folder, create a new folder called `language-predict` and add a `main.go` file.

We will write a program that reads a line from standard input, queries the model, and writes the response in a nice
human readable way.

Add the following code to `main.go`:

```go
var labels = map[string]string{
	"English": "🇬🇧 English",
	"Spanish": "🇪🇸 Spanish",
	"French":  "🇫🇷 French",
}

func main() {
	s := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	for s.Scan() {
		res, err := do(s.Text())
		if err != nil {
			fmt.Println("failed:", err)
			continue
		}
		label, ok := labels[res.OutputLabel]
		if !ok {
			label = res.OutputLabel
		}
		fmt.Printf("  ^^ That looks like %s to me\n", label)
		fmt.Print("> ")
	}
}
```

You can copy and paste the flags from here if you like :)

This code uses a `bufio.Scanner` to read a line at a time from `os.Stdin`, which allows us to type a sentence, and
hit return which will unblock the `Scan` method and execute the `for` block.

We then ask the `do` function to do its magic getting the text string via the `Text()` method on the `Scanner`,
and attempt to add a nice label (that includes a flag) to the result, before writing it out using `fmt.Print` calls.

The response we expect back from the Google Prediction API is modelled in our `response` struct that we will 
add next:

```go
type response struct {
	Kind        string
	ID          string
	OutputLabel string
	OutputMulti []struct {
		Label string
		Score string
	}
	OutputValue float64
}
```

The API gives us more information than we need, including a confidence level for each possible category. So for
very French sentences, we'd expect the confidence to be higher than the English and Spanish values. As a shortcut,
Google provides the OutputLabel string, which contains the most likely category and it is this value we use
in our `main` function above.

Finally, the `do` function is just going to make an HTTP POST request, and decode the JSON response:

```go
func do(query string) (*response, error) {
	values := url.Values{
		"model":  []string{"Language Detection"},
		"Phrase": []string{query},
	}
	res, err := http.PostForm("http://try-prediction.appspot.com/predict", values)
	if err != nil {
		return nil, err
	}
	var result response
	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
```

## Playing with our app

In a terminal, navigate to your project folder and run the program using:

```bash
go run main.go
```

Then type in an English sentence:

```bash
> Happy holidays to all you lovely Gophers
  ^^ That looks like 🇬🇧 English to me
```

Notice that the program correctly predicts the language.

Try some more:

```bash
> Joyeuses fêtes à vous mes chers Gophers
  ^^ That looks like 🇫🇷 French to me
> Felices fiestas para todos los Gophers
  ^^ That looks like 🇪🇸 Spanish to me
```

Thank you to my multi-lingual European friends for their generosity in helping me translate this phrase
in spite of Brexit.

## Conclusion 

This article only really touches the surface of what can be achieved with machine learning, 
but it does get the old brains working on what other kinds of predictive capabilities could be
achieved with a decent set of training data.

To dig deeper, it is recommended that you start exploring by visiting the
[official Google Prediction API website](https://cloud.google.com/prediction/).
