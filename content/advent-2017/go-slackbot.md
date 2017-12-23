+++
author = ["Sebastian Borza"]
title = "Create a Slack bot with golang" 
linktitle = "Create a Slack bot with golang"
date = 2017-12-24T23:06:00Z
series = ["Advent 2017"]
+++

# Create a Slack bot with golang 

## Introduction

In this post we'll look at how to set up a quick Slack bot that receives messages (either direct or
from channel) and replies to the user. I've been an IRC user for many years and always loved setting up 
bots, whether for sports scores, weather, or something else entirely. Recently I've actually had an 
opportunity to implement my first Slack bot and figured I would document the process for others! You 
can find all of the code for this post listed [here](https://github.com/sebito91/nhlslackbot), 
and PRs are certainly welcome :D 

For this assignment we'll need a few things, not all of which are covered in this post. I invite the
reader to take a look at the installation practices for the other software dependencies based on their
specific environment needs. Here I'll be using Fedora 26 (4.14.6-200.fc26.x86_64) along with these tools:

1. ngrok for Slack API replies -- https://ngrok.com/docs#expose 
2. NHL statsapi to collect hockey scores -- https://statsapi.web.nhl.com/api/v1/schedule
3. the excellent golang slack library from nlopes -- https://github.com/nlopes/slack

You'll either need to set up an `ngrok` listener for your chosen localhost port, or develop on a server 
that it externally routable (e.g. DigitalOcean droplet). In my case here I'm developing on my laptop but 
would deploy permanently on a droplet.

## The Slack API

### Initial Configuration

The Slack [API](https://api.slack.com/slack-apps) is well flushed out and spells out what specific
payloads to anticipate for any particular object. There are a number of calls you can develop
your bot to address, but in our case here we'll look at using the Real Time Messaging API 
([RTM](https://api.slack.com/rtm)) and specifically the `chat.postMessage` and `chat.postEphemeral`
methods.

Before any of our code is working we'll need to set up an app within slack itself. Navigate to the
app registration [tool](https://api.slack.com/apps?new_app=1) to create a new application within your
workspace. Here I've created the `NHL Scores` app within my workspace.

![Create App](/postimages/advent-2017/go-slackbot/create_app.png)

Once done you'll be presented with a number of options for your new application. Here we'll need to create
a `Bot User` that will act as our listener within the workspace. My example is called `nhlslackbot` and
will be visible to all users within the workspace once mounted.

![Bot User](/postimages/advent-2017/go-slackbot/bot_user.png)

We'll need to generate an OAuth token for our user in order to actually connect with the Slack API. To do so
click on the `OAuth & Permissions` section to `Install App to Workspace` which will prompt you to authorize
access and generate the tokens you'll use. You'll need to copy the `Bot User OAuth Access Token` somewhere local,
but always make sure this is not shared anywhere! This token is secret and should be treated like your
password!

![Authorize](/postimages/advent-2017/go-slackbot/authorize.png)

Lastly we'll need to set up the `Interative Components` of our application and specify the ngrok (or other) 
endpoint that the API will send responses to. In my case, I've added a custom ngrok value here called 
`https://sebtest.ngrok.io/`. This endpoint is where we'll receive all correspondence from Slack itself, and this is how
we'll be able to process any incoming messages from the channels.

![Interactive](/postimages/advent-2017/go-slackbot/interactive.png)

With that all sorted, we can finally dig into the code!

### Code components

The crux of the code is how we handle receiving messages from the slack connection. Using the `Bot User OAuth
Access Token` to establish the initial connection, we must continuously poll the system for incoming messages.
The API gives us the ability to trigger off of a number of event types, such as:

1. Hello Events
2. Connected Events
3. Presence Change Events
4. Message Events
5. and many more

The beauty of this verbosity is that we can trigger messages on a number of different use-cases, really
giving us the ability to tailor the bot to our specific needs. For this example, we'll look at using the
`*slack.MessageEvent` type to support both indirect (within channel using `@`) or direct messages. From the 
library, The primary poll for message events leverages the `websocket` handler and just loops over events
until we've received one that we want:

```go
func (s *Slack) run(ctx context.Context) {
    slack.SetLogger(s.Logger)

    rtm := s.Client.NewRTM()
    go rtm.ManageConnection()

    s.Logger.Printf("[INFO]  now listening for incoming messages...")
    for msg := range rtm.IncomingEvents {
        switch ev := msg.Data.(type) {
        case *slack.MessageEvent:
            if len(ev.User) == 0 {
                continue
            }

            // check if we have a DM, or standard channel post
            direct := strings.HasPrefix(ev.Msg.Channel, "D")

            if !direct && !strings.Contains(ev.Msg.Text, "@"+s.UserID) {
                // msg not for us!
                continue
            }

            user, err := s.Client.GetUserInfo(ev.User)
            if err != nil {
                s.Logger.Printf("[WARN]  could not grab user information: %s", ev.User)
                continue
            }

            s.Logger.Printf("[DEBUG] received message from %s (%s)\n", user.Profile.RealName, ev.User)

            err = s.askIntent(ev)
            if err != nil {
                s.Logger.Printf("[ERROR] posting ephemeral reply to user (%s): %+v\n", ev.User, err)
            }
        case *slack.RTMError:
            s.Logger.Printf("[ERROR] %s\n", ev.Error())
        }
    }
}
```

Once we confirm that the message is indeed directed to us, we pass the event handler along to our `askIntent`
function. Remember that this is a contrived example that's just going to send back NHL game scores to the
user, iff they acknowledge that specific intent. We could build up an entire workflow around this user
interaction that would send different paths depending on user choices to our prompts, or have no prompts
at all! Those different cases are outside the scope of this introductory post, so for now we just want to 
send back a quick `Yes` v `No` prompt and handle accordingly.

To do precisely that, our handler `askIntent` will process the message and genreate an `chat.postEphemeral`
message to send back to the event user (aka the person asking for details). The "ephemeral" post is one that's
directed _only_ to the requester. Though other users will see the initial request to the bot if within the 
same channel, the subsequent interaction with the bot will only be done between the user and the bot. From
the docs:

> This method posts an ephemeral message, which is visible only to the assigned user in a specific public channel, private channel, or private conversation.

With that in mind, we set up the initial response payload using the [attachments](https://api.slack.com/docs/message-attachments) spec from the API, defining a set of actions that the user is able to choose. For this
part of the conversation the user must reply `Yes` or `No` for whether they'd like us to retrieve the most
recent scores. If `No`, we reply with a basic note and continue listening; if `Yes` then let's retrieve the 
scores!

```go
// askIntent is the initial request back to user if they'd like to see
// the scores from the most recent slate of games
//
// NOTE: This is a contrived example of the functionality, but ideally here
// we would ask users to specify a date, or maybe a team, or even
// a specific game which we could present back
func (s *Slack) askIntent(ev *slack.MessageEvent) error {
    params := slack.NewPostEphemeralParameters()
    attachment := slack.Attachment{
        Text:       "Would you like to see the most recent scores?",
        CallbackID: fmt.Sprintf("ask_%s", ev.User),
        Color:      "#666666",
        Actions: []slack.AttachmentAction{
            slack.AttachmentAction{
                Name:  "action",
                Text:  "No thanks!",
                Type:  "button",
                Value: "no",
            },
            slack.AttachmentAction{
                Name:  "action",
                Text:  "Yes, please!",
                Type:  "button",
                Value: "yes",
            },
        },
    }

    params.Attachments = []slack.Attachment{attachment}
    params.User = ev.User
    params.AsUser = true

    _, err := s.Client.PostEphemeral(
        ev.Channel,
        ev.User,
        slack.MsgOptionAttachments(params.Attachments...),
        slack.MsgOptionPostEphemeralParameters(params),
    )
    if err != nil {
        return err
    }

    return nil
}
```

The attachments in the snippet above present the user with the following dialog:

![Options](/postimages/advent-2017/go-slackbot/options.png)

If the user selects `No, thanks!` then we reply with a basic message:

![Choose No](/postimages/advent-2017/go-slackbot/no.png)

This part of the interaction is precisely where the `ngrok` endpoint comes into play. The user's interaction
is not directly with our code, but instead with slack itself. The message and interaction is passed through
slack and on to us at the redirect URL we specified earlier, in my case `https://sebtest.ngrok.io` which
routes to our internal `localhost:9191` interface, and from there to our `postHandler` as defined in our
webapp router.

The tricky part here is to process the `payload` portion of the JSON response from the API. The `POST` that
slack returns back to our URL is a payload form that contains a bevy of information for our interaction. In
this case, the user's response (either `Yes` or `No`) as well as a `callbackID` which we actually passed
in our original mesage prompt to the user! This is incredibly useful, especially as you have more and more 
users interacting with your bot as you can specify unique actions based on the trigger. For example,
if the user selects `Yes` we could send subsequent ephemeral messages to ask for a specific date, or maybe
a certain team? We could even define the callback value as a key to a function map that would then trigger
some kind of other workflow altogether (like posting to a blog resource, or checking DB credentials, etc). The
options are indeed endless, but for the scope of this contrived example we just stick
to the scores from last night.

```go
func postHandler(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/" {
        w.WriteHeader(http.StatusNotFound)
        w.Write([]byte(fmt.Sprintf("incorrect path: %s", r.URL.Path)))
        return
    }

    if r.Body == nil {
        w.WriteHeader(http.StatusNotAcceptable)
        w.Write([]byte("empty body"))
        return
    }
    defer r.Body.Close()

    err := r.ParseForm()
    if err != nil {
        w.WriteHeader(http.StatusGone)
        w.Write([]byte("could not parse body"))
        return
    }

    // slack API calls the data POST a 'payload'
    reply := r.PostFormValue("payload")
    if len(reply) == 0 {
        w.WriteHeader(http.StatusNoContent)
        w.Write([]byte("could not find payload"))
        return
    }

    var payload slack.AttachmentActionCallback
    err = json.NewDecoder(strings.NewReader(reply)).Decode(&payload)
    if err != nil {
        w.WriteHeader(http.StatusGone)
        w.Write([]byte("could not process payload"))
        return
    }

    action := payload.Actions[0].Value
    switch action {
    case "yes":
        grabStats(w, r)
    case "no":
        w.Write([]byte("No worries, let me know later on if you do!"))
    default:
        w.WriteHeader(http.StatusNotAcceptable)
        w.Write([]byte(fmt.Sprintf("could not process callback: %s", action)))
        return
    }

    w.WriteHeader(http.StatusOK)
}
``` 

A key component to note here is the `http` response code; if you do not specify the `http.StatusOK` value
in your prompt back to the API, the error message you may want to convey to the user gets eaten by the system. 
The default slackbot will absorb that message and reply to you (with an `ephemeral` message no less) with the 
status code, but not the messages. Long story short, whatever message you'd like to actually send back to
the requester should have an `http.StatusOK` header.

Lastly, if our user has selected the `Yes` option we call out to our NHL stats api and process the results
for the user!

```go
// grabStats will process the information from the API and return the data to
// our user!
func grabStats(w http.ResponseWriter, r *http.Request) {
    n := fetch.New()

    buf, err := n.GetSchedule()
    if err != nil {
        w.WriteHeader(http.StatusNoContent)
        w.Write([]byte(fmt.Sprintf("error processing schedule; %v", err)))
        return
    }

    w.WriteHeader(http.StatusOK)
    w.Write(buf)
}

// GetSchedule calls out to the NHL API listed at APIURL
// and returns a formatted JSON blob of stats
//
// This function calls the 'schedule' endpoint which
// returns the most recent games by default
// TODO: add options to provide date range
func (n *NHL) GetSchedule() ([]byte, error) {
    var buf bytes.Buffer

    r, err := http.Get(fmt.Sprintf("%s/schedule", APIURL))
    if err != nil {
        return buf.Bytes(), err
    }
    defer r.Body.Close()

    err = json.NewDecoder(r.Body).Decode(&n.Schedule)
    if err != nil {
        return buf.Bytes(), fmt.Errorf("error parsing body: %+v", err)
    }

    for _, x := range n.Schedule.Dates {
        for idx, y := range x.Games {
            buf.WriteString(fmt.Sprintf("Game %d: %s\n", idx+1, y.Venue.Name))
            buf.WriteString(fmt.Sprintf("Home: %s -- %d\n", y.Teams.Home.Team.Name, y.Teams.Home.Score))
            buf.WriteString(fmt.Sprintf("Away: %s -- %d\n\n", y.Teams.Away.Team.Name, y.Teams.Away.Score))
        }
    }

    return buf.Bytes(), nil
}
```

Sample output below...

![Sample Output](/postimages/advent-2017/go-slackbot/yes.png)

Congratulations, you've now delivered an ephemeral payload to your slack user's request!

-------

## About The Author

Sebastian Borza is a golang, C, and python developer based in Chicago, IL. 

Source   | Handle
---------|--------
freenode | sborza 
efnet    | sebito91
github   | [sebito91](https://github.com/sebito91)
twitter  | [@sebito91](https://twitter.com/sebito91)
keybase  | [sborza](https://keybase.io/sborza)
GPG      | [E4110D3E](https://pgp.cs.uu.nl/stats/37447f3fe4110d3e.html)
