+++
author = ["Daisy Tsang"]
title = "Improve Your Breadmaking Skills with Go and Open-Source Monitoring"
linktitle = "Observability in the Kitchen"
date = "2018-12-11T00:00:00Z"
series = ["Advent 2018"]
+++


I have many different interests, including baking, open-source software, and more recently, systems monitoring and learning Go.  As a way for me to expand my practical knowledge on each item, I devised a fun little project that leverages sensors, Raspberry Pis, and [Prometheus](https://prometheus.io) to improve my sourdough breadmaking process. I will explain the inspiration behind it, go through the steps I took to set up the project, and detail what I learned.  Disclaimer: it is still a work in progress!


### Sourdough Bread and the Difficulty with Maintaining Natural Yeast
A while ago, I became interested in the art of making one’s own sourdough bread.  This is a type of bread that, due to its fermentation process, is a much healthier alternative to the store-bought bread that we are used to seeing. The naturally occurring acids and long fermentation help to break down the proteins and gluten, making it more digestible and easy for the body to absorb. 

Sourdough is the traditional way of making bread until around a hundred years ago.  The process involves cultivating your own wild yeast and allowing time for fermentation.  The sourdough bread-making process is a fascinating combination of biology and physics, but working with natural starters can be difficult since they are very sensitive to temperature and humidity. The cultures are very finicky and can easily turn to mold or not grow as fast as you would like. Parameters such as temperature and humidity need to be closely observed and monitored...perhaps with a systems monitoring tool!


### The Prometheus Project
Prometheus is an open-source systems monitoring and alerting toolkit written in Go. It is particularly suitable for containers and cloud workloads where instances can have a short lifespan, which made it very popular in the last few years. It is the second project in the CNCF to [graduate](https://www.cncf.io/announcement/2018/08/09/prometheus-graduates/), the first one being Kubernetes.  It’s ecosystem consists of the server itself, a time-series database, it’s own query language, an alert manager, client libraries, and special exporters.  It has a dimensional data model and dashboarding and alerting all based on the same query language. The Prometheus server collects time series metrics from instrumented targets, stores them and makes them query-able with their query language.  You can use the information for dashboards and alerting. 


### Observability in the Kitchen
I decided it would be fun and useful to leverage Prometheus to monitor the humidity and temperature of my sourdough starters and maybe gain insight into how temperature and humidity affects the growth of my cultures. Perhaps it can improve the breadmaking process.   

The Prometheus server scales up and down really well and can run efficiently on something like a Raspberry Pi. So I decided to use [Raspberry Pis](https://www.raspberrypi.org/products/raspberry-pi-3-model-b/) and ordered some [Bosch BME280](https://www.ebay.com/itm/Breakout-Temperature-Humidity-Barometric-Pressure-BME280-Digital-Sensor-Module-/401000227934) Temperature Humidity Barometric Pressure Sensor Modules with I2C. 

The plans is to have the Pi run a custom Prometheus node exporter, collecting temperature and humidity metrics from the sensor and making them available for scraping by the Prometheus server. 


### Setting up the Hardware
I connected the sensor to the Pi according to this diagram:

BME280 | Desc    | GPIO Header Pins
------ | ------- |------------------
VIN    | 3.3V    | P1-01
GND    | Ground  | P1-06
SCL    | I2C SCL | P1-05
SDA    | I2C SDA | P1-03

Then I installed [Raspbian](https://www.raspberrypi.org/downloads/raspbian/) on the Pi and enabled the I2C interface.  I2C is a protocol that allows one device to exchange data with one or more connected devices. 


### Prometheus Exporters
Now I need a way to export metrics in the Prometheus format for the sensor, which means that I need a Prometheus exporter. Exporters are tools that let you translate metrics from other systems into a format that Prometheus can understand. You may be able to find an exporter for your system on the Prometheus [website](https://prometheus.io/docs/instrumenting/exporters/), by searching the Internet, [mailing list](https://groups.google.com/forum/#!forum/prometheus-announce), or looking on GitHub. You can also write your own exporter. That's what I did, and while I did it in Go, it can be done in any programming language.
 

### Writing an Exporter in Go
Here is an overview of how to write a Prometheus exporter for the BME280 sensor in Go.  I need to import the [Prometheus Go client library](https://github.com/prometheus/client_golang) and I will also make use of the [Gobot framework](https://gobot.io/) in order to control the sensor and the Pi.

```
import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gobot.io/x/gobot/drivers/i2c"
	"gobot.io/x/gobot/platforms/raspi"
)
```

Then I create a collector type that can be made aware to the Prometheus client.

```
type collector struct {
	sensorDriver *i2c.BME280Driver
}
```

In Go your collectors must implement the [prometheus.Collector](https://github.com/prometheus/client_golang/blob/master/prometheus/collector.go) interface. That is to say, the collectors must be objects with the `Describe` and `Collect` methods with a specific signature.

The `Describe` method returns a description of the metrics that it will produce, in particular the metric name, label names, and help string.  This method is used to avoid duplicate registration of metrics and is called at registration time. 

```
var (
	temperatureDesc = prometheus.NewDesc("bme280_temperature_celsius", "Temperature in celsius degree", nil, nil)
	humidityDesc    = prometheus.NewDesc("bme280_humidity_percentage", "Humidity in percentage of relative humidity", nil, nil)
)

func (c collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- temperatureDesc
	ch <- humidityDesc
}
```

The `Collect` method fetches all the data you need from the application instance and send the metrics back to the client library. Prometheus client libraries offer four core metric types (counters, gauges, histograms, summaries) but only the gauge is needed for this exporter since the metric needs to represent a value that can either go up or down. These metrics will then be returned by the scrape of the `/metrics` endpoint.

Exporters don't need to constantly track the value of their metrics internally. Instead, all they need to do is provide the current value whenever they are accessed by the Prometheus server. This means that our code doesn't need to create (and update) a metric variable. Instead, each time our `Collect` method is called, it will create a throwaway constant metric holding the current value. It turns out that the Prometheus client library provides a method to precisely create such a constant metric: [MustNewConstMetric](https://github.com/prometheus/client_golang/blob/master/prometheus/value.go).

```
func (c collector) Collect(ch chan<- prometheus.Metric) {
	temperature, err := c.sensorDriver.Temperature()
	if err != nil {
		ch <- prometheus.NewInvalidMetric(temperatureDesc, err)
		return
	}
	ch <- prometheus.MustNewConstMetric(temperatureDesc, prometheus.GaugeValue, float64(temperature))

	humidity, err := c.sensorDriver.Humidity()
	if err != nil {
		ch <- prometheus.NewInvalidMetric(humidityDesc, err)
		return
	}
	ch <- prometheus.MustNewConstMetric(humidityDesc, prometheus.GaugeValue, float64(humidity))
}
```

Instantiate the Raspberry Pi adaptor and then the BME280 driver and start the driver.

```
rAdaptor := raspi.NewAdaptor()
bme280 := i2c.NewBME280Driver(rAdaptor, i2c.WithBus(1), i2c.WithAddress(0x76))

if err := bme280.Start(); err != nil {
     log.Fatalf("Error starting driver: %s", err)
}
```

Instantiate the custom collector object and register it with the default registry offered by the client library.  During scrape time, each collector registered in a registry is collected (i.e. asked for its metrics).  These metrics will be returned by the scrape of the `/metrics` endpoint. 

```
registry := prometheus.NewRegistry()
collector := &collector{sensorDriver: bme280}
registry.MustRegister(collector)
```

Then set up a HTTP handler and expose the standard `/metrics` endpoint and start listening for HTTP connections.

```
http.Handle("/metrics", promhttp.Handler())
log.Fatal(http.ListenAndServe(":8080", nil))
```

### Configuring the Prometheus Server
Before starting the server, we need to configure it.  A yaml file (`prometheus.yml`) is used to specify things such as what endpoints to scrape and how frequently to scrape it. 

```
scrape_configs:
  - job_name: 'bme280'
    scrape_interval: 10s

    static_configs:
    - targets: 
      - '192.168.0.100:8080'
      - '192.168.0.101:8080'
```

### Running the Exporter on the Raspberry Pi
With the server and the custom exporter running, Prometheus will come over the network to scrape the metrics exported from the exporter running on the Raspberry Pi that is connected to the temperature and humidity sensor. 

It would be nice to have my custom exporter start automatically upon powering up the Raspberry Pi and there are a couple of ways to enable this. I decided to create a systemd unit since it seems to be the most flexible way to manage services that run in the background on a Linux system. It allows you to run services before the desktop environment starts, wait until you have access to other processes (e.g. networking, graphical desktop), or simply restart your program over and over again until it works. 

```
[Unit]
Description=Sourdough Prometheus Exporter
After=network.target

[Install]
WantedBy=multi-user.target

[Service]
WorkingDirectory=/home/pi/go/src/github.com/sourdough-prometheus-exporter
ExecStart=/home/pi/go/src/github.com/sourdough-prometheus-exporter/sourdough-prometheus-exporter
User=pi
Restart=always
```

### Metrics Exposition
When I run the exporter and go to the `/metrics` endpoint, I see something like this:

```
# HELP bme280_humidity Humidity in percentage of relative humidity
# TYPE bme280_humidity gauge
bme280_humidity 49.532752990722656

# HELP bme280_temperature_celsius Temperature in celsius degree
# TYPE bme280_temperature_celsius gauge
bme280_temperature_celsius 21.880083084106445
```

The exposition is in a line-by-line text-based [format](https://github.com/prometheus/docs/blob/master/content/docs/instrumenting/exposition_formats.md) typically with a HELP and TYPE comment line for each metric.  

Just because metrics can be scraped does not mean that the format is compliant with the [standards](https://prometheus.io/docs/practices/naming/#metric-names).  You can run the `promtool` command-line utility that comes with Prometheus and use it to perform link checks on your metrics for consistency and correctness, by passing the metrics over stdin. 

Example:  `curl http://localhost:8080/metrics | promtool check metrics`


### Dashboarding, PromQL, Alertmanager
Once the scraped data is stored in the time-series database, we can use it to create dashboards. Grafana is a popular choice and has support for querying Prometheus.  All you have to do is create a Prometheus data source in Grafana and you can start creating graphs by querying your scraped data.

![Example Graph](https://cdn.pbrd.co/images/HQO4X2T.jpg)
[enlarge](https://cdn.pbrd.co/images/HQO4X2T.jpg)


PromQL is the Prometheus Query Language. It can help you answer a lot of ad-hoc questions about your system, but for this simple use case with one-dimensional gauges, I can just display the values as they are.

In the Prometheus ecosystem, two components are involved in alerting: Prometheus and the Alertmanager. First you define alerting rules on the Prometheus server for it to periodically evaluate and fire off to the Alertmanager. The Alertmanager takes in all the alerts from Prometheus server(s), performs logic on them and converts them to notifications. I can create alerts for when the server is down or when the temperature or humidity hits a certain mark, etc.


### Summary
This project is still a work in progress but the process has been a really fun and useful way for me to learn more about sourdough breadmaking, Raspberry Pis, IoT, and Prometheus! There are still so many things about all of those items for me to discover and I have many ideas on how to build upon it.  

Feel free to reach out to [me](https://twitter.com/1nfoverload). Happy Holidays!
