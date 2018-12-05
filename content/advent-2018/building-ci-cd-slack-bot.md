+++
author = ["Stephen Afam-Osemene"]
title = "Building a CI/CD Bot with Slack and Kubernetes."
linktitle = "Building a CI/CD Bot on Slack and Kubernetes."
date = 2018-12-05
series = ["Advent 2018"]
+++

This article is about an experiment at [Africa's Talking](https://africastalking.com) on using Slack to manage our deployment process.

Like many companies, we use Kubernetes to manage our deployments, and Slack for internal communications. We decided to investigate how we can use Slack to improve our deployment process and structure the communications needed for a deployment.

## The Structure

At Africa's Talking, each project is written with comprehensive unit testing, also, before a change is approved, several people have to take a look at it and do some manual testing themselves. We'll refer to these people as the QA team.

After the change is approved, the project can only be deployed to production by a few people. We'll call these people the project owners.

## The Flow

The way our deployment process used to work was this:

1. A commit is made to the Git repository
2. Our CI tool does some automated testing.
3. If the tests pass, the QA team is notified so they can do their manual tests (This may involve building and running the project locally).
4. If all is well, the new version is then deployed to production.

## The Goal

This project is meant to help improve the flow and make it this:

1. A commit is made to the Git repository
2. Our CI tool does some automated testing.
3. The CI tool should then build a Docker image with a unique tag and push to a company registry
4. Once all the tests pass, the CI tool will then notify our Bot.
5. The bot deploys the new docker image to a specific internal URL based on the branch/tag that was committed to.
6. The bot would inform the QA team on Slack. Each of the people in the QA team would then be able to approve or reject the change through Slack.
7. If any of the QA team approves, the project leader would be notified on Slack.
8. When the leader wants to, they will deploy to production, also through Slack.

## Proof of Concept

### Creating our Slack bot

* Let us register our Bot with Slack [here](https://api.slack.com/apps).

![Create a Slack App](/postimages/advent-2018/building-ci-cd-slack-bot/create-slack-app.png)

* Next, we need to register the permission scope `chat:write:bot`.

![Slack App Permissions](/postimages/advent-2018/building-ci-cd-slack-bot/slack-scopes.png)

* Now we set up a request URL for interactions. Let's put that at `http://our-bot-url.test/slack-interactions`.

![Slack Interactions Request URL](/postimages/advent-2018/building-ci-cd-slack-bot/slack-interactions-request-url.png)

* Once all those are done, we install the app into our workspace and copy the slack token.

### Structuring our Slack Client

Let us describe the Slack objects in Go. We will create structs for the slack message object, and what we receive from Slack when an interaction happens.
To send out our messages, we need to use the token we got after we installed our application to our Slack workspace.

In this example, I'm using [Viper](https://github.com/spf13/viper) to load `slackToken` from the config. Later during the `init()` function, we'll set up [Viper](https://github.com/spf13/viper) so we load the config from the right place.

```go
// slack.go
package main

import (
    "bytes"
    "encoding/json"
    "errors"
    "io/ioutil"
    "net"
    "net/http"
    "time"

    "github.com/spf13/viper"
)

type SlackMessage struct {
    Channel     string            `json:"channel,omitempty"`
    Text        string            `json:"text,omitempty"`
    Attachments []SlackAttachment `json:"attachments,omitempty"`
    User        string            `json:"user,omitempty"`
    Ts          string            `json:"ts,omitempty"`
    ThreadTs    string            `json:"thread_ts,omitempty"`
    Update      bool              `json:"-"`
    Ephemeral   bool              `json:"-"`
}

type SlackAttachment struct {
    Title      string        `json:"title,omitempty"`
    Fallback   string        `json:"fallback,omitempty"`
    Fields     []SlackField  `json:"fields,omitempty"`
    CallbackID string        `json:"callback_id,omitempty"`
    Color      string        `json:"color,omitempty"`
    Actions    []SlackAction `json:"actions,omitempty"`
}

type SlackField struct {
    Title string `json:"title,omitempty"`
    Value string `json:"value,omitempty"`
    Short bool   `json:"short,omitempty"`
}

type SlackAction struct {
    Name    string            `json:"name,omitempty"`
    Text    string            `json:"text,omitempty"`
    Type    string            `json:"type,omitempty"`
    Value   string            `json:"value,omitempty"`
    Confirm map[string]string `json:"confirm,omitempty"`
    Style   string            `json:"style,omitempty"`
    URL     string            `json:"url,omitempty"`
}

// SlackInteraction is a struct that describes what we
// would receive on our interactions endpoint from slack
type SlackInteraction struct {
    Type        string            `json:"type,omitempty"`
    Actions     []SlackAction     `json:"actions,omitempty"`
    CallbackID  string            `json:"callback_id,omitempty"`
    Team        map[string]string `json:"team,omitempty"`
    Channel     map[string]string `json:"channel,omitempty"`
    User        map[string]string `json:"user,omitempty"`
    MessageTs   string            `json:"message_ts,omitempty"`
    OrigMessage SlackMessage      `json:"original_message,omitempty"`
}

// sendSlack() sends a slack message.
// It expects that viper can find "slackToken" in the config file.
func sendSlack(message SlackMessage) (response []byte, err error) {

    netTransport := &http.Transport{
        Dial: (&net.Dialer{
            Timeout: 5 * time.Second,
        }).Dial,
        TLSHandshakeTimeout: 5 * time.Second,
    }

    netClient := &http.Client{
        Timeout:   time.Second * 10,
        Transport: netTransport,
    }

    slackMessage, err := json.MarshalIndent(message, "", "  ")
    if err != nil {
        return
    }

    slackToken := viper.GetString("slackToken")
    slackBytes := bytes.NewBuffer(slackMessage)

    endpoint := "https://slack.com/api/chat.postMessage"
    if message.Update && message.Ts != "" {
        endpoint = "https://slack.com/api/chat.update"
    } else if message.Ephemeral && message.User != "" {
        endpoint = "https://slack.com/api/chat.postEphemeral"
    }

    req, err := http.NewRequest("POST", endpoint, slackBytes)
    req.Header.Add("Authorization", "Bearer "+slackToken)
    req.Header.Add("Content-Type", "application/json")
    if err != nil {
        return
    }

    resp, err := netClient.Do(req)
    if err != nil {
        return
    }

    response, err = ioutil.ReadAll(resp.Body)
    if err != nil {
        return
    }

    type SlackResponse struct {
        Ok    bool
        Error string
    }

    var slackR SlackResponse
    err = json.Unmarshal(response, &slackR)
    if err != nil {
        return
    }

    if !slackR.Ok {
        err = errors.New(slackR.Error)
    }

    return
}
```

### Creating the Server

I like to create a server struct to hold all the dependencies. It is an idea I borrowed from [here](https://medium.com/statuscode/how-i-write-go-http-services-after-seven-years-37c208122831).

Here is what my server struct for this will look like.

```go
// server.go
package main

import (
    "net/http"
)

type server struct {
    Router       http.Handler
    Handlers     Handlers
    Projects     map[string]Project
    Builds       chan Build
    Interactions chan SlackInteraction
}

func NewServer() (*server, error) {
    s := &server{}

    s.Builds = make(chan Build, 5)
    s.Interactions = make(chan SlackInteraction, 5)
    s.Projects = make(map[string]Project)
    s.Handlers = make(map[string]func() http.HandlerFunc)
    s.load()

    return s, nil
}

func (s *server) load() {
    s.startProcessors() // to read from the channels
    s.addProjects()     // all our projects
    s.addHandlers()     // the handlers for our routes
    s.addRoutes()       // Setting up the routes
}
```

The methods we've called in `load()` will all be located elsewhere. We'll see those in the following sections.

Now we can write our `main.go`.

```go
// main.go
package main

import (
    "fmt"
    "log"
    "net/http"

    "github.com/spf13/viper"
)

// We use viper here to load configuration from a config.yml file
func setupConfig() {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    err := viper.ReadInConfig()
    if err != nil {
        log.Println(err)
        return
    }
}

func init() {
    setupConfig()
}

func main() {

    s, err := NewServer()
    if err != nil {
        panic(err)
    }

    http.Handle("/", s.Router)
    fmt.Println("listening on port 80")

    err = http.ListenAndServe(":80", s.Router)
    if err != nil {
        panic(err)
    }
}
```

Here, we asked `viper` to load the config from `config.yml` in the same directory, but in practice, the `config` could be loaded from a different place.
Take a look at viper's [documentation](https://github.com/spf13/viper) if you'd like to modify that.

### Defining our Projects

So far we can see that we need the following details for each project

1. A unique ID.
2. A name for display purposes.
3. A base URL for all deployments.
4. A Slack channel for general information about the project.
4. The QA team. A list of Slack IDs.
5. The project leaders. A list of Slack IDs.

We can define this in our Go code like this. We will also define the `addProjects()` method of our server.

```go
// projects.go
package main

import (
    "log"

    "github.com/spf13/viper"
)

type Project struct {
    ID      string
    Name    string
    URL     string
    Channel string
    QA      []string
    Owners  []string
}

// This loads projects defined in the config to the server
func (s *server) addProjects() {
    var projects []Project

    err := viper.UnmarshalKey("projects", &projects)
    if err != nil {
        log.Println(err)
        return
    }

    for _, p := range projects {
        s.Projects[p.ID] = p
    }
}
```

The `addProjects()` method expects a `projects` key in the `config.yml` which would look something like this:

```yaml
projects:
  - id: "test1"
    name: "Testing One"
    url: "https://example.org"
    channel: "ChannelID"
    qa: ["QA ID 1", "QA ID 2"]
    owners: ["Owner ID 1", "Owner ID 2"]
  - id: "test2"
    name: "Testing Two"
    url: "https://example.org"
    channel: "ChannelID"
    qa: ["QA ID 1", "QA ID 2"]
    owners: ["Owner ID 1", "Owner ID 2"]
```

### Setting up build notifications

To avoid vendor lock-in, we decided not to use the native webhooks of any platform.
Instead, we will add a final step to our CI service. This step will be to make a POST request to our Bot with these parameters:

1. The project id
2. The docker image that was built
3. The build type (branch or tag)
4. The target (name of branch or tag)

On our bot, we can listen for that at `/build-complete`. The request will be something like this 

```
curl -X POST \
  http://our-bot-url.test/build-complete \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -H 'Postman-Token: edcea5c8-9a04-4373-9246-45e18deefa25' \
  -H 'cache-control: no-cache' \
  -d 'project=test&image=registry%2Fusername%2Fimage%3A0.1.0&type=branch&target=staging'
```

In our code, we can represent each build with a struct like this: 

```go
type Build struct {
    Project Project
    Target  string
    Image   string
    Type    string
}
```

### The Deployments

To deploy we're going to make use of Kubernetes. Thankfully, there is an amazing [Go client](https://github.com/kubernetes/client-go) to help us interact with our cluster.

We're going to write a couple of functions that take a `Build` struct and does the deployment. One of them will be to deploy to auto-generated URLs, the other to deploy to production.

In this case, I am using [Ambassador](https://www.getambassador.io/) for routing. It is assumed that Ambassador is already set up on the cluster, so I will only need to add a service with the [proper annotations](https://www.getambassador.io/user-guide/getting-started#5-adding-a-service) to set up my routing to the deployment.

Here are the functions: 

```go
// deploy.go
package main

import (
    "net/url"

    "github.com/spf13/viper"
    appsv1 "k8s.io/api/apps/v1"
    apiv1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/util/intstr"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
)

// deploy() takes a Build, deploys and return the URL
// We have to generate an ID and the appropriate URL first
func deploy(build Build) (URL string, err error) {

    u, err := url.Parse(build.Project.URL)
    if err != nil {
        return
    }
    u.Host = build.Target + "." + build.Type + "." + u.Host
    URL = u.String()

    Id := build.Project.ID + "-" + build.Type + "-" + build.Target
    labels := map[string]string{
        "project":     build.Project.ID,
        "target":      build.Target,
        "type":        build.Type,
        "environment": "qa",
    }

    err = deployToUrl(build.Image, Id, URL, labels)
    return
}

// deployToProd() is to depoly a build to production
// Unlike deploy(), it does not add any sepcial identifiers
// to the url or ID.
func deployToProd(build Build) (URL string, err error) {

    u, err := url.Parse(build.Project.URL)
    if err != nil {
        return
    }
    URL = u.String()

    Id := build.Project.ID
    labels := map[string]string{
        "project":     build.Project.ID,
        "environment": "production",
    }

    err = deployToUrl(build.Image, Id, URL, labels)
    return
}

// deployToUrl() is the generic deploy function.
// Improvements to be made:
//     Allow flixibility in defining ports, resources and replicas
func deployToUrl(Image, Id, URL string,
    labels map[string]string) (err error) {

    config, err := clientcmd.BuildConfigFromFlags(
        "",
        viper.GetString("KubeConfigPath"),
    )

    if err != nil {
        return
    }
    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        return
    }

    svcClient := clientset.CoreV1().Services(apiv1.NamespaceDefault)
    service := &apiv1.Service{
        ObjectMeta: metav1.ObjectMeta{
            Name: Id,
            Annotations: map[string]string{
                "getambassador.io/config": ` |
                      ---
                      apiVersion: ambassador/v0
                      kind:  Mapping
                      name:  ` + Id + `
                      host: ` + URL + `
                      service: ` + Id,
            },
        },
        Spec: apiv1.ServiceSpec{
            Selector: labels,
            Ports: []apiv1.ServicePort{
                {
                    Port:       80,
                    TargetPort: intstr.FromInt(80),
                },
            },
        },
    }

    depClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)
    deployment := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name: Id,
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: int32Ptr(1),
            Strategy: appsv1.DeploymentStrategy{
                Type: appsv1.RollingUpdateDeploymentStrategyType,
            },
            Selector: &metav1.LabelSelector{
                MatchLabels: labels,
            },
            Template: apiv1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: labels,
                },
                Spec: apiv1.PodSpec{
                    Containers: []apiv1.Container{
                        {
                            Name:  Id,
                            Image: Image,
                            Ports: []apiv1.ContainerPort{
                                {
                                    ContainerPort: 80,
                                },
                            },
                        },
                    },
                },
            },
        },
    }

    _, getErr := svcClient.Get(Id, metav1.GetOptions{})
    if getErr != nil {
        switch getErr.(type) {
        case *errors.StatusError:
            if getErr.(*errors.StatusError).Status().Code == 404 {
                _, err = svcClient.Create(service)
                if err != nil {
                    return
                }
            }

        default:
            err = getErr
            return
        }
    }

    _, getErr = depClient.Get(Id, metav1.GetOptions{})
    if getErr != nil {
        switch getErr.(type) {
        case *errors.StatusError:
            if getErr.(*errors.StatusError).Status().Code == 404 {
                _, err = depClient.Create(deployment)
                if err != nil {
                    return
                }
            }

        default:
            err = getErr
            return
        }
    } else {
        _, updateErr := depClient.Update(deployment)
        if updateErr != nil {
            err = updateErr
            return
        }
    }

    return
}

func int32Ptr(i int32) *int32 { return &i }
```

*NOTE:* `Viper` expects to find "KubeConfigPath" in the `config.yml` file.

### Routing and Handlers

Now let's define our handlers. We'll also create a `use()` method to load handlers by name or return an error if the handler is not registered.

```go
// handlers.go
package main

import (
    "encoding/json"
    "errors"
    "fmt"
    "log"
    "net/http"
)

type Handlers map[string]func() http.HandlerFunc

// The is used to load a handler by name
// It sends a http 500 error if the handler does not exist
func (h Handlers) use(name string) http.HandlerFunc {
    method, ok := h[name]
    if ok == false {
        return func(w http.ResponseWriter, r *http.Request) {
            http.Error(w, "Handler not found", 500)
        }
    }
    return method()
}

func (s *server) addHandlers() {

    s.Handlers["404"] = func() http.HandlerFunc {
        // This handles 404 errors
        return func(w http.ResponseWriter, r *http.Request) {
            http.Error(w, "404: Page not found", 404)
        }
    }

    s.Handlers["BuildComplete"] = func() http.HandlerFunc {
        // This handles build notifications and sends them
        // into the server Builds channel
        return func(w http.ResponseWriter, r *http.Request) {

            projectName := r.FormValue("project")
            project, ok := s.Projects[projectName]
            if !ok {
                log.Println(errors.New("Project " + projectName + " not found"))
                http.Error(w, "Error encountered", 500)
                return
            }

            build := Build{}
            build.Project = project
            build.Image = r.FormValue("image")   //docker image
            build.Target = r.FormValue("target") // name of the branch or tag
            build.Type = r.FormValue("type")     // branch or tag

            s.Builds <- build
            w.Write([]byte("Received successfully"))
        }
    }

    s.Handlers["SlackInteractions"] = func() http.HandlerFunc {
        // This handles slack interactions and sends them
        // into the server Interactions channel
        return func(w http.ResponseWriter, r *http.Request) {

            var theResp SlackInteraction
            bodyBytes := []byte(r.FormValue("payload"))

            err := json.Unmarshal(bodyBytes, &theResp)
            if err != nil {
                log.Println(err)
                return
            }

            switch theResp.Type {
            case "interactive_message":
                s.Interactions <- theResp
                return
            default:
                fmt.Println("Unknown Interaction", string(bodyBytes))
            }
        }
    }
}
```

Now we can add our routes. I've used the [Chi Router](https://github.com/go-chi/chi) here, but anything that implements the `http.Handler` interface will work too.

```go
// routes.go
package main

import (
    "github.com/go-chi/chi"
)

func (s *server) addRoutes() {
    r := chi.NewRouter()
    r.NotFound(s.Handlers.Use("404")) // A route for 404s
    r.Post("/build-complete", s.Handlers.Use("BuildComplete"))
    r.Post("/slack-interactions", s.Handlers.Use("SlackInteractions"))
    s.Router = r
}
```

### The Processors

The processors would read from the streams and act on them.
We need 2 processors:

* A "build processor" to act on a new build notification.
* An "interaction processor" which will act on interactions from slack.

We'll begin by invoking processors in goroutines like this:

```go
func (s *server) startProcessors() {
    go s.buildProcessor()
    go s.interactionProcessor()
}

func (s *server) buildProcessor() {
    for build := range s.Builds {
        go func() {
            // handle the build
        }()
    }
}

func (s *server) interactionProcessor() {
    for interaction := range s.Interactions {
        go func() {
            // handle the slack interaction
        }()
    }
}
```

#### The Build Processor

When a build comes in, what we want to achieve is this:

* First, a notification is sent to the general channel that a build was completed and a deployment will then we attempted.

![Attempting Deployment](/postimages/advent-2018/building-ci-cd-slack-bot/attempting-deployment.png)

* If the deployment fails, the previous message is updated to let everyone know. Also, the failure reason (the error given when the deployment was attempted) is shown.

![Deployment Failed](/postimages/advent-2018/building-ci-cd-slack-bot/deployment-failed.png)

* If the deployment was successful, the general message is updated to let everyone know. A link to the deployed project is also given.

![Deployment Successful](/postimages/advent-2018/building-ci-cd-slack-bot/deployment-successful.png)

* The project owners will then be sent personal messages which in addition to the link will show:
    * The users that are to do QA
    * A button to deploy the build to production
    * A button to close the build

![Owner Message](/postimages/advent-2018/building-ci-cd-slack-bot/owner-message.png)

* The members of the QA team are then sent personal messages which will have the following:
    * A link to the project
    * A button to approve the build 
    * A button to reject the build

![Perform QA](/postimages/advent-2018/building-ci-cd-slack-bot/perform-qa.png)

Our final `buildProcessor()` came to look like this:

```go
//-------------------------------------
func (s *server) buildProcessor() {
    for build := range s.Builds {
        go func() {

            ts, attemptErr := sendAttemptDeployMessage(build)
            if attemptErr != nil {
                log.Println(attemptErr)
                return
            }

            url, deployErr := deploy(&build)

            if deployErr != nil {
                log.Println(deployErr)

                failErr := sendFailedDeployMessage(build, ts, deployErr)
                if failErr != nil {
                    log.Println(failErr)
                }
                return
            }

            err := sendDeploySuccessMessage(build, ts, url)
            if err != nil {
                log.Println(err)
                return
            }
            
            // payload is used to hold the information
            payload, errs := sendOwnerMessages(build, url)
            if len(errs) > 0 {
                log.Println(errs)
                return
            }

            errs = sendQaMessages(build, url, payload)
            if len(errs) > 0 {
                log.Println(errs)
                return
            }

        }()
    }
}
//----------------------------------
```

The tricky part was storing the details of the build without having to resort to using a database, to keeping it in memory where it can be lost upon restart.

In particular, the details that need to be persisted are:

1. The build details that was sent. The docker image, branch/tag, and target
2. The `ts`(timestamps use to identify slack messages) of the messages that were sent to the owners. This is necessary so we can inform the owners when any member of the QA team approves/rejects the build or if a different owner deploys or closes the build.

To "store" these details, we can make use of the `value` parameter of Slack message actions.
From the [documentation](https://api.slack.com/docs/interactive-message-field-guide#action_fields).

>Provide a string identifying this specific action. It will be sent to your Action URL along with the `name` and attachment's `callback_id`. If providing multiple actions with the same name, `value` can be strategically used to differentiate intent. Your `value` may contain up to 2000 characters.

As you can see, when a button is clicked, we will be sent the `name`, `value` and `callback_id`. However, `value` is the only one that allows up to 2000 character. This should be enough to hold the details.

What I did was to create a `payload` struct like this: 

```go
type actionPayload struct {
    Build         Build      `json:"build,omitempty"`
    OwnerMessages []ownerMsg `json:"owner_messages,omitempty"`
}

type ownerMsg struct {
    Owner   string `json:"owner,omitempty"`
    Ts      string `json:"ts,omitempty"`
    Channel string `json:"channel,omitempty"`
}
```

And then `marshal` this into JSON and use as the value for all the actions. We can separate intent using the name field.

Now, we need to send the Owner message to get all the necessary details for the payload, but we also need to include the payload in the owner message actions, I ended up sending twice. The first with just the regular details, and then update it with the `payload` and the owner actions.

Here is the final function: 

```go
//----------------------------------
func sendOwnerMessages(build Build, url string) (
    payload actionPayload, errs []error) {
    var oMsgs []ownerMsg

    successMessage := getDeploySuccessMessage(build, url)

    for _, user := range build.Project.Owners {
        successMessage.Channel = user
        resp, err := sendSlack(successMessage)
        if err != nil {
            errs = append(errs, err)
            continue
        }

        respMap := make(map[string]string)
        json.Unmarshal(resp, &respMap)

        oMsgs = append(oMsgs, ownerMsg{
            Owner:   user,
            Ts:      respMap["ts"],
            Channel: respMap["channel"],
        })
    }

    payload = actionPayload{
        Build:         build,
        OwnerMessages: oMsgs,
    }

    OwnerMessage, err := getOwnerMessage(build, url, payload)
    if err != nil {
        errs = append(errs, err)
        return
    }

    for _, oM := range payload.OwnerMessages {
        OwnerMessage.Update = true
        OwnerMessage.Channel = oM.Channel
        OwnerMessage.Ts = oM.Ts

        _, err := sendSlack(OwnerMessage)
        if err != nil {
            errs = append(errs, err)
            continue
        }
    }

    return
}
//----------------------------------
```

For brevity, I will not show the other functions here, but if you're interested, be sure to check out the [git repository](https://github.com/stephenafamo/ci-bot).

#### The Interaction Processor

When an interaction comes in, it is first sorted. We currently have only 2 types of interactions, identified by their `callback_id`. "QA Response" or "Deploy Decision".

So, our `interactionProcessor()` can look like this:

```go
//-------------------------------------
func (s *server) interactionProcessor() {
    for interaction := range s.Interactions {
        go func() {
            switch interaction.CallbackID {
            case "QA Response":
                go s.handleQaResponse(interaction)
            case "Deploy Decision":
                go s.handleOwnerDeploy(interaction)
            }
        }()
    }
}
//-------------------------------------
```

When handling the QA response, we have to do the following:

1. Update the QA message to remove the options and state what was chosen.
    ![QA Approved Self](/postimages/advent-2018/building-ci-cd-slack-bot/qa-approved-self.png)
    ![QA Rejected Self](/postimages/advent-2018/building-ci-cd-slack-bot/qa-rejected-self.png)
2. Inform the owners of the verdict from that QA member.
    ![QA Approved Thread](/postimages/advent-2018/building-ci-cd-slack-bot/qa-approved-thread.png)
    ![QA Rejected Thread](/postimages/advent-2018/building-ci-cd-slack-bot/qa-rejected-thread.png)


Here is the final function: 

```go
//-------------------------------------
func handleQaResponse(action SlackInteraction) {

    user := action.User["id"]
    channel := action.Channel["id"]

    var payload actionPayload
    err := json.Unmarshal([]byte(action.Actions[0].Value), &payload)
    if err != nil {
        log.Println(err)
        return
    }

    var newAttch SlackAttachment

    switch action.Actions[0].Name {
    case "approve":
        newAttch = SlackAttachment{
            Title:    "Approved",
            Fallback: "Approved",
            Color:    "good",
        }
    case "reject":
        newAttch = SlackAttachment{
            Title:    "Rejected",
            Fallback: "Rejected",
            Color:    "danger",
        }
    }

    updtMsg := action.OrigMessage
    updtMsg.Channel = channel
    updtMsg.Ts = action.MessageTs
    updtMsg.Update = true
    updtMsg.Attachments = updtMsg.Attachments[:2]
    updtMsg.Attachments = append(updtMsg.Attachments, newAttch)

    _, err = sendSlack(updtMsg)
    if err != nil {
        log.Println(err)
        return
    }

    // Create a new slack message and add it as a threaded
    // reply to the Owner messages
    var newM SlackMessage
    newM.Text = "<@"+ user +"> has *"+ newAttch.Title +"* this build"

    for _, oM := range payload.OwnerMessages {
        newM.ThreadTs = oM.Ts
        newM.Channel = oM.Channel

        _, err = sendSlack(newM)
        if err != nil {
            log.Println(err)
            continue
        }
    }

    return
}
//-------------------------------------
```

Handling the "Deploy Decision" is somewhat more complicated, so let's split that further.

```go
//-------------------------------------
func handleOwnerDeploy(action SlackInteraction) {

    var errs []error

    switch action.Actions[0].Name {
    case "deploy":
        errs = handleDeployToProd(action)
    case "close":
        errs = handleCloseDeployment(action)
    }

    if len(errs) > 0 {
        log.Println(errs)
        return
    }

    return
}
//-------------------------------------
```

If the deployment is closed, we'll do the following:

* Remove the "Deploy/Close" buttons from all the Owner messages.
* Add an attachment to the owner messages showing that it has been closed.

![Deployment Closed](/postimages/advent-2018/building-ci-cd-slack-bot/deployment-closed.png)

Our `handleCloseDeployment()` function will look like this:

```
//-------------------------------------
func handleCloseDeployment(action SlackInteraction) (errs []error) {

    var payload actionPayload
    err := json.Unmarshal([]byte(action.Actions[0].Value), &payload)
    if err != nil {
        errs = append(errs, err)
        return
    }

    updateMessage := action.OrigMessage
    updateMessage.Update = true
    updateMessage.Attachments = updateMessage.Attachments[:3]
    updateMessage.Attachments = append(updateMessage.Attachments,
        SlackAttachment{
            Title:    "Closed",
            Fallback: "Closed",
            Color:    "danger",
        },
    )

    for _, oM := range payload.OwnerMessages {
        updateMessage.Channel = oM.Channel
        updateMessage.Ts = oM.Ts

        _, err = sendSlack(updateMessage)
        if err != nil {
            errs = append(errs, err)
            continue
        }
    }

    return
}
//-------------------------------------
```
When a production deployment is triggered, we first attempt to deploy.

* If the deploy fails, we send an announcement to the channel with the failure reason.

![Production Deployment Failed](/postimages/advent-2018/building-ci-cd-slack-bot/production-deployment-failed.png)

* If the deploy is successful, we will:
    * Send an announcement to the channel that a new deployment to production was done and who triggered it.
    * Remove the "Deploy/Close" buttons from all the Owner messages.
    * Add an attachment to the owner messages showing the deploy decision.

![Deployed to Production](/postimages/advent-2018/building-ci-cd-slack-bot/deployed-to-production.png)
![New Production Deployment](/postimages/advent-2018/building-ci-cd-slack-bot/new-production-deployment.png)

Our `handleDeployToProd()` function will look like this:

```go
//-------------------------------------
func handleDeployToProd(action SlackInteraction) (errs []error) {

    var payload actionPayload
    err := json.Unmarshal([]byte(action.Actions[0].Value), &payload)
    if err != nil {
        errs = append(errs, err)
        return
    }

    url, deployErr := deployToProd(payload.Build)

    if deployErr != nil {
        errs = append(errs, deployErr)

        failErr := sendFailedProdDeploy(payload, deployErr)
        if failErr != nil {
            errs = append(errs, failErr)
        }
        return
    }

    updtMsg := action.OrigMessage
    updtMsg.Update = true
    updtMsg.Attachments = updtMsg.Attachments[:3]
    updtMsg.Attachments = append(updtMsg.Attachments,
        SlackAttachment{
            Title:    "Deployed to production",
            Fallback: "Deployed to production",
            Color:    "good",
        },
    )

    for _, oM := range payload.OwnerMessages {
        updtMsg.Channel = oM.Channel
        updtMsg.Ts = oM.Ts

        _, err = sendSlack(updtMsg)
        if err != nil {
            errs = append(errs, err)
            continue
        }
    }

    err = sendSuccessProdDeploy(payload, action.User["id"], url)
    if err != nil {
        errs = append(errs, err)
    }

    return
}
//-------------------------------------
```

There are some functions not shown here, if you'd like to see how it was implemented, check out the [git repository](https://github.com/stephenafamo/ci-bot).

## Conclusion

We did it! We've built a Proof-of-Concept slack bot that allows us to manage our deployment process within Slack.
It was a nice process and I'll definitely be looking to make it more production ready.

I'd definitely love to get thoughts and contributions on this, so please leave a comment here, or you can contact me through any of these ways:

Source   | Handle
---------|--------
Website  | [stephenafamo.com](https://stephenafamo.com)
Twitter  | [@StephenAfamO](https://twitter.com/stephenafamo)
GitHub   | [stephenafamo](https://github.com/stephenafamo)
Slack    | [StephenAfamO](https://gophers.slack.com/)
LinkedIn | [Stephen Afam-Osemene](https://www.linkedin.com/in/stephenafamo/)
