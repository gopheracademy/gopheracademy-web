+++
author = ["Jorick Caberio"]
title = "How to Send and Receive SMS: Implementing a GSM Protocol in Go"
date = 2018-12-05T00:00:00Z
series = ["Advent 2018"]
+++

When developers add an SMS component in their app either for verification or notification purposes, they usually do it via RESTful API like the ones provided by [Twilio](https://www.twilio.com/docs/sms/api). But what really happens behind the scenes? 

In this post, you'll learn what [Universal Computer Protocol (UCP)](https://wiki.wireshark.org/UCP) is and how you can use it to directly communicate with a [Short Message Service Centre (SMSC)](https://en.wikipedia.org/wiki/Short_Message_service_center) to send and receive [SMS](https://en.wikipedia.org/wiki/SMS) using Go.

## Terminology

### Mobile Terminating Message

Messages from telco to subscriber. For example, weather update messages.

![mobile-terminating](/postimages/advent-2018/mobile-terminating.png)


### Mobile Originating Message

Messages from subscriber to telco. For example, texting a keyword to an accesscode for balance inquiry.

![mobile-originating](/postimages/advent-2018/mobile-originating.png)

### Multi-part MTs and MOs

SMS greater than 160 characters are considered multi-part SMS.
To send a multi-part mobile terminating messages, we need to split it into message parts. Each message part will contain a message part number, total message parts number, and a reference number. 

The multi-part mobile originating message also contains a message part number, total message parts number, and a reference number. We need to concatenate those message parts in order to interpret the mobile originating message that the user sent.



## Universal Computer Protocol


Universal Computer Protocol or UCP is primarily used to connect to Short Message Service Centres (SMSC) to send and receive SMS.

![protocol](/postimages/advent-2018/emiucp.svg)

### session management operation

Allows us the send login credentials to the SMSC.

### alert operation

Allows us to send pings to the SMSC.

### submit short message operation

Allows us to send mobile terminating SMS.

### delivery notification operation

Sent by the SMSC to the client as a delivery status receipt indicating if the SMS message was successfully sent or not.

### delivery short message operation

Sent by the SMSC to the client on behalf of a subscriber's mobile originating SMS message.

## Implementation

We can treat UCP like a traditional client-server protocol. After establishing a TCP connection, we send UCP requests with a sequence number (called "transaction reference number" in [the protocol specification](http://documents.swisscom.com/product/1000174-Internet/Documents/Landingpage-Mobile-Mehrwertdienste/UCP_R4.7.pdf)) from 00 to 99 and the SMSC will respond with a UCP response message synchronously. However, the SMSC can also send UCP requests, just like in the case of "delivery notification operation" and "delivery short message operation". We also need to periodically send keepalive pings to the SMSC so that it won't treat the connection as stale and disconnect us.



Lets start with a `Client` struct containing the login credentials to the SMSC.
```go
// Client represents a UCP client connection.
type Client struct {
  // IP:PORT address of the SMSC	
  addr string
  // SMSC username
  user string
  // SMSC pasword
  password string
  // SMSC accesscode
  accessCode string 
}
```

### Transaction Reference Number

To generate valid transaction reference numbers ranging from 00 to 99, we can use the [ring](https://golang.org/pkg/container/ring/) package from the standard library.

```go
// Client represents a UCP client connection.
type Client struct {
  // skipped fields ...
  
  // ring counter for sequence numbers 00-99
  ringCounter *ring.Ring 
}
```

```go
const maxRefNum = 100

// initRefNum initializes the ringCounter counter from 00 to 99
func (c *Client) initRefNum() {
  ringCounter := ring.New(maxRefNum)
  for i := 0; i < maxRefNum; i++ {
    ringCounter.Value = []byte(fmt.Sprintf("%02d", i))
    ringCounter = ringCounter.Next()
  }
  c.ringCounter = ringCounter
}

// nextRefNum returns the next transaction reference number
func (c *Client) nextRefNum() []byte {
  refNum := (c.ringCounter.Value).([]byte)
  c.ringCounter = c.ringCounter.Next()
  return refNum
}
```

### Establishing TCP Connection

We can use the [net](https://golang.org/pkg/net/
) package to establish a TCP connection with the SMSC, then create a buffered reader and writer using the [bufio](https://golang.org/pkg/bufio) package.

After establishing the TCP connection, we can now send a `session management operation` request to the SMSC. This request contains our credentials to the SMSC.
```go
type Client struct {

  // skipped fields ....

  conn net.Conn
  reader *bufio.Reader
  writer *bufio.Writer

}
```

```go
const etx = 3

func (c *Client) Connect() error {
  // initialize ring counter from 00-99
  c.initRefNum()

  // establish TCP connection
  conn, _ := net.Dial("tcp", c.addr)
  c.conn = conn

  // create buffered reader and writer
  c.reader = bufio.NewReader(conn)
  c.writer = bufio.NewWriter(conn)

  // login to SMSC
  c.writer.Write(login(c.nextRefNum(), c.user, c.password))
  c.writer.Flush()
  resp, _ := c.reader.ReadString(etx)
  err = parseSessionResp(resp)
    
  // ....other processing....
  
  return err
}
```
`login` creates a `session management operation` request packet containing our credentials.
`parseSessionResp` parses the `session management operation` response packet from the SMSC. If our credentials are invalid, it will return an `error` otherwise it will return `nil`.

### Channels and Goroutines

We can treat the different UCP operations as separate goroutines and channels.
 
```go
type Client struct {
  // skipped fields ....
  // channel for handling submit short message responses from SMSC
  submitSmRespCh chan []string
  // channel for handling delivery notification requests from SMSC
  deliverNotifCh chan []string
  // channel for handling delivery message requests from SMSC
  deliverMsgCh chan []string
  // channel for handling incomplete delivery message from SMSC
  deliverMsgPartCh chan deliverMsgPart
  // channel for handling complete delivery message requests from SMSC
  deliverMsgCompleteCh chan deliverMsgPart
  // we close this channel to signal goroutine termination
  closeChan chan struct{}
  // waitgroup for the running goroutines
  wg *sync.WaitGroup
  // guard against closing closeChan multiple times
  once sync.Once
}

func (c *Client) Connect() error {
  // after login
  sendAlert(/*....*/)
  readLoop(/*....*/)
  readDeliveryNotif(/*....*/)
  readDeliveryMsg(/*....*/)
  readPartialDeliveryMsg(/*....*/)
  readCompleteDeliveryMsg(/*....*/)

  return err
}

// Close will close the UCP connection. 
// It's safe to call Close multiple times.
func (c *Client) Close() {
  // close closeChan to terminate the spawned goroutines
  // we use a sync.Once to close closeChan only once.
  c.once.Do(func() {
    close(c.closeChan)
  })
  // close the underlying TCP connection
  if c.conn != nil {
    c.conn.Close()
  }
  // wait for all goroutines to terminate  
  c.wg.Wait()
}
```

### Read UCP packets

To read packets from the UCP connection, we start the `readLoop` goroutine. A valid UCP packet is delimited by an [End-of-Text indicator (ETX)](https://en.wikipedia.org/wiki/End-of-Text_character), that is the byte `03`.
`readLoop` will read up to `etx`, parse the packet and send it to the appropriate channel.
```go
// readLoop reads incoming messages from the SMSC 
// using the underlying bufio.Reader
func readLoop(/*.....*/) {
  wg.Add(1)
  go func() {
    defer wg.Done()
    for {
      select {
      case <-closeChan:
        return
      default:
        readData, _ := reader.ReadString(etx)
        opType, fields, _ := parseResp(readData)
        switch opType {
        case opSubmitShortMessage:
          submitSmRespCh <- fields
        case opDeliveryNotification:
          deliverNotifCh <- fields
        case opDeliveryShortMessage:
          deliverMsgCh <- fields
        }
      }
    }
  }()
}
```

### Send Keepalive

To send periodic pings to the SMSC, we start the `sendAlert` goroutine.
We use [time.NewTicker](https://golang.org/pkg/time/#NewTicker) to create a ticker that will fire periodically.
`ping` creates a valid `alert operation` request packet with the appropriate transaction reference number.

```go
func sendAlert(/*....*/) {
  wg.Add(1)
  ticker := time.NewTicker(alertInterval)
  go func() {
    defer wg.Done()
    for {
      select {
      case <-closeChan:
        ticker.Stop()
        return
      case <-ticker.C:
        writer.Write(ping(transRefNum, user))
        writer.Flush()
      }
    }
  }()
}
```

### Read Delivery Notification

To read SMS delivery notification status, we start the `readDeliveryNotif` goroutine.
Once a `delivery notification operation` message is read, it sends an acknowledgement response packet to the SMSC.
```go
// Read deliver notifications from deliverNotifCh channel. 
func readDeliveryNotif(/*....*/) {
  wg.Add(1)
  go func() {
    defer wg.Done()
    for {
      select {
      case <-closeChan:
        return
      case dr := <-deliverNotifCh:
        refNum := dr[refNumIndex]
        // msg contains the complete delivery status report from the SMSC
        msg, _ := hex.DecodeString(dr[drMsgIndex]) 
        // sender is the access code of the SMSC
        sender := dr[drSenderIndex]
        // recvr is the mobile number of the recipient subscriber
        recvr := dr[drRecvrIndex]
        // scts is the service center time stamp
        scts := dr[drSctsIndex]
        msgID := recvr + ":" + scts
        writer.Write(deliveryNotifAckPacket([]byte(refNum), msgID))
        writer.Flush()
      }
    }
  }()
}
```

### Read Delivery Short Message

To read incoming mobile originating messages, we start the `readDeliveryMsg` goroutine.

```go
// Reads all delivery short messages (mobile-originating messages) 
// from the deliverMsgCh channel.
func readDeliveryMsg(/*....*/) {
  wg.Add(1)
  go func() {
    defer wg.Done()
    for {
      select {
      case <-closeChan:
        return
      case mo := <-deliverMsgCh:
        xser := mo[xserIndex]
        xserData := parseXser(xser)
        msg := mo[moMsgIndex]
        refNum := mo[refNumIndex]
        sender := mo[moSenderIndex]
        recvr := mo[moRecvrIndex]
        scts := mo[moSctsIndex]
        sysmsg := recvr + ":" + scts
        msgID := sender + ":" + scts

        // send ack to SMSC with the same reference number
        writer.Write(deliverySmAckPacket([]byte(refNum), sysmsg))
        writer.Flush()
          
        var incomingMsg deliverMsgPart
        incomingMsg.sender = sender
        incomingMsg.receiver = recvr
        incomingMsg.message = msg
        incomingMsg.msgID = msgID
        // further processing    
      }
    }
  }()
}
```

`deliverMsgPart` is a struct that contains the neccessary parts to concatenate and decode the partial incoming mobile originating message.

```go
// deliverMsgPart represents a deliver sm message part
type deliverMsgPart struct {
  currentPart int
  totalParts  int
  refNum      int
  sender      string
  receiver    string
  message     string
  msgID       string
  dcs         string
}
```


To handle multi-part mobile originating SMS, we send partial mobile originating messages to `deliverMsgPartCh` channel and complete mobile originating messages to `deliverMsgCompleteCh` channel.

```go

// Reads all deliver sm messages(mobile-originating messages) 
// from the deliverMsgCh channel.
func readDeliveryMsg(/*....*/) {
  wg.Add(1)
  go func() {
    defer wg.Done()
    for {
      select {
      case <-closeChan:
        return
      case mo := <-deliverMsgCh:
        // initial processing ...... 
          
        if xserUdh, ok := xserData[udhXserKey]; ok {
          // handle multi-part mobile originating message
          // get the total message parts in the xser data
          msgPartsLen := xserUdh[len(xserUdh)-4 : len(xserUdh)-2]
          // get the current message part in the xser data
          msgPart := xserUdh[len(xserUdh)-2:]
          // get message part reference number
          msgRefNum := xserUdh[len(xserUdh)-6 : len(xserUdh)-4]
          // convert hexstring to integer
          msgRefNumInt, _ := strconv.ParseInt(msgRefNum, 16, 0)
          msgPartsLenInt, _ := strconv.ParseInt(msgPartsLen, 16, 64)
          msgPartInt, _ := strconv.ParseInt(msgPart, 16, 64)
          incomingMsg.currentPart = int(msgPartInt)
          incomingMsg.totalParts = int(msgPartsLenInt)
          incomingMsg.refNum = int(msgRefNumInt)
          // send to partial channel
          deliverMsgPartCh <- incomingMsg 
        } else {
          // handle mobile originating message with only 1 part
          // send the incoming message to the complete channel
          deliverMsgCompleteCh <- incomingMsg
        }
      }
    }
  }()
}
```

The goroutine spawned in `readPartialDeliveryMsg` will read from `deliverMsgPartCh` channel and concatenate the incoming mobile originating message parts. The goroutine spawned in `readCompleteDeliveryMsg` will receive from `deliverMsgCompleteCh` channel and execute the callback for mobile originating messages.

 


### Send SMS

To send an SMS, we call the `Send` method.

```go
// Send will send the message to the receiver with a sender mask.
// It returns a list of message IDs from the SMSC.
func (c *Client) Send(sender, receiver, message string) ([]string, error) {
  msgType := getMessageType(message)
  msgParts := getMessageParts(message)
  refNum := rand.Intn(maxRefNum)
  ids := make([]string, len(msgParts))
  for i := 0; i < len(msgParts); i++ {
    sendPacket := encodeMessage(c.nextRefNum(), sender, receiver, msgParts[i], msgType,
      c.GetBillingID(), refNum, i+1, len(msgParts))
    c.writer.Write(sendPacket)
    c.writer.Flush()
    select {
    case fields := <-c.submitSmRespCh:
      ack := fields[ackIndex]
      if ack == negativeAck {
        errMsg := fields[len(fields)-errMsgOffset]
        errCode := fields[len(fields)-errCodeOffset]
        return ids, &UcpError{errCode, errMsg}
      }
      id := fields[submitSmIdIndex]
      ids[i] = id
    case <-time.After(c.timeout):
      return ids, &UcpError{errCodeTimeout, "Network time-out"}
    }
  }
  return ids, nil
}
```
`getMessageType` determines whether the message contains plain GSM 7-bit characters or Unicode characters.

`getMessageParts` splits the message into multiple parts in case it's a multi-part message.


`encodeMessage` takes care of creating a valid `submit short message orperation` request packet with the appropriate reference number. It handles text encoding to [UCS2](https://en.wikipedia.org/wiki/Universal_Coded_Character_Set) for unicode messages as well as masking the sender name. 

To get the response from the SMSC, we use `select` statement, that blocks until the data from the `submitSmRespCh` channel can be read or a given timeout occurred.

`Send` returns a list of message identifiers indicating that the SMSC received the `submit short message operation` request. This response is synchronous. For example, if we send a multi-part SMS consisting of five message parts, `Send` will return a list of five strings.

```go
[09191234567:130817221851, 09191234567:130817221852, 09191234567:130817221853, 09191234567:130817221854, 09191234567:130817221855]
```
 Each identifier has the form `recipient:timestamp`
 
## Conclusion

Go's built-in features such as goroutines and channels enabled us to implement the UCP protocol. We used Go's message-passing style for concurrently processing different types of UCP messages. We treat the independent operations as goroutines and communicate with them via channels. We also relied heavily on the standard library to implement the protocol operations. If you work on the telco field and have an access to an SMSC, feel free to try the [ucp package](https://github.com/go-gsm/ucp). It has additional features such as rate limiting and tariff charging. I've also written a [CLI](https://github.com/go-gsm/ucp-cli) to test it out. Suggestions and recommendations are welcome.

Thanks!
 






















