+++
title = "Navigating the uncharted waters of SSL/TLS certificates and gRPC with Go"
date = "2019-12-12T00:00:00+00:00"
series = ["Advent 2019"]
author = ["Nicolas Leiva"]
+++

There are different ways to establishing a secure TLS connection with Go and gRPC. Contrary to popular belief, you don't need to manually provide the Server certificate to your gRPC client in order to encrypt the connection. This post will provide a list of code examples for different scenarios. The source code is available in this [repository](https://github.com/nleiva/grpc-tls).

## TLS

As stated in [RFC 5246](https://tools.ietf.org/html/rfc5246), *the primary goal of the Transport Layer Security (TLS) protocol is to provide privacy and data integrity between two communicating applications*. TLS is one of the authentication mechanisms that are built-in to gRPC. *It has TLS integration and promotes the use of TLS to authenticate the server, and to encrypt all the data exchanged between the client and the server* [[gRPC
Authentication](https://grpc.io/docs/guides/auth/)].

In order to establishing a TLS Connection, the client must send a `Client Hello` message to the Server to initiate the TLS Handshake. *The TLS Handshake Protocol, allows the server and client to authenticate each other and to negotiate an encryption algorithm and cryptographic keys before the application protocol transmits or receives its first byte of data* [[RFC 5246](https://tools.ietf.org/html/rfc5246)].

A `Client Hello` message includes a list of options the Client supports to establish a secure connection; The TLS `Version,` a `Random` number, a `Session ID`, the `Cipher Suites`, `Compression Methods` and `Extensions`.

The Server replies back with a `Server Hello` including its preferred TLS `Version`, a `Random` number, a `Session ID`, and the `Cipher Suite` and `Compression Method` selected. The Server will also include a signed TLS `Certificate`. The client⁠ —depending on its configuration⁠— will validate this certificate with a Certificate Authority (CA) to prove the identity of the Server. A CA is a trusted party that issues digital certificates. The certificate could also come on a separate message.

After this negotiation, they start the Client Key exchange over an encrypted channel (Symmetric vs. Asymmetric encryption). Next, they start sending encrypted application data. I’m oversimplifying this part a bit, but I think we already have enough context to evaluate the code snippets to follow.

## Certificates

Before we jump into code, let’s talk about certificates. The X.509 v3
certificate format is described in detail in [RFC 5280](https://tools.ietf.org/html/rfc5280). It encodes, among other things, the server’s public key and a digital signature (to validate the certificate’s authenticity).

```ruby
Certificate  ::=  SEQUENCE  {
    tbsCertificate       TBSCertificate,
    signatureAlgorithm   AlgorithmIdentifier,
    signatureValue       BIT STRING  }
```

Before you ask, TBS implies To-Be-Signed.

```ruby
TBSCertificate  ::=  SEQUENCE  {
    version         [0]  EXPLICIT Version DEFAULT v1,
    serialNumber         CertificateSerialNumber,
    signature            AlgorithmIdentifier,
    issuer               Name,
    validity             Validity,
    subject              Name,
    subjectPublicKeyInfo SubjectPublicKeyInfo,
    ...
    }
```

Some of the most relevant fields of a X.509 certificate are:

* `subject`: Name of the subject the certificate is issued to.
* `subjectPublicKey`: Public Key and algorithm with which the key is used (e.g., RSA, DSA, or Diffie-Hellman). See below.
* `issuer`: Name of the CA that has signed and issued the certificate
* `signature`: algorithm identifier for the algorithm used by the CA to sign the certificate (same as `signatureAlgorithm`).

```ruby
SubjectPublicKeyInfo  ::=  SEQUENCE  {
    algorithm            AlgorithmIdentifier,
    subjectPublicKey     BIT STRING  }
```

You can see this as Go code in the [x.509 library](https://golang.org/pkg/crypto/x509/#Certificate).

```go
type Certificate struct {
    ...
    Signature          []byte
    SignatureAlgorithm SignatureAlgorithm

    PublicKeyAlgorithm PublicKeyAlgorithm
    PublicKey          interface{}

    Version             int
    SerialNumber        *big.Int
    Issuer              pkix.Name
    ...
```

While an SSL Certificate is most reliable when issued by a trusted Certificate
Authority (CA), you can create self-signed certificates as decribed in [Creating self-signed certificates](https://itnext.io/practical-guide-to-securing-grpc-connections-with-go-and-tls-part-1-f63058e9d6d1#9495). You can alternatively run `make cert` after cloning the [repository](https://github.com/nleiva/grpc-tls), which is requiered to executed the following examples.

## gRPC

Now, let’s take a look at how we apply and take advantage of all this with Go and gRPC with a very simple gRPC Service. This Service will retrieve usernames by their ID. In the examples, we will query for `ID=1`, which returns user `Nicolas`. The protobuf definition is the following.

```protobuf
syntax = "proto3";

package test;

service gUMI {
  rpc GetByID (GetByIDRequest) returns (User);
}

message GetByIDRequest {
  uint32 id = 1;
}

message User {
  string name = 1;
  string email = 2;
  uint32 id = 3;
}
```

### Insecure gRPC connections

Let’s check a couple of non-recommended practices.

#### Connection without encryption

If you do **NOT** want to encrypt the connection, the Go `grpc` package offers the `DialOption` `WithInsecure()` for the Client. This, plus a Server without any `ServerOption` will result in an unencrypted connection.

```go
// Client
conn, err := grpc.Dial(address, grpc.WithInsecure())
if err != nil {
	log.Fatalf("did not connect: %v", err)
}
defer conn.Close()

// Server
s := grpc.NewServer()
// ... register gRPC services ...
if err = s.Serve(lis); err != nil {
	log.Fatalf("failed to serve: %v", err)
}
```

In order to reproduce this, run `make run-server-insecure` in one tab and `run-client-insecure` in another.

```bash
$ make run-server-insecure
2019/07/05 18:08:03 Creating listener on port: 50051
2019/07/05 18:08:03 Starting gRPC services
2019/07/05 18:08:03 Listening for incoming connections
```

Second tab.

```bash
$ make run-client-insecure
User found:  Nicolas
```

#### Client does not authenticate the Server

In this case, we do encrypt the connection using the Server’s public key, however the client won’t validate the integrity of the Server’s certificate, so you can’t make sure you are actually talking to the Server and not to a man in the middle (`man-in-the-middle` attack).

To do this, we provide the public and private key pair on the server side we created previously. The client needs to set the config flag `InsecureSkipVerify` from the `tls` package to `true`.

```go
// Client
config := &tls.Config{
	InsecureSkipVerify: true,
}
conn, err := grpc.Dial(address, grpc.WithTransportCredentials(credentials.NewTLS(config)))
if err != nil {
	log.Fatalf("did not connect: %v", err)
}
defer conn.Close()

// Server
creds, err := credentials.NewServerTLSFromFile("service.pem", "service.key")
if err != nil {
	log.Fatalf("Failed to setup TLS: %v", err)
}
s := grpc.NewServer(grpc.Creds(creds))
// ... register gRPC services ...
if err = s.Serve(lis); err != nil {
	log.Fatalf("failed to serve: %v", err)
}
```

In order to reproduce this, run `make run-server` in one tab and `run-client` in another.

### Secure gRPC connections

Let’s look at how we can encrypt the communication channel and validate we are talking to who we think we are.

#### Automatically download the Server certificate and validate it

In order to validate the identity of the Server (authenticate it), the client uses the certification authority (CA) certificate **to authenticate the CA signature on the server certificate.** You can provide the CA certificate to your client or rely on a set of trusted CA certificates included in your operating system (trusted key store).

**Without a CA cert file**

In the previous example we didn’t really do anything special on the client side to encrypt the connection, other than setting the `InsecureSkipVerify` flag to `true`. In this case we will switch the flag to `false` to see what happens. The connection won’t be established and the client will log `x509: certificate signed by unknown authority`.

```go
// Client
config := &tls.Config{
	InsecureSkipVerify: false,
}
conn, err := grpc.Dial(address, grpc.WithTransportCredentials(credentials.NewTLS(config)))
if err != nil {
	log.Fatalf("did not connect: %v", err)
}
defer conn.Close()

// Server
creds, err := credentials.NewServerTLSFromFile("service.pem", "service.key")
if err != nil {
	log.Fatalf("Failed to setup TLS: %v", err)
}
s := grpc.NewServer(grpc.Creds(creds))
// ... register gRPC services ...
if err = s.Serve(lis); err != nil {
	log.Fatalf("failed to serve: %v", err)
}
```

In order to reproduce this, run `make run-server` in one tab and `run-client-noca` in another.

**With a Certification Authority (CA) cert file**

Let’s manually provide the CA cert file (`ca.cert`) and keep the `InsecureSkipVerify` option as `false`.

```go
// Client
b, _ := ioutil.ReadFile("ca.cert")
cp := x509.NewCertPool()
if !cp.AppendCertsFromPEM(b) {
	return nil, errors.New("credentials: failed to append certificates")
}
config := &tls.Config{
	InsecureSkipVerify: false,
	RootCAs:            cp,
}
conn, err := grpc.Dial(address, grpc.WithTransportCredentials(credentials.NewTLS(config)))
if err != nil {
	log.Fatalf("did not connect: %v", err)
}
defer conn.Close()

// Server
creds, err := credentials.NewServerTLSFromFile("service.pem", "service.key")
if err != nil {
	log.Fatalf("Failed to setup TLS: %v", err)
}
s := grpc.NewServer(grpc.Creds(creds))
// ... register gRPC services ...
if err = s.Serve(lis); err != nil {
	log.Fatalf("failed to serve: %v", err)
}
```

In order to reproduce this, run `make run-server` in one tab and `run-client-ca` in another.

**With CA certificates included in the system (OS/Browser)**

An empty `tls` config (`tls.Config{}`) will take care of loading your system CA
certs. We will validate this scenario in with certificates from [Let’s Encrypt](https://letsencrypt.org/) for a public domain in a few paragraphs.

You can alternatively manually load the CA certs from the system with `SystemCertPool()`.

```go
certPool, err := x509.SystemCertPool()
```

**If you have the Server cert and you trust it**

This is most common scenario found on Internet tutorials. If you own the server and client, you could pre-share the server’s certificate (`service.pem`) with the client and use it directly to encrypt the channel.

```go
// Client
creds, err := credentials.NewClientTLSFromFile("service.pem", "")
if err != nil {
	log.Fatalf("could not process the credentials: %v", err)
}
conn, err := grpc.Dial(address, grpc.WithTransportCredentials(creds))
if err != nil {
	log.Fatalf("did not connect: %v", err)
}
defer conn.Close()

// Server
creds, err := credentials.NewServerTLSFromFile("service.pem", "service.key")
if err != nil {
	log.Fatalf("Failed to setup TLS: %v", err)
}
s := grpc.NewServer(grpc.Creds(creds))
// ... register gRPC services ...
if err = s.Serve(lis); err != nil {
	log.Fatalf("failed to serve: %v", err)
}
```

In order to reproduce this, run `make run-server` in one tab and `run-client-file` in another.

## Certificate Authorities and automation

In the previous examples, we examined different (SSL/TLS) certificate combinations to secure a gRPC channel. As the number of endpoints grows, this process soon gets too complicated to carry out manually. It’s time to look at how to automate the generation of signed certificates our gRPC endpoints can use without our intervention. We will need a Certificate Authority (CA) we can interact with from our Go gRPC endpoints. We will explore alternatives for private and public domains.

For private domains our CA of choice will be the [Vault PKI Secrets Engine](https://www.vaultproject.io/docs/secrets/pki/index.html). In order to generate certificate signing requests (CSR) and renewals from our gRPC endpoints, we will use [Certify](https://github.com/johanbrandhorst/certify).

For public certificate generation and distribution, we’ll go with [Let’s Encrypt](https://letsencrypt.org/about/); *a free, automated, and open Certificate Authority*… how cool is that!?. The only thing they require from you is to demonstrate control over the domain with the Automatic Certificate Management Environment ([ACME](https://tools.ietf.org/html/rfc8555)) protocol.

This means we need an [ACME](https://tools.ietf.org/html/rfc8555) client, fortunately there is a list of Go [libraries we can chose from](https://letsencrypt.org/docs/client-options/) for this. In this opportunity, we will use [autocert](https://godoc.org/golang.org/x/crypto/acme/autocert) for its ease of use and support for [TLS-ALPN-01](https://tools.ietf.org/html/draft-ietf-acme-tls-alpn-05) challenge.

### Private domains: Vault and Certify

#### Vault

Vault is a secrets management and data protection open source project, which can store and control access to certificates, among other secrets like passwords and tokens. It’s distributed as a binary you can place anywhere in your `$PATH`. If you want to learn more about Vault, its [Getting Started](https://learn.hashicorp.com/vault/) guide is a good place to start. All the details of the setup used for this post are [documented here](https://github.com/nleiva/grpc-tls/blob/master/vault-cert.md#vault).

First, we run Vault with `vault server -config=vault_config.hcl`. The config file (`vault_config.hcl`) provides the `storage` backend where Vault data is stored. For simplicity, we are just using a local file. You could alternatively choose to store it in-memory, on a cloud provider or else. See all the options in [storage Stanza](https://www.vaultproject.io/docs/configuration/storage/index.html).

```ruby
storage "file" {
    path = ".../data"
}
```

Additionally, we specify the address Vault will bind to. TLS is enabled by default, so we need to provide a certificate and private key pair. If you choose to self-sign these (see [these instructions](https://github.com/nleiva/grpc-tls#generating-tsl-certificates) for an example), make sure you keep the Root certificate (`ca.cert`) handy, you will need it later on to make requests to Vault (*). Other TCP config options are documented in [tcp Listener Parameters](https://www.vaultproject.io/docs/configuration/listener/tcp.html#tcp-listener-parameters).

```ruby
listener "tcp" {
    address     = "localhost:8200"
    tls_cert_file = ".../vault.pem"
    tls_key_file = ".../vault.key"
}
```

After [initializing Vault’s Server](https://github.com/nleiva/grpc-tls/blob/master/vault-cert.md#initialize-the-server) and [unsealing Vault](https://github.com/nleiva/grpc-tls/blob/master/vault-cert.md#unseal-the-vault) you can validate is working with an API call.

```bash
$ curl \
    --cacert ca.cert \
    -i https://localhost:8200/v1/sys/health

HTTP/1.1 200 OK
...

{"initialized":true,"sealed":false,"standby":false, ...}
```

The next step is to enable Vault PKI Secrets Engine backend with `vault secrets enable pki`, generate a CA certificate and private key Vault will use to sign certificates, and create a role (`my-role`) that can make requests for our domain (`localhost`). See all the [details here](https://github.com/nleiva/grpc-tls/blob/master/vault-cert.md#enable-vault-pki-secrets-engine-backend).

```bash
vault write pki/roles/my-role \
    allowed_domains=localhost \
    allow_subdomains=true \
    max_ttl=72h
```

#### Certify

Now that our Certificate Authority (CA) is ready to go, we can make requests to it, to have our certificates signed. Which certificates you might ask, and how to automatically tell our gRPC endpoints to use them, if we don’t have them yet?. Enter [Certify](https://github.com/johanbrandhorst/certify), a Go library to *perform certificate distribution and renewal whenever it’s needed, automatically*. It not only works with Vault as CA backend, but also with [Cloudflare CFSSL](https://blog.cloudflare.com/introducing-cfssl/) and [AWS ACM](https://aws.amazon.com/certificate-manager/private-certificate-authority/).

The first step to configure Certify is to specify the backend `issuer`, Vault in this case.

```go
issuer := &vault.Issuer{
	URL: &url.URL{
		Scheme: "https",
		Host:   "localhost:8200",
	},
	TLSConfig: &tls.Config{
		RootCAs: cp,
	},
	Token: getenv("TOKEN"),
	Role:  "my-role",
}
```

In this example we identify our Vault instance and access credentials by providing:

* The listener address we configured for Vault (`localhost:8200`).
* The `TOKEN` we get after initializing Vault’s Server.
* The role we created (`my-role`).
* The CA certificate of the issuer of the certs we provided in Vault’s config. `cp` is a `x509.CertPool` that includes `ca.cert` in this case, as noted in (*).

You can, optionally, provide certificate details via `CertConfig`. We do it in this case to specify we want to generate private keys for our Certificate Signing Requests (CSR) using the `RSA` algorithm instead of Certify’s default `ECDSA P256`.

```go
cfg := certify.CertConfig{
	SubjectAlternativeNames: []string{"localhost"},
	IPSubjectAlternativeNames: []net.IP{
		net.ParseIP("127.0.0.1"),
		net.ParseIP("::1"),
	},
	KeyGenerator: RSA{bits: 2048},
}
```

Certify hooks into the `GetCertificate` and `GetClientCertificate` methods of `tls.Config` via the [Certify](https://godoc.org/github.com/johanbrandhorst/certbot#Certify) type, which we now build with; the previously collected information, a `Cache` *method to prevent requesting a new certificate for every incoming connection*, and a login plugin (`go-kit/log` in tis example).

```go
c := &certify.Certify{
	CommonName:  "localhost",
	Issuer:      issuer,
	Cache:       certify.NewMemCache(),
	CertConfig:  &cfg,
	RenewBefore: 24 * time.Hour,
	Logger:      kit.New(logger),
}
```

The last step is to create a `tls.Config` pointing to the `GetCertificate` method of the `Certify` we just created. Then, use this config in our gRPC Server.

```go
// Client
// ... as in http://bit.ly/go-grpc-tls-ca ...

// Server
tlsConfig := &tls.Config{
  GetCertificate: c.GetCertificate,
}

s := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))
// ... register gRPC services ...
if err = s.Serve(lis); err != nil {
	log.Fatalf("failed to serve: %v", err)
}
```

You can reproduce this by running `make run-server-vault` in one tab and `make run-client-ca` in another after pointing the environmental variable `CAFILE` to Vault’s certificate file (`ca-vault.cert`), which you can get as follows:

```bash
$ curl \
    --cacert ca.cert \
    https://localhost:8200/v1/pki/ca/pem \
    -o ca-vault.cert
```

Server:

```bash
$ make run-server-vault
...
level=debug time=2019-07-15T19:37:12.694833Z caller=logger.go:36 server_name=localhost remote_addr=[::1]:64103 msg="Getting server certificate"
level=debug time=2019-07-15T19:37:12.694936Z caller=logger.go:36 msg="Requesting new certificate from issuer"
level=debug time=2019-07-15T19:37:12.815081Z caller=logger.go:36 serial=451331845556263599050597627925015657462097174315 expiry=2019-07-18T19:37:12Z msg="New certificate issued"
level=debug time=2019-07-15T19:37:12.815115Z caller=logger.go:36 serial=451331845556263599050597627925015657462097174315 took=120.284897ms msg="Certificate found"
```

Client:

```bash
$ export CAFILE="ca-vault.cert"
$ make run-client-ca
...
User found:  Nicolas
```

Inspecting the certificate we generated and had signed automatically, will reveal some of the specifics we just configured.

```bash
$ openssl x509 -in grpc-cert.pem -text -noout
Certificate:
    Data:
    ...
        Validity
            Not Before: Jul 15 19:36:42 2019 GMT
            Not After : Jul 18 19:37:12 2019 GMT
        Subject: CN=localhost
        Subject Public Key Info:
            Public Key Algorithm: rsaEncryption
                Public-Key: (2048 bit)
                Modulus:
                    00:bf:3c:a3:d8:8c:d8:3c:d0:bd:0c:e0:4c:9d:4d:
                    ...
        X509v3 extensions:
            ...
            Authority Information Access:
                CA Issuers - URI:https://localhost:8200/v1/pki/ca
X509v3 Subject Alternative Name:
                DNS:localhost, DNS:localhost, IP Address:127.0.0.1, IP Address:0:0:0:0:0:0:0:1
```

### Public Domains: Let’s Encrypt and autocert

#### Let’s Encrypt

Can we use [Let’s Encrypt](https://letsencrypt.org/about/) for gRPC?. Well, it did work for me. The question might be whether having a public facing gRPC API’s is a good idea or not. Google Cloud seems to be doing it, see [Google APIs](https://github.com/googleapis/googleapis). However, this is not a very common practice. Anyways, here is how I was able to expose a public gRPC API with certificates we automatically get from Let’s Encrypt.

Is important to emphasize this example is not meant to be replicated for internal/private services. In talking to [Jacob Hoffman-Andrews](https://twitter.com/j4cob) from Let’s Encrypt, he mentioned: *In general, I recommend that people don’t use Let’s Encrypt certificates for gRPC or other internal RPC services. In my opinion, it’s both easier and safer to generate a single-purpose internal CA using something like [minica](https://github.com/jsha/minica/) and generate both server and client certificates with it. That way you don’t have to open up your RPC servers to the outside internet, plus you limit the scope of trust to just what’s needed for your internal RPCs, plus you can have a much longer certificate lifetime, plus you can get revocation that works.*

*Let’s Encrypt uses the ACME protocol to verify that an applicant for a certificate legitimately represents the domain name(s) in the certificate. It also provides facilities for other certificate management functions, such as certificate revocation. ACME describes an extensible framework for automating the issuance and domain validation procedure, thereby allowing servers and infrastructure software to obtain certificates without user interaction*. [[RFC 8555](https://tools.ietf.org/html/rfc8555).]

In a nutshell, all we need to do in order to leverage Let’s Encrypt is to run an [ACME client](https://letsencrypt.org/docs/client-options/). We will use [autocert](https://godoc.org/golang.org/x/crypto/acme) in this example.

#### autocert

The autocert package *provides automatic access to certificates from Let’s Encrypt and any other ACME-based CA*. However, keep in mind *this package is a work in progress and makes no API stability promises*. [[Documentation](https://godoc.org/golang.org/x/crypto/acme/autocert)]

In terms of code requirements, the first step to is to declare a `Manager` with a `Prompt` that *indicate acceptance of the CA’s Terms of Service during account registration*, a `Cache` method* to store and retrieve previously obtained certificates* (directory on the local filesystem in this case), a `HostPolicy` with the list of domains we can respond to, and optionally and `Email` *address to notify about problems with issued certificates*.

```go
manager := autocert.Manager{
	Prompt:     autocert.AcceptTOS,
	Cache:      autocert.DirCache("golang-autocert"),
	HostPolicy: autocert.HostWhitelist(host),
	Email:      "test@example.com",
}
```

This `Manager` will create a TLS config for us automagically, taking care of the interaction with Let’s Encrypt. The client, on the other hand, just needs a pointer to an empty `tls` config (`&tls.Config{}`), which will, by default, load the system CA certificates and therefore trust our CA (Let’s Encrypt).

```go
// Client
config := &tls.Config{}

conn, err := grpc.Dial(address, grpc.WithTransportCredentials(credentials.NewTLS(config)))
if err != nil {
	log.Fatalf("did not connect: %v", err)
}
defer conn.Close()


// Server
creds := credentials.NewTLS(manager.TLSConfig())
s := grpc.NewServer(grpc.Creds(creds))
// ... register gRPC services ...

// Listener...
```

If you are paying close attention, you might have noticed we didn’t include the listener section in this example. The reason is how the ACME TLS-based challenge TLS-ALPN-01 works. *The TLS with Application Level Protocol Negotiation (TLS ALPN) validation method proves control over a domain name by requiring the client to configure a TLS server to respond to specific connection attempts utilizing the ALPN extension with identifying information*. [[draft-ietf-acme-tls-alpn-05](https://tools.ietf.org/html/draft-ietf-acme-tls-alpn-05#page-3)].

As a side note, autocert [added support for TLS-ALPN-01](https://github.com/golang/crypto/commit/c126467f60eb25f8f27e5a981f32a87e3965053f) after Let’s Encrypt announced [End-of-Life for all TLS-SNI-01 validation support](https://community.letsencrypt.org/t/march-13-2019-end-of-life-for-all-tls-sni-01-validation-support/74209).

In other words, we need to listen to HTTPS request. The good news is [autocert](https://godoc.org/golang.org/x/crypto/acme/autocert#NewListener) got you covered and can create this special [Listener](https://godoc.org/golang.org/x/crypto/acme/autocert#NewListener) with `manager.Listener()`. Now, the question is whether HTTPS and gRPC should listen on the same port or not?. Long story short, I [couldn’t make it work](https://github.com/grpc/grpc-go/issues/2729#issuecomment-508144954) with independent ports, but if both services listen on 443, it works flawlessly.

gRPC and HTTPS on the same port… say what!?. I know, just because you can doesn’t mean you should. However, the Go gRPC library provides the `ServeHTTP` method that can help us route incoming requests to the corresponding service.
*Note that *`ServeHTTP`* uses Go’s *`HTTP/2`* server implementation which is totally separate from grpc-go’s *`HTTP/2`* server. Performance and features may vary between the two paths*. [[go-grpc](https://godoc.org/google.golang.org/grpc#Server.ServeHTTP)]. You can check some benchmarks in [gRPC serveHTTP performance penalty](https://github.com/grpc/grpc-go/issues/586). Having said that, routing would then look like this:

```go
func grpcHandlerFunc(g *grpc.Server, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if r.ProtoMajor == 2 && strings.Contains(ct, "application/grpc") {
			g.ServeHTTP(w, r)
		} else {
			h.ServeHTTP(w, r)
		}
	})
}
```

So we can listen to requests as follows, notice we provide the handler `grpcHandlerFunc` we just created to `http.Serve`:

```go
// Listener
lis = manager.Listener()

if err = http.Serve(lis, grpcHandlerFunc(s, httpsHandler())); err != nil {
	log.Fatalf("failed to serve: %v", err))
}
```

You can reproduce this by running `make run-server-public` in one tab and `make run-client-default` in another. For this to work, you need to own a domain (`HOST`). In my case I used:

```bash
export HOST=grpc.nleiva.com
export PORT=443
```

Now, I can make gRPC requests from anywhere in the world over the Internet with:

```bash
$ export HOST=grpc.nleiva.com
$ export PORT=443
$ make run-client-default
User found:  Nicolas
```

Finally, we can take a look at the certificate generated by making an HTTPS request on your browser to [https://grpc.nleiva.com/](https://grpc.nleiva.com/).

## Conclusion

There are different ways to go about setting TLS for gRPC. Providing integrity and privacy doesn’t take too much effort, so it’s strongly recommended you stay away of methods like `WithInsecure()` or setting `InsecureSkipVerify` flag to `true`.

Also, managing and distributing certificates for your gRPC endpoints shouldn’t be a hassle if you leverage some of the resources discussed in this post.

If you have any questions, feel free to contact me! I'm nleiva on [GitHub](https://github.com/nleiva) and nleiv4 on [Twitter](https://twitter.com/nleiv4).

## Links

* [Practical guide to securing gRPC connections with Go and TLS — Part 1](https://itnext.io/practical-guide-to-securing-grpc-connections-with-go-and-tls-part-1-f63058e9d6d1?source=friends_link&sk=7d9921af5742be00506bb38e50be1a66)
* [Practical guide to securing gRPC connections with Go and TLS — Part 2](https://itnext.io/practical-guide-to-securing-grpc-connections-with-go-and-tls-part-2-994ef93b8ea9?source=friends_link&sk=de526794eb30887988c9c78cf077fdf6)
* [Understanding Public Key Infrastructure and X.509 Certificates](https://www.linuxjournal.com/content/understanding-public-key-infrastructure-and-x509-certificates) by [Jeff Woods](https://www.linkedin.com/in/jeff-woods-a50b921)
* [gRPC Client Authentication](https://jbrandhorst.com/post/grpc-auth/) by [Johan Brandhorst](https://twitter.com/JohanBrandhorst)
* [Secure gRPC with TLS/SSL](https://bbengfort.github.io/programmer/2017/03/03/secure-grpc.html) by
[Benjamin Bengfort](https://twitter.com/bbengfort)
* [The Go Programmer’s Guide to Secure Connections](https://www.youtube.com/watch?v=kxKLYDLzuHA) by [Liz Rice](https://twitter.com/lizrice)
* [Build Your Own Certificate Authority with Vault](https://learn.hashicorp.com/vault/secrets-management/sm-pki-engine)
* [Automatic TLS certificate distribution with Vault](https://jbrandhorst.com/post/certify/)
* [The ACME Protocol is an IETF Standard](https://letsencrypt.org/2019/03/11/acme-protocol-ietf-standard.html)