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
1. **TCP servers** - Needs to keep maintain as many as possible concurrent tcp sockets with the endpoints. all of the endpoints messages will be processed on a different layer through the message broker. This keeps the TCP servers layer very thin and effective and to keep as many concurrent connection as possible, and Go is a good pick for it ( see [this article](https://medium.freecodecamp.org/million-websockets-and-go-cc58418460bb))  
2. **Message broker** - Responsible for delivering the messages between the TCP servers layer and the workers layer. I chose [Apache Kafka](https://kafka.apache.org/) for that purpose.  
3. **Workers layer** - will process the messages through services exposed in the backend layer.  
4. **Backed services layer** - An encapsulation of services requires by your application such as DB, Authentication, Logging, external APIs and more.  
  
So, this Go Server:  
1. communicates with its endpoint clients by TCP sockets.  
2. queues the messages in the message broker.  
3. receives back messages from the broker after they were processed and send response acknowledgment and/or errors to the TCP clients.  

The full source code is available in : https://github.com/orrchen/go-messaging  
I have also included a Dockerfile and a build script to push the image to your Docker repository.  
Special thanks to the great go Kafka [sarama library from Shopify](https://github.com/Shopify/sarama).

_The article is divided to sections representing the components of the system. Each component should be decoupled from the others in a way that if you are interested in reading about just one of components it should be straight forward._

## TCP Client 
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
pay attention that ```onConnectionEvent``` and ```onDataEvent``` are callbacks for the Struct that will obtain and manage Clients.

We will also define constants for events:
```go
const
(
	CONNECTION_EVENT_TYPE_NEW_CONNECTION           ConnectionEventType = "new_connection"
	CONNECTION_EVENT_TYPE_CONNECTION_TERMINATED    ConnectionEventType = "connection_terminated"
	CONNECTION_EVENT_TYPE_CONNECTION_GENERAL_ERROR ConnectionEventType = "general_error"
)
```
Our client will listen permanently using the ```listen()``` function:
```go
// Read client data from channel
func (c *Client) listen() {
	reader := bufio.NewReader(c.conn)
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)

		switch err {
		case io.EOF:
			// connection terminated
			c.conn.Close()
			c.onConnectionEvent(c,CONNECTION_EVENT_TYPE_CONNECTION_TERMINATED, err)
			return
		case nil:
			// new data available
			c.onDataEvent(c, buf[:n])
		default:
			log.Fatalf("Receive data failed:%s", err)
			c.conn.Close()
			c.onConnectionEvent(c, CONNECTION_EVENT_TYPE_CONNECTION_GENERAL_ERROR, err)
			return
		}
	}
}
```

## Kafka Consumer

Its role is to consume messages from our Kafka broker, and to broadcast them back to relevant clients by their uids.  
In this example we are consuming from multiple topics using the [cluster implementation of sarama](github.com/bsm/sarama-cluster).

let's define our ```Consumer``` struct:  
```go
type Consumer struct {
	consumer *cluster.Consumer
	callbacks ConsumerCallbacks
}
```
```ConsumerCallbacks``` are:
```go

type ConsumerCallbacks struct {
	OnDataReceived func(msg []byte)
	OnError func(err error)
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
It will consume permanently on a new goroutine:
```go
func (c *Consumer) Consume() {
	// Create signal channel
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	// Consume all channels, wait for signal to exit
	go func(){
		for {
			select {
			case msg, more := <-c.consumer.Messages():
				if more {
					if c.callbacks.OnDataReceived!=nil {
						c.callbacks.OnDataReceived(msg.Value)
					}
					fmt.Fprintf(os.Stdout, "%s/%d/%d\t%s\n", msg.Topic, msg.Partition, msg.Offset, msg.Value)
					c.consumer.MarkOffset(msg, "")
				}
			case ntf, more := <-c.consumer.Notifications():
				if more {
					log.Printf("Rebalanced: %+v\n", ntf)
				}
			case err, more := <-c.consumer.Errors():
				if more {
					if c.callbacks.OnError!=nil {
						c.callbacks.OnError(err)
					}
				}
			case <-sigchan:
				return
			}
		}
	}()

}
```

## Kafka Producer
Its role is to produce messages to our Kafka broker.  
In this example we are producing to a single topic.  
This section is mainly inspired from the example in https://github.com/Shopify/sarama/blob/master/examples/http_server/http_server.go

Let's define our ```Producer``` Struct:
```go
type Producer struct {
	asyncProducer sarama.AsyncProducer
	callbacks     ProducerCallbacks
	topic         string
}
```
These are the callbacks:
```go
type ProducerCallbacks struct {
	OnError func(error)
}
```
```Producer``` is constructed with the callbacks for error, and the details to connect to the Kafka broker including optional ssl configurations:
```go
func NewProducer(callbacks ProducerCallbacks,brokerList []string,topic string,certFile *string,keyFile *string,caFile *string,verifySsl *bool ) *Producer {
	producer := Producer{ callbacks: callbacks, topic: topic}

	config := sarama.NewConfig()
	tlsConfig := createTlsConfiguration(certFile,keyFile,caFile,verifySsl)
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
To create the TLS configurations use:
```go
func createTlsConfiguration(certFile *string,keyFile *string,caFile *string,verifySsl *bool)(t *tls.Config) {
	if certFile!=nil && keyFile!=nil && caFile!=nil && *certFile != "" && *keyFile != "" && *caFile != "" {
		cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
		if err != nil {
			log.Fatal(err)
		}

		caCert, err := ioutil.ReadFile(*caFile)
		if err != nil {
			log.Fatal(err)
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		t = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            caCertPool,
			InsecureSkipVerify: *verifySsl,
		}
	}
	// will be nil by default if nothing is provided
	return t
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
## TCP Server

Its role is to obtain and manage a set of ```Client```, and send and receive messages from them.
```go
type TcpServer struct {
	address                  string // Address to open connection: localhost:9999
	connLock sync.RWMutex
	connections map[string]*Client
	callbacks Callbacks
	listener net.Listener
}
```
it is constructed simply with an address to bind to and the callbacks to send:
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
```go
func (s *TcpServer) onConnectionEvent(c *Client,eventType ConnectionEventType, e error ) {
	switch eventType {
	case CONNECTION_EVENT_TYPE_NEW_CONNECTION:
		s.connLock.Lock()
		u1 := uuid.NewV4()
		uidString := u1.String()
		c.Uid = uidString
		s.connections[uidString] = c
		s.connLock.Unlock()
		if s.callbacks.OnNewConnection != nil {
			s.callbacks.OnNewConnection(uidString)
		}
	case CONNECTION_EVENT_TYPE_CONNECTION_TERMINATED, CONNECTION_EVENT_TYPE_CONNECTION_GENERAL_ERROR:
		s.connLock.Lock()
		delete(s.connections,c.Uid)
		s.connLock.Unlock()
		if s.callbacks.OnConnectionTerminated!=nil {
			s.callbacks.OnConnectionTerminated(c.Uid)
		}
	}
}
```
we define ```OnDataEvent``` callback to pass for each ```Client```:
```go
func (s *TcpServer) onDataEvent(c *Client, data []byte) {
	if s.callbacks.OnDataReceived!=nil {
		s.callbacks.OnDataReceived(c.Uid, data)
	}
}
```
```TcpServer``` will listen permanently for new connections and new data with ```Listen```:
```go
// Start network Server
func (s *TcpServer) Listen() {
	var err error
	s.listener, err = net.Listen("tcp", s.address)
	if err != nil {
		log.Fatal("Error starting TCP Server.: " , err)
	}
	for {
		conn, _ := s.listener.Accept()
		client := NewClient(conn,s.onConnectionEvent,s.onDataEvent)
		s.onConnectionEvent(client, CONNECTION_EVENT_TYPE_NEW_CONNECTION,nil)
		go client.listen()

	}
}
```
We need to shut it down gracefully by closing all open connections:
```go
func (s *TcpServer) Close(){
	log.Println("TcpServer.Close()")
	log.Println("s.connections length: " , len(s.connections))
	for k := range s.connections {
		fmt.Printf("key[%s]\n", k)
		s.connections[k].Close()
	}
	s.listener.Close()
}
```
We provide 2 options ot send data to our clients, by their device uid ( generated from the client side) or by the client id which is generated in our system:
```go
func (s *TcpServer) SendDataByClientId(clientUid string, data []byte) error{
	if s.connections[clientUid]!=nil {
		return s.connections[clientUid].Send(data)
	} else {
		return errors.New(fmt.Sprint("no connection with uid ", clientUid))
	}

	return nil
}

func (s *TcpServer) SendDataByDeviceUid(deviceUid string, data []byte) error{
	for k := range s.connections {
		if s.connections[k].DeviceUid == deviceUid {
			return s.connections[k].Send(data)
		}
	}
	return errors.New(fmt.Sprint("no connection with deviceUid ", deviceUid))
}
```
And we also provide a way to bind a device uid to an existing client id ( in case a registration process happens during the processing of our messages and pushed back to us ):
```go
func (s *TcpServer) SendDataByClientId(clientUid string, data []byte) error{
	if s.connections[clientUid]!=nil {
		return s.connections[clientUid].Send(data)
	} else {
		return errors.New(fmt.Sprint("no connection with uid ", clientUid))
	}

	return nil
}

func (s *TcpServer) SendDataByDeviceUid(deviceUid string, data []byte) error{
	for k := range s.connections {
		if s.connections[k].DeviceUid == deviceUid {
			return s.connections[k].Send(data)
		}
	}
	return errors.New(fmt.Sprint("no connection with deviceUid ", deviceUid))
}
```

## API

We need to create structs for the API that the tcp clients use, and the API for the messages sent to/from the messages broker.  
For the TCP clients:
```go
type DeviceRequest struct {
	Action string `json:"action"`
	DeviceUid string `json:"deviceUid"`
	Uid string `json:"uid"`
	Data map[string]interface{} `json:"data"`
}
type DeviceResponse struct {
	Action string `json:"action"`
	Uid string `json:"uid"`
	Data map[string]interface{} `json:"data"`
	Status string `json:"status"`
	ErrorCode string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}
```
For the message broker:
```go
type ServerRequest struct{
	DeviceRequest DeviceRequest `json:"deviceRequest"`
	ServerId string `json:"serverId"` // unique identifier for each server in case of having more than 1 server
	ClientId string `json:"clientId"`
}
type ServerResponse struct {
	DeviceResponse DeviceResponse `json:"deviceResponse"`
	ServerId string `json:"serverId"` // unique identifier for each server in case of having more than 1 server
	ClientId string `json:"clientId"`
	DeviceUid string `json:"deviceUid"`
}
```
## Configurations
I use ```.yml``` files for configurations that change between environments. Here is how to model them and parse from the ```.yml``` file.
```go
func InitConfig(configPath string) {
	if conf == nil {
		filename, _ := filepath.Abs(configPath)
		log.Println("trying to read file ", filename)
		yamlFile, err := ioutil.ReadFile(filename)
		var confi Configuration
		err = yaml.Unmarshal(yamlFile, &confi)
		if err != nil {
			log.Println(err)
			panic(err)
		}
		conf = &confi
	}
}

func Get() *Configuration {
	return conf
}

type Configuration struct {
	BrokersList []string  `yaml:"brokers_list"`
	ProducerTopic		string `yaml:"producer_topic"`
	ConsumerTopics		[]string `yaml:"consumer_topics"`
	ConsumerGroupId     string `yaml:"consumer_group_id"`
}
```
And here is an example for a ```yml``` config file:
```yaml
brokers_list:
  - "localhost:9092"
producer_topic: "tcp_layer_messages"
consumer_topics:
  - "workers_layer_messages"
consumer_group_id: "id-1"
```
## Main function - putting it all together
Here obtain and manages all the other components in this system. It will include the TCP server that holds an array of TCP clients, and a connection to the Kafka broker for consuming and sending messages to it.
Here is the full ```main.go``` file:
```go
var tcpServer *lib.TcpServer
var producer *messages.Producer
var consumer *messages.Consumer

var (
	configPath = flag.String("config", "", "config file")
	consumerGroupId string
)

func main() {
	flag.Parse()
	if *configPath == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
	config.InitConfig(*configPath)
	configuration := config.Get()
	if configuration.ConsumerGroupId==""{
		consumerGroupId = uuid.NewV4().String()
	} else {
		consumerGroupId = configuration.ConsumerGroupId
	}
	log.Printf("Kafka brokers: %s", strings.Join(configuration.BrokersList, ", "))
	
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

	signal_chan := make(chan os.Signal, 1)
	signal.Notify(signal_chan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGKILL)

	go func() {
		for {
			s := <-signal_chan
			switch s {
			case syscall.SIGINT:
				fmt.Println("syscall.SIGINT")
				cleanup()
				// kill -SIGTERM XXXX
			case syscall.SIGTERM:
				fmt.Println("syscall.SIGTERM")
				cleanup()
				// kill -SIGQUIT XXXX
			case syscall.SIGQUIT:
				fmt.Println("syscall.SIGQUIT")
				cleanup()
			case syscall.SIGKILL:
				fmt.Println("syscall.SIGKILL")
				cleanup()
			default:
				fmt.Println("Unknown signal.")
			}
		}
	}()

	go func(){
		http.HandleFunc("/", handler)
		http.ListenAndServe(":8080", nil)
	}()

	tcpServer.Listen()


}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "TCP Server is up and running!")
}

func cleanup(){
	tcpServer.Close()
	producer.Close()
	consumer.Close()
	os.Exit(0)
}



func onNewConnection(clientUid string) {
	log.Println("onNewConnection, uid: ", clientUid)
}

func onConnectionTerminated(clientUid string) {
	log.Println("onConnectionTerminated, uid: ", clientUid)
}

/**
Called when data is received from a TCP client, will generate a message to the message broker
 */
func onDataReceived(clientUid string, data []byte) {
	log.Println("onDataReceived, uid: ", clientUid, ", data: ", string(data))
	if string(data)=="Ping" {
		log.Println("sending Pong")
		//answer with pong
		tcpServer.SendDataByClientId(clientUid, []byte("Pong"))
	}
	if producer!=nil {
		var deviceRequest models.DeviceRequest
		err:= json.Unmarshal(data,&deviceRequest)
		if err==nil {
			serverRequest := models.ServerRequest{
				DeviceRequest: deviceRequest,
				ServerId: "1",
				ClientId: clientUid,
			}
			producer.Produce(serverRequest)
		} else {
			log.Println(err)
		}

	}

}

func onProducerError(err error){
	log.Println("onProducerError: ", err)
}

func onConsumerError(err error){
	log.Println("onConsumerError: ",err)
}

func onDataConsumed(data []byte){
	log.Println("onDataConsumed: ", string(data))
	var serverResponse models.ServerResponse
	err := json.Unmarshal(data,&serverResponse)
	if err!=nil {
		log.Println(err)
		return
	}
	if serverResponse.DeviceResponse.Action == "connect.response" && serverResponse.DeviceResponse.Status == "ok" && serverResponse.ClientId!= "" {
		//attach the device id to our existing client
		err =tcpServer.SetDeviceUidToClient(serverResponse.ClientId,serverResponse.DeviceUid)
		if err!=nil {
			log.Println(err)
		}
	}
	toSend, err := json.Marshal(serverResponse.DeviceResponse)
	if err!=nil {
		log.Println(err)
		return
	}
	if serverResponse.ClientId!="" {
		tcpServer.SendDataByClientId(serverResponse.ClientId,toSend)
	} else {
		if serverResponse.DeviceUid!=""{
			tcpServer.SendDataByDeviceUid(serverResponse.DeviceUid,toSend)
		}
	}


}
```
Few things to notice here:  
* TCP server is listening on port 3000.
* Test handler on port 8080 to see on the browser everything is running.
* Ping will be answered with Pong :)
* Listening to various system calls to gracefully shutting down.
* An example for setting a device uid to a given client id, assuming there is an event with ``` Action ==  "connect.response" ```
## Build, run and deploy to Docker image
To build:
```bash 
go build main.go
```
To run:
```bash
go run main.go -config=config/config.yml
```
To build and run with Docker I first set this ```Dockerfile```:
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


