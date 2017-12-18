+++
date = "2017-12-17T06:40:42Z"
linktitle = "Extending Osquery with Go"
title = "Extending Osquery with Go"
author = ["Victor Vrantchan"]
+++

What if you could use SQL to query any aspect of your infrastructure? [Osquery](https://github.com/facebook/osquery#osquery), an open source instrumentation tool released by the Facebook security team allows you to do just that.

For example, `SELECT network_name, last_connected, captive_portal FROM wifi_networks WHERE captive_portal=1;` will show all captive portal WiFi networks that a laptop has connected to.
And `SELECT * FROM processes WHERE on_disk = 0;` will show any process that is running where the binary has been deleted from disk.
When the root password vulnerability became know a few weeks ago, the osquery community [quickly crafted a query](https://twitter.com/osquery/status/935688822678892545) which would identify vulnerable macs in a fleet of devices.
With almost 200 tables available by default and support for macOS, Linux and Windows hosts, osquery is the tool of choice for many security and system administration teams.

Osquery is a powerful tool, but it’s written in C++, so why are we talking about it in a GopherAcademy post? Osquery uses Thrift (a project similar to gRPC) to allow developers to extend osquery through a series of plugin types. Earlier this year our team at [Kolide](https://kolide.com/) released a set of [Go packages](https://github.com/kolide/osquery-go) with idiomatic interfaces that allow anyone to use the full power of Go to extend osquery. In this blog post, it’s my goal to show you how you can get started with osquery development using the `osquery-go` SDK.


## Writing a custom logger plugin

When a scheduled query like `SELECT name, version from deb_packages` is executed, the `osqueryd` daemon will create a JSON log event with the results of the query. By default, a `filesystem` plugin is used, which logs the results to a local file. Commonly oquery users use aggregation tools like `filebeat` to send the result logs to a centralized log platform. Other plugins exist too. The `tls` plugin sends all logs to a remote TLS server like [Fleet](https://kolide.com/fleet). The `kinesis` plugin sends logs results to AWS, allowing advanced monitoring with applications like [StreamAlert](https://medium.com/airbnb-engineering/streamalert-real-time-data-analysis-and-alerting-e8619e3e5043). But what if you already have a well established logging pipeline with the systemd `journal`, Splunk, `fluentd` or any number of proprietary logging systems. With the Thrift bindings to osquery, you can write your own logger. Go, having support for most APIs these days, is an ideal language for implementing a logger.

For the purpose of this tutorial, we’ll implement a systemd `journal` logger. The [`go-systemd`](http://github.com/coreos/go-systemd/journal) library from CoreOS has a convenient package we can use to write to `journald`.

The [`github.com/kolide/osquery-go/plugin/logger`](https://godoc.org/github.com/kolide/osquery-go/plugin/logger) package exposes the following API which we need to implement.


    type Plugin struct {}

    type LogFunc func(ctx context.Context, typ LogType, log string) error

    func NewPlugin(name string, fn LogFunc) *Plugin

To create our own logger, we have to implement a function that satisfies the signature of `LogFunc`.

For `journald` the function looks like this:


    func JournalLog(_ context.Context, logType logger.LogType, logText string) error {
            return journal.Send(
                    logText,
                    journal.PriInfo,
                    map[string]string{"OSQUERY_LOG_TYPE": logType.String()},
            )
    }

Now we can call `logger.NewPlugin("journal", JournalLog)` to get back a functioning osquery plugin we can register with the Thrift extension server.


## Configuring osquery to use our custom extension

We have implemented a logger plugin, but we still have to link it to `osqueryd`. Osquery has a few specific requirements for registering plugins.
Plugins must be be packaged as executables, called extensions. A single extension can bundle one or more plugins. We’ll use a `package main` to create an extension.

Osquery will call our extension with 4 possible CLI flags, the most important of which is the unix socket we’ll use to communicate back to the process.


            var (
                    flSocketPath = flag.String("socket", "", "")
                    flTimeout    = flag.Int("timeout", 0, "")
                    _            = flag.Int("interval", 0, "")
                    _            = flag.Bool("verbose", false, "")
            )
            flag.Parse()

We’ll ignore the `interval` and `verbose` flag in this extension, but they still have to be parsed to avoid an error.

Next, we’ll add `time.Sleep(2 * time.Second)` to wait for the unix socket to become available. In production code we would add a retry with a backoff.

Once the extension file is available, we can bind to the socket by creating a `ExtensionManagerServer`. The extension will use the `socket` path provided to us by the osquery process.


            server, err := osquery.NewExtensionManagerServer(
                    "go_extension_tutorial",
                    *flSocketPath,
                    osquery.ServerTimeout(time.Duration(*flTimeout)*time.Second),
            )
            if err != nil {
                    log.Fatalf("Error creating extension: %s\n", err)
            }

Next, we can create our logger and register it with the server.


           journal := logger.NewPlugin("journal", JournalLog)
           server.RegisterPlugin(journal)

Finally, we can run the extension. The `server.Run()` method will block until an error is returned.


          log.Fatal(server.Run())

Now that we created our `package main`, we can build the binary and start osqueryd with the custom logger. Osquery has a few requirements for executables we have to follow:


- The executable must have a `.ext` file extension.
- The executable path should be added to an  `extensions.load` file which can be passed to the osqueryd `--extensions_autoload` CLI flag.
- The extension must be owned by the same user that is running osquery, and the permissions must be read+exec only. This is a precaution against an attacker replacing an extension executable that the `osqueryd` process runs as root. For development, you can use the `--allow_unsafe` flag, but we won’t need it here since we’ll be running the osquery process as our current user account.

Putting it all together we get:

    echo "$(pwd)/build/tutorial-extension.ext" > /tmp/extensions.load
    go build -i -o build/tutorial-extension.ext
    osqueryd \
      --extensions_autoload=/tmp/extensions.load \
      --pidfile=/tmp/osquery.pid \
      --database_path=/tmp/osquery.db \
      --extensions_socket=/tmp/osquery.sock \
      --logger_plugin=journal

Immediately we can see our logger working with `journalctl`

    sudo journalctl OSQUERY_LOG_TYPE=status -o export -f |awk -F'MESSAGE=' '/MESSAGE/ {print $2}'

    {"s":0,"f":"events.cpp","i":825,"m":"Event publisher not enabled: audit: Publisher disabled via configuration","h":"dev","c":"Mon Dec 18 03:34:31 2017 UTC","u":1513568071}
    {"s":0,"f":"events.cpp","i":825,"m":"Event publisher not enabled: syslog: Publisher disabled via configuration","h":"dev","c":"Mon Dec 18 03:34:31 2017 UTC","u":1513568071}
    {"s":0,"f":"scheduler.cpp","i":75,"m":"Executing scheduled query foobar: SELECT 1","h":"dev","c":"Mon Dec 18 03:34:38 2017 UTC","u":1513568078}


## Adding tables to osquery

Loggers are great, but what if we need to implement a custom table? Let’s stick with the `go-systemd` package and prototype a `systemd` table which will list the systemd units and their state.

The [`github.com/kolide/osquery-go/plugin/table`](https://godoc.org/github.com/kolide/osquery-go/plugin/table) package has a similar API to that of the `logger` plugin.


    type Plugin struct {}

    type GenerateFunc func(ctx context.Context, queryContext QueryContext) ([]map[string]string, error)

    type ColumnDefinition struct {
        Name string
        Type ColumnType
    }

    func NewPlugin(name string, columns []ColumnDefinition, gen GenerateFunc) *Plugin

The `ColumnDefinition` struct defines four SQL column types: `TEXT`, `INTEGER`, `BIGINT` and `DOUBLE`. To create the table, we’ll have to implement the `GenerateFunc` which returns the table as a `[]map[string]string`.

We’ll implement the required `Generate` function using the [`dbus`](https://godoc.org/github.com/coreos/go-systemd/dbus#Conn.ListUnits) package, which has a helpful `ListUnits()` method.

*Note: I’m using package globals and ignoring errors to keep the example code short. The full implementation is linked at the end of this post.*

    var conn *dbus.Conn

    func generateSystemdUnitStatus(_ context.Context, _ table.QueryContext) ([]map[string]string, error) {
            units, _ := conn.ListUnits()
            var results []map[string]string
            for _, unit := range units {
                    // get the pid value
                    var pid int
                    p, _ := conn.GetServiceProperty(unit.Name, "MainPID")
                    pid = int(p.Value.Value().(uint32))

                    // get the stdout path of the service unit
                    var stdoutPath string
                    p, _ := conn.GetServiceProperty(unit.Name, "StandardOutput")
                    stdoutPath = p.Value.String()

                    //... a few more getters like this
                    // then populate the table rows
                    results = append(results, map[string]string{
                            "name":         unit.Name,
                            "load_state":   unit.LoadState,
                            "active_state": unit.ActiveState,
                            "exec_start":   execStart,
                            "pid":          strconv.Itoa(pid),
                            "stdout_path":  stdoutPath,
                            "stderr_path":  stderrPath,
                    })
            }
        return results, nil
    }

Now we can create the osquery-go `*table.Plugin`:


    func SystemdTable() *table.Plugin {
            columns := []table.ColumnDefinition{
                    table.TextColumn("name"),
                    table.IntegerColumn("pid"),
                    table.TextColumn("load_state"),
                    table.TextColumn("active_state"),
                    table.TextColumn("exec_start"),
                    table.TextColumn("stdout_path"),
                    table.TextColumn("stderr_path"),
            }
            return table.NewPlugin("systemd", columns, generateSystemdUnitStatus)
    }

Back in our `func main`, we can register this plugin with the server, similar to how we registered the logger plugin.


    systemd := SystemdTable()
    server.RegisterPlugin(systemd)

We can now use the `systemd` service in our queries.


    osquery> SELECT process.start_time, systemd.name AS service, process.name, listening.address, listening.port, process.pid FROM processes AS process JOIN listening_ports AS listening ON (process.pid = listening.pid) JOIN systemd ON systemd.pid = process.pid and listening.port = 443;
    +------------+------------------+----------+---------+------+-------+
    | start_time | service          | name     | address | port | pid   |
    +------------+------------------+----------+---------+------+-------+
    | 6308708    | nignx.service    | nginx    | ::      | 443  | 25859 |
    +------------+------------------+----------+---------+------+-------+

By configuring the query to run on a schedule, and using the logger plugin to aggregate the results centrally, we can begin to instrument our systems and create alerts.

Speaking of configuration, how are you configuring the osquery process? The recommended way is a configuration management tool like Chef, or a dedicated TLS server like [Fleet](https://kolide.com/fleet), but maybe you’ve got *custom* requirements?


## Config plugins for osquery

Just like you can log results with a custom logger, you can load configuration through a custom plugin. We’ll implement a plugin which configures the osquery process and schedules a list of schedules queries to run. To keep things simple, we’ll load configuration from a GitHub [gist](https://gist.github.com/groob/5cfb6062eb155585f1d6adb6a3857256).

By now, you can probably guess what the API of the [`github.com/kolide/osquery-go/plugin/config`](https://godoc.org/github.com/kolide/osquery-go/plugin/config) looks like.


    type Plugin struct {}

    type GenerateConfigsFunc func(ctx context.Context) (map[string]string, error)

    func NewPlugin(name string, fn GenerateConfigsFunc) *Plugin

Here, we implement the `GenerateConfigs` function to return one or more config sources as a map, where each value represents the full config JSON file as a string.


    var client *github.Client

    func (p *Plugin) GenerateConfigs(ctx context.Context) (map[string]string, error) {
            gistID := os.Getenv("OSQUERY_CONFIG_GIST")

            gist, _, err := client.Gists.Get(ctx, p.gistID)
            if err != nil {
                    return nil, errors.Wrap(err, "get config gist")
            }
            var config string
            if file, ok := gist.Files["osquery.conf"]; ok {
                    config = file.GetContent()
            } else {
                    return nil, fmt.Errorf("no osquery.conf file in gist %s", p.gistID)
            }
            return map[string]string{"gist": config}, nil
    }

One thing I want to highlight here is that our plugin needs it’s own configuration.

    gistID := os.Getenv("OSQUERY_CONFIG_GIST")

You might need to provide configuration like API keys to your plugin, and environment variables provide a convenient way of doing that.

Now that we’ve created the plugin, one thing left to do is register it inside `func main` and restart `osqueryd`.


    gistConfig := config.NewPlugin("gist", GenerateConfigs)
    server.RegisterPlugin(gistConfig)

Restart the `osqueryd` daemon with two new flags. A refresh interval (in seconds) and the config plugin to use instead of the default `filesystem` one.

    --config_refresh=60 \
    --config_plugin=gist


## Conclusion

In the article I’ve given an overview of `osquery` and how to use the Go plugin SDK to write your own custom extensions.
Besides creating the plugins we also have to think about packaging, distribution and the platforms we’re running the osquery daemon on. For example, the `journal`  and `systemd` APIs are not available on macOS or windows, so we have to compile our custom extensions in a different way for each platform. Once again, Go makes this process easy by allowing us to use build tags when writing platform specific plugins.

At Kolide, we’ve been writing our own open source osqueryd extension called [Launcher](https://kolide.com/launcher). Launcher implements config, logger and other pugins for osquery using [gRPC](https://grpc.io/) and the [Go kit](https://gokit.io/) toolkit to effectively manage osqueryd at scale for various environments. If you’ve found this article interesting, I encourage you to check out the Launcher [source](https://github.com/kolide/launcher). The osquery has a vibrant community of users and developers, most of which [hang out on Slack](https://osquery-slack.herokuapp.com/).
In addition to the Go SDK, a similar one is available for [`python`](https://github.com/osquery/osquery-python).

I’ve described three plugin types `logger`, `table` and `config`, but there’s a fourth plugin type the `osquery-go` SDK allows you to write, and that’s a `distributed` plugin. What makes the `distributed` plugin interesting is that you can schedule queries and get query results from your whole fleet of endpoints in real time. While writing this blog post, I got the idea of implementing the distributed plugin as a [Twitter bot](https://twitter.com/querygopher). If you tweet a valid query with the `#osqueryquery` hash tag, you’ll get back a response with the results.
Although I’ve left out the implementation of this final plugin from the article, It has a very similar API to the plugins I’ve described above.

You can check out the source of all the plugins above, and a few more examples in the [Github repo](https://github.com/kolide/go-extension-tutorial) that I’ve created for this post.
