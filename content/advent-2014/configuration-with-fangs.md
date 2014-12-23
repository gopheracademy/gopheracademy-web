+++
author = ["Steve Francia"]
date = "2014-12-23T10:40:33-08:00"
series = ["Advent 2014"]
title = "Viper: Configuration with Fangs"
+++

One of the hardest things to get right when building command line or
server applications is user configuration. One look at the many
different INI formats or various approaches used in /etc demonstrates
that there really isn’t a good and standard approach. With modern
applications being used in so many different environments from the dev
environment to docker containers to cloud infrastructures it’s never
been harder to provide a consistent and appropriate solution to
configuration.


<img alt="Viper"
     src="/postimages/viper/viper.png"
     style="float:right;"/>

# Introducing Viper

[Viper](http://github.com/spf13/viper) is a complete configuration
solution for Go applications. It is a library specifically crafted to
work exceptionally well no matter the intended environment. Applications
written with Viper handle all types of configuration including
seamless integration of environment variables for 12
factor apps. 

In [my last
post](http://blog.gopheracademy.com/advent-2014/introducing-cobra/) I
introduced [Cobra](http://github.com/spf13/cobra), a modern & refined
cli commander. This post will introduce a companion library to Cobra
called Viper. While Viper and Cobra can easily be used independently,
together them make a deadly combination to provide all of your command
line needs.

It supports:

* reading from yaml, toml and json config files
* reading from environment variables
* reading from remote config systems (currently Etcd or Consul)
* using values set by command line flags

One of the best features of Viper is how easy it is to not only support
each of these configuration methods, but also all or any number of them
simultaneously. This will give your application’s users complete
flexibility in how they will use your application.

# How Viper works

Viper is, at it’s essence, a registry for all of your applications
configuration needs. I think the easiest way to think about Viper is by
looking at the two fundamental problems it solves.

1. Viper reads configuration settings from a variety of different
   sources using established standards.

2. Viper provides an simple way to access the “current” settings for an
   application regardless of how the values were set.


# Reading Configuration Settings into Viper

With very minimal configuration Viper can do the following for your application:

1. Provide a mechanism to set default values for your different
   configuration options
2. Find, load and marshal a configuration file in YAML, TOML or JSON.
3. Provide a mechanism to set override values for options specified
   through command line flags.
4. Provide an alias system to easily rename parameters without breaking
   existing code.
5. Make it easy to tell the difference between when a user has provided
   a command line or config file which is the same as the default.


Viper uses the following precedence order. Each item takes precedence
over the item below it:

 * explicit call to Set
 * flag
 * env
 * config
 * key/value store
 * default

We will walk through examples of using each of these. It’s important to
recognize that Viper does not require any initialization before using
and each different section can be called in any order.

## Setting Defaults

Our first place is a logical one. Viper permits you to set default
values for any key. A default value is not required, but can
establish a default to be used in the event that the key has not been
set via config file, environment variable, remote configuration or flag.

### Default Example:

	viper.SetDefault("ContentDir", "content")

## Reading Config Files

Viper supports reading from yaml, toml and/or json files. Viper can
search multiple paths. Paths will be searched in the order they are
provided. The following is just an example, see the full documentation
for more possibilities and details.

### Reading Config File Example:

	viper.SetConfigName("config") // name of config file (without extension)
	viper.AddConfigPath("/etc/appname/")   // path to look for the config file in
	viper.AddConfigPath("$HOME/.appname")  // call multiple times to add many search paths
	viper.ReadInConfig() // Find and read the config file

## Binding to Environment Variables:

Viper has full support for environment variables. This enables 12 factor
applications out of the box. You can either automatically read from any
ENV variables matching a key (with or without a prefix) or explicitly
bind an environment variable to a key. The latter provides a simple and
effective mechanism for ENV aliases.

_When working with ENV variables it’s important to recognize that Viper
treats ENV variables as case sensitive._

### Binding Env to Specific Keys Example:

    viper.BindEnv("port") // bind to ENV "PORT"
    viper.BindEnv("name", USERNAME) // bind to ENV "USERNAME"

	os.Setenv("PORT", "13") // typically done outside of the app
	os.Setenv("USERNAME", "spf13") // typically done outside of the app

	port := viper.GetInt("port")) // 13
	name := viper.GetString("name")) // "spf13"

## Automatic Environment Binding:

AutomaticEnv is a powerful helper especially when combined with
SetEnvPrefix. When called, Viper will check for an environment variable
any time a viper.Get request is made. It will apply the following rules.
It will check for a environment variable with a name matching the key
uppercased and prefixed with the EnvPrefix if set.

### Automatic Env Binding Example:

	viper.SetEnvPrefix("foo") // Becomes "FOO_"
	os.Setenv("FOO_PORT", "1313") // typically done outside of the app
    viper.AutomaticEnv()
	port := viper.GetInt("port")) // 1313

## Using a remote key/value configuration store

Viper will read a config string (as JSON, TOML, or YAML) retrieved from a
path in a Key/Value store such as Etcd or Consul.  These values take precedence
over default values, but are overriden by configuration values retrieved from disk, 
flags, or environment variables.

Viper uses [crypt](https://github.com/xordataexchange/crypt) to retrieve configuration
from the k/v store, which means that you can store your configuration values
encrypted and have them automatically decrypted if you have the correct
gpg keyring.  Encryption is optional.

You can use remote configuration in conjunction with local configuration, or
independently of it.

### Remote Key/Value Store Example - Unencrypted

	viper.AddRemoteProvider("etcd", "http://127.0.0.1:4001","/config/hugo.json")
	viper.SetConfigType("json") // because there is no file extension in a stream of bytes
	err := viper.ReadRemoteConfig()


## Reading from Command line Flags

Viper has the ability to bind to flags provided by the 
[Cobra](http://github.com/spf13/cobra) library though the `BindPFlag()`
method.

When you bind a flag it will set both the default value as defined by
the flag as well as the value the user provides on the command line.
Viper is smart enough to distinguish between the default and the flag
value even when they are the same and will apply the overrides properly.

Viper will bind the flag to it’s key when it is accessed. This means you
can bind as early as you want, even in an init() function.

### Viper using Cobra Flags Example:

     serverCmd.Flags().Int("port", 1138, "Port to run Application server on")
     viper.BindPFlag("port", serverCmd.Flags().Lookup("port"))

# Getting Values from Viper

Once we have set the values for our configuration we will need to access
them. Viper provides a very straightforward interface to access our
applications settings. In Viper you simply need to provide the key you
want the value for and the type you expect the value to be. Viper
provides a set of Get\_\_\_\_ methods where the blank is the type (Int,
String, etc) expected.

## Getting Values of Type from Viper

Go has a strict type system that provides some considerable advantages,
but when working with flexible typeless configurations (ENV, Flags, etc)
it can pose a challenge. Viper will transparently do it’s very best to
convert any set value into the type that you need regardless of it’s
mechanism for being set. Viper takes a conservative approach to any
conversion.

In Viper there are a few ways to get a value depending on what type of
value you want to retrieved. Viper will attempt to satisfy your type
requests. In the event that a value is not provide or the type requested
does not match the type provided viper will return the zero value for
that type.

For example if the key port has been set to the value "13" by an ENV
variable, and you call GetInt("port") it will return an integer value of 13. 
However if the ENV value "port" is set to "three" GetInt("port")
(and no default is set) will return the zero value for an int. In this
case 0.

## Checking if a Key has been Set

To check if a specific key has been set, the  IsSet() method has been
provided. This will check to see if a given key has been set via any of
the different input mechanisms.

### Retrieving Values Example:

    viper.GetString("logfile") // case insensitive Setting & Getting
	if viper.GetBool("verbose") {
        fmt.Println("verbose enabled")
	}

    if viper.IsSet("foo") {
        i := viper.GetInt("foo")
    }

## Marshaling Viper into a Struct

Viper also provides the ability for the configuration or just a nested
key within the configuration to be marshaled into a struct.

### Marshal Example:

	type config struct {
		Port int
		Name string
	}

	var C config

	err := Marshal(&C)
	if err != nil {
		t.Fatalf("unable to decode into struct, %v", err)
	}


# Conclusion

Viper is the last configuration tool you will ever need. It’s simple and
effective with a rich enough feature set to work well in any
environment. Viper is off to a great start, but there’s a lot more that
could be done. We welcome contributions! Please feel free to fork the
project and help us make the best configuration library ever.

Viper is a great tool to build modern cli applications, but it’s only
half of the battle. Check out [part
one](http://blog.gopheracademy.com/advent-2014/introducing-cobra/)
where I introduce Viper’s companion,
[Cobra](http://github.com/spf13/cobra/).
