+++
author = ["Steve Francia"]
date = "2014-12-20T10:30:19-08:00"
title = "Cobra: A Modern & Refined CLI Commander"
linktitle = "Introducing Cobra"
series = ["Advent 2014"]
+++



Go is the perfect language to develop command line applications. Go
has a few advantages that really set it apart from other languages:

1. Single binary
2. Very fast execution time, no interpreter needed
3. Go is awesome!
4. Cross platform support

Command line based applications are nearly as old as computing itself but this doesn’t
mean that they haven’t evolved. Traditional cli applications used flags to
manage the different behaviors an application could perform. Modern cli
applications have evolved to use a combination of commands, subcommands and
flags to control behavior.

As I was developing [Hugo](http://gohugo.io) I realized that I needed a
cli commander that was able to support the refined interface a modern
application like hugo requires. Though the Go tool itself utilizes a
subcommands interface, the standard library only provides a flags
package... and that package uses a non-standard (Plan 9) flag interface.

The community had provided a few options, but none fit my needs (lack of
support for nesting, non-customizable help, lack of posix compliance). Unable
to find a suitable existing library, I began work on a library
for a modern cli commander.

<img alt="Cobra"
     src="/postimages/cobra/cobra.png"
     style="float:right;"/>

# Introducing Cobra

[Cobra](http://github.com/spf13/cobra) is a commander providing a simple
interface to create powerful modern CLI interfaces similar to git & go tools.
In addition to providing an interface, Cobra simultaneously provides a
controller to organize your application code.

Inspired by cli, go, go-Commander, gh and subcommand, Cobra improves on these by
providing **fully posix compliant flags** (including short & long versions),
**nesting commands**, and the ability to **define your own help and usage** for any or
all commands.

Cobra has an exceptionally clean interface and simple design without needless
constructors or initialization methods.

# Using Cobra

Cobra is built on a structure of commands & flags.

**Commands** represent actions.

**Flags** are modifiers for those actions.

In the following example 'server' is a command and 'port' is a flag. In
this example port only applies to the server command. Cobra provides the
ability to bind a flag to a single command or to many commands.

    hugo server --port=1313


## Start with your root command

For clean source organization I recommend creating a top level directory
called `commands/` in your source tree. Files placed in here will be part of
your “commands” package. I recommend creating a new file for each
command.

We will create our root command first. The root command represents a bare call
to your application. I would create the root command in a file called
commands/root.go.

Cobra doesn't require any special constructors. Simply create a new
command:

**commands/root.go**:

    package commands

    import "github.com/spf13/cobra"

    var RootCmd = &cobra.Command{
        Use:   "name of application",
        Short: "Short description",
        Long: `Longer description.. 
                feel free to use a few lines here.
                `,
        Run: func(cmd *cobra.Command, args []string) {
            // Do Stuff Here
        },
    }

## Creating additional commands

Additional commands are defined as needed. A few note worthy things
about the example below:

1. The Use field defines how the command or subcommand will be used. The
   first word in the Use field will be used as the name of the command. 
2. This command accepts arguments and uses them.
3. Instead of defining the Run function inline we simply refer to a
   function defined on the package.
4. This command isn’t exported. It’s generally a good idea to keep your
   commands unexported.
5. In this example we are attaching the version command to the root, but commands
   can be attached at any level.


**commands/echo.go**:

    package commands

    import (
            "github.com/spf13/cobra"
            "fmt"
    )

    var cmdEcho = &cobra.Command{
        Use:   "echo [string to echo]",
        Short: "Echo anything to the screen",
        Long:  `echo is for echoing anything back.
        Echo echo’s.
        `,
        Run: echoRun,
    }

    func echoRun(cmd *cobra.Command, args []string) {
        fmt.Println(strings.Join(args, " "))
    }

    func init() {
        RootCmd.AddCommand(echoCmd)
    }

## Adding Flags

Flags provide the user a way to adjust the behavior of an application.
In Cobra a flag can be bound to a specific command or it can be bound to
a command and all of it’s subcommands.

In the following example we will demonstrate a few different ways of
creating a flag. A common way is to pass in a pointer so that you can
easily check the value the user has set the flag to. When you attach a
persistent flag to your root command it is a global flag.

**In commands/root.go**:


    var CfgFile string
    var Verbose bool


    func init() {
        RootCmd.PersistentFlags().StringVar(&CfgFile, "config", "", "config file (default is $HOME/dagobah/config.yaml)")
        RootCmd.PersistentFlags().String("mongodb_uri", "mongodb://localhost:27017/", "Uri to connect to mongoDB")
    }


You can also create a flag on a subcommand. In this example we are
binding the ‘times’ flag to the echo command. This is a local flag, unlike
the flags demonstrated above which persist to the children commands. It
is also has a short version, accessible via `--times` or `-n`

In **commands/echo.go**:

    var times int

    func init() {
        ....

        echoCmd.Flags().IntVarP(&times, "times", "n", 1, "times to echo")
    }


    func echoRun(cmd *cobra.Command, args []string) {
        for i = 0; i < times; i++ {
            fmt.Println(strings.Join(args, " "))
        }
    }

## Invoking Cobra

Once you’ve defined your commands package, you need to invoke it in your
main function.


**main.go**:

    package main

    import(
        "Path/To/Your/Application/commands"
    )

    func main() {
        commands.RootCmd.Execute()
    }


# Conclusion

The Cobra library has a bunch of other really helpful features we
haven’t discussed here. One such example is that Cobra provides a help
command out of the box. You are welcome to use Cobra’s built in help or
to define your own. See the [cobra
documentation](http://github.com/spf13/cobra) for more details.

Cobra is already being used by many go applications, including
some of the most popular go applications including kubernetes, hugo and openshift.


Some praise:

"Cobra is a really clean package. I wish my own CLI library was as good!" - Jeremy Sanz (author of CLI)

“Cobra is perfect”  - Dane Henson


We welcome contributions. Please feel free to fork the project and
help us make the best commander ever.

Cobra is a great tool to build modern cli applications, but it’s only
half of the battle. Stay tuned for part two where I introduce Cobra’s
companion, Viper.
