+++
author = ["Ahsanul Haque"]
date = "2014-12-07T08:00:00+00:00"
title = "Reading config files the Go way"
series = ["Advent 2014"]
+++

In the middle of writing my blog engine [dynocator](https://github.com/ahsanulhaque/dynocator), I wondered about the best possible way to read data from a config file. My first approach was to read line by line from the file and use the wonderful [strings](http://golang.org/pkg/strings/) package to parse the data I want. Another approach revolved around using [regexp](http://golang.org/pkg/regexp/) to seek out the info from the file. But these approaches were both very hacky and involved dealing with a lot of string operations, which I'm not a big fan of.

When examining [Hugo](http://hugo.spf13.com), I realized that it reads settings data from a [TOML](https://github.com/toml-lang/toml) config file. My first impression was "Oh god, not another markup language", but as it turned out, I really like TOML. Here's some exaple TOML data:
```
# This is a TOML document. Boom.

title = "TOML Example"

[owner]
name = "Lance Uppercut"
dob = 1979-05-27T07:32:00-08:00 # First class dates? Why not?

[database]
server = "192.168.1.1"
ports = [ 8001, 8001, 8002 ]
connection_max = 5000
enabled = true

[servers]

  # You can indent as you please. Tabs or spaces. TOML don't care.
  [servers.alpha]
  ip = "10.0.0.1"
  dc = "eqdc10"

  [servers.beta]
  ip = "10.0.0.2"
  dc = "eqdc10"

[clients]
data = [ ["gamma", "delta"], [1, 2] ]

# Line breaks are OK when inside arrays
hosts = [
  "alpha",
  "omega"
]
```

I wanted to go ahead and write my own TOML parser but then I stumbled upon [this](https://github.com/BurntSushi/toml) great package. The idea is to have TOML data relate directly to Go structs. Here's some config data from my config file:
```
baseurl="http://localhost:1414"
title="My Beautiful Site"
templates="templates"
posts="posts"
public="public"
admin="admin"
metadata="metadata"
index="default"
```


And here's how to read it:
```go
// Info from config file
type Config struct {
	Baseurl   string
	Title     string
	Templates string
	Posts     string
	Public    string
	Admin     string
	Metadata  string
	Index     string
}

// Reads info from config file
func ReadConfig() Config {
	var configfile = flags.Configfile
	_, err := os.Stat(configfile)
	if err != nil {
		log.Fatal("Config file is missing: ", configfile)
	}

	var config Config
	if _, err := toml.DecodeFile(configfile, &config); err != nil {
		log.Fatal(err)
	}
	//log.Print(config.Index)
	return config
}
```

Now I can access all my config data very easily like this:
```
var config = ReadConfig()
fmt.Print(config.Title)
```