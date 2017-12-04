+++
author = ["Oklahomer"]
title = "Creating Your Own Bot Experience with go-sarah"
linktitle = "Creating Your Own Bot Experience with go-sarah"
date = 2017-12-04T00:00:00Z
series = ["Advent 2017"]
+++

Chat tools have been good friends of developers.
On chat tools we share our thoughts, problems, solutions, jokes, and pretty much everything we do as software engineers. And when some kind of tasks can be done in the extension of daily chat conversation, chat becomes even more comfortable yet powerful place to stay.
That is why I think many developers eager to customize their chat experience with bot frameworks.
In this article I am going to introduce a new golang-based bot framework, [go-sarah](https://github.com/oklahomer/go-sarah), along with its notable characteristics and components.

# Characteristics
## Conversational Context
As previously stated, we prefer to execute tasks in the extension of chat-based conversation.
To maximize that user experience, go-sarah has an idea of "conversational context," which stores current user state and defines what action to follow.

Think about when, like any other day on chat tool, you are talking with your colleagues about a critical issue reported by a user.
Everybody agrees this must be fixed before Christmas holidays.
In the extension of this conversation, on chat tool, you type in something like ".todo Fix Sarah's issue #123 by 2017-12-20 12:00:00" to register schedule.
This is somewhat handy, but user still has to input command and its arguments at once.
There is some room to improve.
With go-sarah's "conversational context" feature, you can stash currently provided arguments and still prompt user to provide unfilled argument one at a time just like the image below:

![](/postimages/creating-your-own-bot-experience-with-go-sarah/conoversational_context.png)

This is more "conversational" and user friendly. With this feature, not only can the bot let the user input arguments step by step, the bot can also validate partial user input and prompt user to re-input.
This can also be used in a way such as user and AI have conversation during when user inputs ".ai" command until user inputs something like "quit."

## Live Configuration Update
Let's say you have a command that receives user input such as ".weather San Antonio, Texas" and respond local weather.
A problem is that this command internally calls a third party web api with expirable token, and the token must be manually updated every once in a while. Do you update configuration file and reboot bot process?

With proper settings, go-sarah reads a configuration file when that file is updated and applies changes to corresponding command's configuration struct in a thread safe manner. So the bot process does not need to be rebooted just to update configuration value. Details will be covered later on this post.

## Scheduled Task
There are two kinds of executable jobs in this project: command and scheduled task.

Command is a job that is matched against user input and, if matched, executed. For example, bot says "Hello, {user name}" when user input ".hello" on chat window. This is pretty basic.

Scheduled task is the one executed in a scheduled manner without any user input. Output is sent to a predefined destination or the output itself can designate its destination chat room. Examples can be a task that sends daily weather forecast to predefined room on every morning, and a task that sends statistical data to every project room defined in a configuration struct. Thanks to the live configuration update feature, the schedule or task-specific configuration values can be updated without process reboot.

## Alerting Mechanism
Developers can register as many `sarah.Alerter` implementations as wanted to notify bot's critical states to administrators.

## Higher Customizability with Replaceable Components
Instead of trying to meet all developers' needs, this bot framework is composed of fine grained replaceable components.
Each component has its interface and at least one default implementation.
If one wishes to modify bot's behavior, one may implement corresponding interface and replace default implementation.
They each are designed to have one and only one domain to serve, so customization should require minimum effort.
More details will be covered in next section.

# Component
Now let's take a bit closer look at its major components.

![](/postimages/creating-your-own-bot-experience-with-go-sarah/component.png)

## Runner
As you can see, the Runner is orchestrating other components and letting them work together.
Runner internally has its own panic-proof worker mechanism and jobs are executed in a concurrent manner by default. Even command execution job against given user input is passed to this worker so adapter developers never have to worry about implementing own mechanism to handle flooding incoming messages.

Thanks to this design, each component can concentrate on its own domain and hence implementation can be minimal yet powerful.
I believe this should ease developers to create customized implementation of desired component.
To utilize customized component or change behavior, this project widely employs so-called [functional options](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis).
The below example adds `sarah.Alerter` implementation to notify critical state, and replaces default `sarah.Worker` with given `sarah.Worker` implementation.
```go
alerter := myapp.NewCustomizedAlerter()
worker := myapp.NewWorker()
runner, _ := sarah.NewRunner(config, sarah.WithAlerter(alerter), sarah.WithWorker(worker))
runner.Run(context.TODO())
```

Sometimes, this becomes cumbersome to initialize all components and pass to `sarah.NewRunner()` at once. Then `sarah.RunnerOptions` comes in handy.
This can be used to stash options as you code like below.
In this way you will never forget to pass customized component to `sarah.Runner`.
```go
options := sarah.NewRunnerOptions()

// Some dozens of lines to setup customized alerter come here.
options.Append(sarah.WithAlerter(myAlerter))

// Here comes another dozen of codes to initialize bot adapter.
options.Append(sarah.WithBot(sarah.NewBot(myAdapter)))

// Some more lines. blah blah blah...

// Finally initialize Runner with stashed options.
runner, _ := sarah.NewRunner(config, options.Arg())
runner.Run(context.TODO())
```

## Bot / Adapter
`sarah.Bot` interface is responsible for actual interaction with chat services such as [LINE](https://github.com/oklahomer/go-sarah-line), Slack, gitter, etc...
Or if two or more parties are messaging each other over pre-defined protocol and executing corresponding Command, such a system can be created by providing one Bot for each party just like [go-sarah-iot](https://github.com/oklahomer/go-sarah-iot) does to support communication between IoT devices and a central server.

Since `sarah.Bot` is merely an interface, anything that implements `sarah.Bot` can be passed to `sarah.Runner`.
To ease its implementation, however, common bot behaviors are already implemented by `sarah.defaultBot` and can be initialized as `sarah.Bot` by supplying `sarah.Adapter` to `sarah.NewBot`.
In this way developers only have to implement chat-tool specific messaging part.
It looks somewhat like below:
```go
// Setup slack bot.
// Any Bot implementation can be fed to Runner.RegisterBot(), but for convenience slack and gitter adapters are predefined.
// sarah.NewBot takes adapter and returns defaultBot instance, which satisfies Bot interface.
configBuf, _ := ioutil.ReadFile("/path/to/adapter/config.yaml")
slackConfig := slack.NewConfig() // config struct is returned with default settings.
yaml.Unmarshal(configBuf, slackConfig)
slackAdapter, _ := slack.NewAdapter(slackConfig)
slackBot := sarah.NewBot(slackAdapter)

runner, _ := sarah.NewRunner(sarah.NewConfig(), sarah.WithBot(slackBot))
runner.Run(context.TODO())
```

## Command
We already covered what command is.
As a matter of fact anything that implements `sarah.Command` can be a Command and be passed to `sarah.Bot`.

```go
type myCommand struct {
}

func (cmd *myCommand) Identifier() string {
	panic("implement me")
}

func (cmd *myCommand) Execute(context.Context, Input) (*CommandResponse, error) {
	panic("implement me")
}

func (cmd *myCommand) InputExample() string {
	panic("implement me")
}

func (cmd *myCommand) Match(Input) bool {
	panic("implement me")
}

slackAdapter, _ := slack.NewAdapter(slack.NewConfig())
slackBot := sarah.NewBot(slackAdapter)

slackBot.AppendCommand(&myCommand{})
```

But to have richer yet simple `sarah.Command` building experience, `sarah.CommandPropsBuilder` is provided.
This builder is so powerful that few lines of codes can create variety of `sarah.Command` with totally different behaviors.

To have the simplest form of `sarah.Command`, it goes as below:
```go
// This is a simple command that echos whenever user input starts with ".echo".
var matchPattern = regexp.MustCompile(`^\.echo`)
var SlackProps = sarah.NewCommandPropsBuilder().
        BotType(slack.SLACK).
        Identifier("echo").
        InputExample(".echo knock knock").
        MatchPattern(matchPattern).
        Func(func(_ context.Context, input sarah.Input) (*sarah.CommandResponse, error) {
                // ".echo foo" to "foo"
                return slack.NewStringResponse(sarah.StripMessage(matchPattern, input.Message())), nil
        }).
        MustBuild()
```

Sometimes you wish to match against user input in a more complex way. Then use `MatchFunc` instead of `MatchPattern` like this.
```go
var SlackProps = sarah.NewCommandPropsBuilder().
	BotType(slack.SLACK).
	Identifier("morning").
	InputExample(".morning").
	MatchFunc(func(input sarah.Input) bool {
		// 1. See if the input message starts with ".morning"
		match := strings.HasPrefix(input.Message(), ".morning")
		if !match {
			return false
		}

		// 2. See if current time between 00:00 - 11:59
		hour := time.Now().Hour()
		return hour >= 0 && hour < 12
	}).
	Func(func(_ context.Context, _ sarah.Input) (*sarah.CommandResponse, error) {
		return slack.NewStringResponse("Good morning."), nil
	}).
	MustBuild()
```

Would you like to reference some kind of configuration struct within command function, lazily initialize the configuration struct, and re-configure it when configuration file is updated?
No problem. Use `ConfigurableFunc` instead of `Func`.
```go
// When config is passed to ConfigurableFunc and if sarah.Config.PluginConfigRoot is defined,
// sarah.Runner's internal watcher, sarah.Watcher implementation, supervises config directory to re-configure on file update event.
// File located at sarah.Config.PluginConfigRoot + "/" + BotType + "/" Identifier ".(yaml|yml|json) is subject to supervise.
config := &myConfig{}
var SlackProps = sarah.NewCommandPropsBuilder().
	BotType(slack.SLACK).
	Identifier("morning").
	InputExample(".morning").
	MatchFunc(func(input sarah.Input) bool {
		return true
	}).
	ConfigurableFunc(config, func(_ context.Context, _ sarah.Input, cfg sarah.CommandConfig) (*sarah.CommandResponse, error) {
		typedConfig := cfg.(*myConfig)
		return slack.NewStringResponse(typedConfig.Foo), nil
	}).
	MustBuild()
```

On `CommandPropsBuilder.Build` or `CommandPropsBuilder.MustBuild`, this validates previous input and returns `sarah.CommandProps` that represents a non-contradicting set of arguments.
To instantiate `sarah.Command` on the fly, this props can be passed to `sarah.Runner`, not `sarah.Bot`.
Remember all components' life cycles are managed by `sarah.Runner`.
```go
var matchPattern = regexp.MustCompile(`^\.echo`)
var props = sarah.NewCommandPropsBuilder().
        BotType(slack.SLACK).
        Identifier("echo").
        InputExample(".echo knock knock").
        MatchPattern(matchPattern).
        Func(func(_ context.Context, input sarah.Input) (*sarah.CommandResponse, error) {
                // ".echo foo" to "foo"
                return slack.NewStringResponse(sarah.StripMessage(matchPattern, input.Message())), nil
        }).
        MustBuild()

runner, _ := sarah.NewRunner(config, sarah.WithCommandProps(props))
runner.Run(context.TODO())
```

Scheduled task also has a similar mechanism and detail can be found at [project wiki](https://github.com/oklahomer/go-sarah/wiki/ScheduledTask)

## UserContextStorage
I mentioned about user's "conversational context" earlier in this post. `sarah.UserContextStorage` is where conversational context is stored and currently two kinds of storages are available.

One is that stores context information in process memory space. When command response includes `sarah.UserContext`, that context information is stored in the process memory. On next input, stored `sarah.UserContext` is extracted and `UserContext.ContextualFunc()` is executed against given input. This execution is to return `sarah.CommandResponse` with or without new `sarah.UserContext` just like regular command execution.

Another one stores serialized arguments and function identifier along with user key in an external storage such as Redis. When user input for the next time, stored context information is fetched from storage, deserialized, and then executed.

The first one is handy because you can simply pass function that satisfies `func(context.Context, Input) (*CommandResponse, error)` interface, but the one with external storage is more reliable and can be used on multiple processes. There are so many available usages so take a look at  [project wiki](https://github.com/oklahomer/go-sarah/wiki/UserContextStorage) for detail.

# Wrapping Up
With introduced major components, many different bot experience can be achieved. To start with, runnable example code is located at [example codes](https://github.com/oklahomer/go-sarah/tree/master/examples).

Documents are being covered in [project top page](https://github.com/oklahomer/go-sarah) and [wiki pages](https://github.com/oklahomer/go-sarah/wiki), but not completed yet. To improve that, I would appreciate your feedbacks: github star, issue, pull request, tweet, or whatever. [I named this project after my new born daughter](http://blog.oklahome.net/2017/08/parenting-software-engineer.html) and I am so into it, so any feedback would encourage me. Thanks.
