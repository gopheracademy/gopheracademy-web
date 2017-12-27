+++
author = ["Orr Chen"]
title = "Simple messaging framework using Go TCP server and Kafka"
linktitle = "messaging framework"
date = 2017-12-31T00:00:00Z
series = ["Advent 2017"]
+++  
  
![System diagram](/static/postimages/advent-2017/messaging-framework/system-diagram.png)  
I needed to create a simple framework to provide my endpoint devices ( doesn't matter which platform they run on ) the option to send and receive messages from my backend.  
I require those messages to be managed by a message broker so that they can be processed in an asynchronous way.  
The system contains 4 main layers, this article section is mainly about the first one:  
1. **TCP servers** - Needs to maintain as many TCP sockets in synch with the endpoints as possible. All of the endpoints messages will be processed on a different layer by the message broker. This keeps the TCP servers layer very thin and effective. I also want to keep as many concurrent connection as possible, and Go is a good choice for this ( see [this article](https://medium.freecodecamp.org/million-websockets-and-go-cc58418460bb))  
2. **Message broker** - Responsible for delivering the messages between the TCP servers layer and the workers layer. I chose [Apache Kafka](https://kafka.apache.org/) for that purpose.  
3. **Workers layer** - will process the messages through services exposed in the backend layer.  
4. **Backed services layer** - An encapsulation of services required by your application such as DB, Authentication, Logging, external APIs and more.  
  
So, this Go Server:  
1. communicates with its endpoint clients by TCP sockets.  
2. queues the messages in the message broker.  
3. receives back messages from the broker after they were processed and sends response acknowledgment and/or errors to the TCP clients.  

The full source code is available in : https://github.com/orrchen/go-messaging  
I have also included a Dockerfile and a build script to push the image to your Docker repository.  
Special thanks to the great go Kafka [sarama library from Shopify](https://github.com/Shopify/sarama).

_The article is divided to sections representing the components of the system. Each component should be decoupled from the others in a way that allows you to read about a single component in a straight forward manner._

## [TCP Client](https://github.com/orrchen/go-messaging/blob/1c5f38664ee822b1e8ebd29560bd41fcdd1c12c3/lib/client.go) 
Its role is to represent a TCP client communicating with our TCP server.
```go
type Client struct {
	Uid  string /* client is responsible of generating a unique uid for each request,   
	it will be sent in the response from the server so that client will know what request generated this response */
	DeviceUid string /* a unique id generated from the client itself */
	conn net.Conn
	onConnectionEvent func(c *Client, eventType ConnectionEventType, e error) /* function for handling new connections */
	onDataEvent func(c *Client, data []byte) /* function for handling new date events */
}
```
Please notice that `onConnectionEvent` and `onDataEvent` are callbacks for the Struct that will obtain and manage Clients.

Our client will listen permanently using the `listen()` function and response to new connections, new data received and connections terminations.

## [Kafka Consumer](https://github.com/orrchen/go-messaging/blob/1c5f38664ee822b1e8ebd29560bd41fcdd1c12c3/messages/consumer.go)

Its role is to consume messages from our Kafka broker, and to broadcast them back to relevant clients by their uids.  
In this example we are consuming from multiple topics using the [cluster implementation of sarama](github.com/bsm/sarama-cluster).

Let's define our `Consumer` struct:  
```go
type Consumer struct {
	consumer *cluster.Consumer
	callbacks ConsumerCallbacks
}
```

The constructor receives the callbacks and relevant details to connect to the topic:
```go
func NewConsumer(callbacks ConsumerCallbacks,brokerList []string, groupId string, topics []string) *Consumer {
	consumer := Consumer{callbacks:callbacks}

	config := cluster.NewConfig()
	config.ClientID = uuid.NewV4().String()
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	saramaConsumer, err := cluster.NewConsumer(brokerList, groupId, topics, config)
	if err != nil {
		panic(err)
	}
	consumer.consumer = saramaConsumer
	return &consumer

}
```
It will consume permanently on a new goroutine inside the `Consume()` function.  
It reads from the `Messages()` channel for new messages and the `Notifications()` channel for events.

## [Kafka Producer](https://github.com/orrchen/go-messaging/blob/1c5f38664ee822b1e8ebd29560bd41fcdd1c12c3/messages/producer.go)
Its role is to produce messages to our Kafka broker.  
In this example we are producing to a single topic.  
This section is mainly inspired from the example in https://github.com/Shopify/sarama/blob/master/examples/http_server/http_server.go

Let's define our `Producer` Struct:
```go
type Producer struct {
	asyncProducer sarama.AsyncProducer
	callbacks     ProducerCallbacks
	topic         string
}
```

`Producer` is constructed with the callbacks for error, and the details to connect to the Kafka broker including optional ssl configurations that are created with `createTLSConfiguration`:
```go
func NewProducer(callbacks ProducerCallbacks,brokerList []string,topic string,certFile *string,keyFile *string,caFile *string,verifySsl *bool ) *Producer {
	producer := Producer{ callbacks: callbacks, topic: topic}

	config := sarama.NewConfig()
	tlsConfig := createTLSConfiguration(certFile,keyFile,caFile,verifySsl)
	if tlsConfig != nil {
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = tlsConfig
	}
	config.Producer.RequiredAcks = sarama.WaitForLocal       // Only wait for the leader to ack
	config.Producer.Compression = sarama.CompressionSnappy   // Compress messages
	config.Producer.Flush.Frequency = 500 * time.Millisecond // Flush batches every 500ms

	saramaProducer, err := sarama.NewAsyncProducer(brokerList, config)
	if err != nil {
		log.Fatalln("Failed to start Sarama producer:", err)
		panic(err)
	}
	go func() {
		for err := range saramaProducer.Errors() {
			if producer.callbacks.OnError!=nil {
				producer.callbacks.OnError(err)
			}
		}
	}()
	producer.asyncProducer = saramaProducer
	return &producer
}
```

I decided to produce messages that are encoded to JSON and to ensure it before sending them:
```go
type message struct {
	value interface{}
	encoded []byte
	err     error
}

func (ale *message) ensureEncoded() {
	if ale.encoded == nil && ale.err == nil {
		ale.encoded, ale.err = json.Marshal(ale.value)
		if ale.err!=nil {
			log.Println(ale.err)
		}
	}
}

func (ale *message) Length() int {
	ale.ensureEncoded()
	return len(ale.encoded)
}

func (ale *message) Encode() ([]byte, error) {
	ale.ensureEncoded()
	return ale.encoded, ale.err
}
```
And finally, we provide the functions to produce the message and close the producer:
```go
func (p *Producer) Produce(payload interface{}) {
	value := message{
		value: payload,
	}
	value.ensureEncoded()
	log.Println("producing: ", string(value.encoded))
	p.asyncProducer.Input() <- &sarama.ProducerMessage{
		Topic: p.topic,
		Value: &value,
	}
}  
func (p *Producer) Close() error{
	log.Println("Producer.Close()")
	if err := p.asyncProducer.Close(); err != nil {
		return err
	}
	return nil
}
```
## [TCP Server](https://github.com/orrchen/go-messaging/blob/1c5f38664ee822b1e8ebd29560bd41fcdd1c12c3/lib/tcp_server.go)

Its role is to obtain and manage a set of `Client`, and send and receive messages from them.
```go
type TcpServer struct {
	address                  string // Address to open connection: localhost:9999
	connLock sync.RWMutex
	connections map[string]*Client
	callbacks Callbacks
	listener net.Listener
}
```
It is constructed simply with an address to bind to and the callbacks to send:
```go
// Creates new tcp Server instance
func NewServer(address string, callbacks Callbacks ) *TcpServer {
	log.Println("Creating Server with address", address)
	s := &TcpServer{
		address: address,
		callbacks: callbacks,
	}
	s.connections = make(map[string]*Client)
	return s
}
```
When a connection event occurs we process it and handle it, if it's a new event we attach a new UID to the client.  
If connection is terminated we delete this client.  
In both cases we send the callbacks to notify about those events.

`TcpServer` will listen permanently for new connections and new data with `Listen()`, and support a graceful shutdown with `Close()`.

We provide 2 options ot send data to our clients, by their device uid ( generated from the client side) with `SendDataByDeviceUid`or by the client id which is generated in our system with `SendDataByClientId`.  

## API

We need to create structs for the API that the tcp clients use, and the API for the messages sent to/from the messages broker.  
For the TCP clients:  
*  [DeviceRequest](https://github.com/orrchen/go-messaging/blob/1c5f38664ee822b1e8ebd29560bd41fcdd1c12c3/models/device_request.go)
*  [DeviceResponse](https://github.com/orrchen/go-messaging/blob/1c5f38664ee822b1e8ebd29560bd41fcdd1c12c3/models/device_response.go)  

For the message broker:  
*  [ServerRequest](https://github.com/orrchen/go-messaging/blob/1c5f38664ee822b1e8ebd29560bd41fcdd1c12c3/models/server_request.go)  
*  [ServerResponse](https://github.com/orrchen/go-messaging/blob/1c5f38664ee822b1e8ebd29560bd41fcdd1c12c3/models/server_response.go)  

## [Main function - putting it all together](https://github.com/orrchen/go-messaging/blob/1c5f38664ee822b1e8ebd29560bd41fcdd1c12c3/main.go)
Obtains and manages all the other components in this system. It will include the TCP server that holds an array of TCP clients, and a connection to the Kafka broker for consuming and sending messages to it.
Here are the main parts of `main.go` file:
```go
var tcpServer *lib.TcpServer
var producer *messages.Producer
var consumer *messages.Consumer

func main() {
    callbacks := lib.Callbacks{
		OnDataReceived: onDataReceived,
		OnConnectionTerminated: onConnectionTerminated,
		OnNewConnection: onNewConnection,
	}
	tcpServer = lib.NewServer(":3000", callbacks)
	producerCallbacks := messages.ProducerCallbacks{
		OnError: onProducerError,
	}
	f := false
	producer = messages.NewProducer(producerCallbacks,configuration.BrokersList,configuration.ProducerTopic,nil,nil,nil,&f)

	consumerCallbacks := messages.ConsumerCallbacks{
		OnDataReceived: onDataConsumed,
		OnError: onConsumerError,
	}
	consumer = messages.NewConsumer(consumerCallbacks,configuration.BrokersList,consumerGroupId,configuration.ConsumerTopics)
	consumer.Consume()

	go func(){
		http.HandleFunc("/", handler)
		http.ListenAndServe(":8080", nil)
	}()

	tcpServer.Listen()


}

func cleanup(){
	tcpServer.Close()
	producer.Close()
	consumer.Close()
	os.Exit(0)
}
```
## Build, run and deploy to Docker image
To build:
```bash 
go build main.go
```
To run:
```bash
go run main.go -config=config/config.yml
```
To build and run with Docker I first set this `Dockerfile`:
```docker
FROM debian
MAINTAINER "Orr Chen"
WORKDIR /app
ADD app/tcp-server.linux /app/
ADD config /app/
EXPOSE 8080 3000
CMD ["./tcp-server.linux","-config=config.yml"]
```
And build and push to my Docker repository with the [build.sh](https://github.com/orrchen/go-messaging/blob/master/build.sh) script.
```bash
build.sh --tag <tag>  --name <name> --repository <repository>
```

## Future improvements
Of course this is just a base framework, it lacks a few things mandatory for production environments which are mainly authentication, better logging, recovery from errors and input checking.  
But I believe this might be a very useful start point for many developers who need this kind of a service, just like I needed it before implementing it :)  
I will be very happy to read your thoughts and comments, happy holidays to all!

## About the author:
Hi, my name is Orr Chen, a software engineer and a gopher for the past 3 years.
My first experience with Go was migrating the entire backend of my startup [PushApps](www.pushapps.mobi) from Rails to Golang.  Since then I am a big fun of the language!  
Github: [OrrChen](https://github.com/orrchen)  
Twitter: [OrrChen](https://twitter.com/OrrChen)  
LinkedIn: [orrchen](www.linkedin.com/in/orrchen)

### 3rd parties libraries used:  
* [gopkg.in/yaml.v2](gopkg.in/yaml.v2)
* [github.com/satori/go.uuid](github.com/satori/go.uuid)
* [github.com/Shopify/sarama](github.com/Shopify/sarama)
* [github.com/bsm/sarama-cluster](github.com/bsm/sarama-cluster)


