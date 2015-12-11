+++
author = ["Matt Holt"]
date = "2015-12-15T08:00:00-07:00"
series = ["Advent 2015"]
title = "Generate and Use Free TLS Certificates with Lego"
+++

If your Go program uses the transport layer of the network at all&mdash;whether to serve static files, an API, or something else over the wire&mdash;you should be encrypting connections using TLS. Hopefully this is obvious by now. But developers still don't do it. Why?

Because TLS (formerly "SSL") certificates cost money and require manual labor to obtain, install, and maintain. Besides, there's no reason to encrypt unless you collect or send sensitive data, right? **Wrong.** Encryption not only prevents eavesdropping and surveillance, it also protects packets from being modified in flight&mdash;modifications that could break your API or track your users. Essentially, TLS adds a layer of privacy and integrity to your application.

This post will guide you through a free, easy way to obtain real, trusted TLS certificates using Go. Thanks to the efforts of the Internet Security Research Group (ISRG) and, in particular, [Let's Encrypt](https://letsencrypt.org), the ACME protocol makes it possible to do this. Sebastian Erhart has done an excellent job building an ACME client in Go called [lego](https://github.com/xenolf/lego) that we can use to get free, valid TLS certificates in seconds. The technique shown here is very similar to what [Caddy](https://caddyserver.com) does to serve your sites over HTTPS by default.

(Note that lego can be used as a stand-alone CLI tool as well.)


## Definitions

There's been a lot of confusion about ACME, Let's Encrypt, and this whole "free certificates" thing, so first, a few clarifications:

- **ACME** is the protocol that facilitates the automatic issuance, renewal, and revocation of x.509 certificates between certificate authorities and applicants. At time of writing, the spec is still a [working draft](https://github.com/ietf-wg-acme/acme/) at the IETF.

- **ISRG** is [the non-profit organization](https://letsencrypt.org/isrg/) behind Let's Encrypt.

- **Let's Encrypt** is the first certificate authority (CA) to implement the ACME protocol.

- **Domain Validation (DV) Certificates** are issued once a CA is convinced you own the domain you are requesting a certificate for. Let's Encrypt issues DV certs. Make no mistake: all DV certificates are technically the same. A free, automated DV cert offers no fewer benefits than one costing $10 or $20.

Currently, the only ACME-based CA is Let's Encrypt, so for now, the terms "ACME client" and "Let's Encrypt client" are mostly interchangeable. This may not always be the case, however, so pay attention to library docs and implementation details in the future. (For example, Let's Encrypt's CA server software is [Boulder](https://github.com/letsencrypt/boulder), but not all Boulder features are defined in the ACME spec.)

Alright, now let's encrypt your Go program's transmissions.


## Getting Started

Import the [acme package](https://godoc.org/github.com/xenolf/lego/acme):

```go
import "github.com/xenolf/lego/acme"
```


The first thing we'll need is a type that implements the acme.User interface. Before a certificate can be issued, you'll need to register an account with the CA. In other words, you give the CA your public key (and optional email). From the ACME spec:

> ACME functions much like a traditional CA, in which a user creates an account, adds identifiers to that account (proving control of the domains), and requests certificate issuance for those domains while logged in to the account.

Here's an example implementation:

```go
type MyUser struct {
    Email        string
    Registration *acme.RegistrationResource
    key          *rsa.PrivateKey
}
func (u MyUser) GetEmail() string {
    return u.Email
}
func (u MyUser) GetRegistration() *acme.RegistrationResource {
    return u.Registration
}
func (u MyUser) GetPrivateKey() *rsa.PrivateKey {
    return u.key
}
```

If you don't have one already, generate a private key for the user:

```go
const rsaKeySize = 2048
privateKey, err := rsa.GenerateKey(rand.Reader, rsaKeySize)
if err != nil {
    log.Fatal(err)
}
myUser := MyUser{
    Email: "you@yours.com",
    key: privateKey,
}
```

The email address is optional, but highly recommended so you can recover your account later if you lose your private key. Also, don't lose (or lose control of) any of the private keys we generate in this process. That means once you generate a key, you should save it for next time. How you do this is up to you.


## Setting up a Client

Now we can create a client for the user that will speak ACME with the CA server.

```go
client, err := acme.NewClient("https://acme-v01.api.letsencrypt.org/directory", &myUser, rsaKeySize, "")
if err != nil {
    log.Fatal(err)
}
```

The acme package expects the URL to the CA server's ACME directory. In this case, we're using Let's Encrypt's live endpoint. If you're just testing, use their staging endpoint, `https://acme-staging.api.letsencrypt.org/directory`. The staging endpoint is mostly the same, except it returns untrusted certificates and doesn't have the same rate limits.

We also pass in our user and the size of the key for generated certs. The last argument to NewClient is an optional alternate port to bind to; we left it blank to use the default port, which depends on the challenge (see below). You can leave it blank if your Go program has permission to bind to low ports.

If your user is new, you will have to register it with the CA:

```go
reg, err := client.Register()
if err != nil {
    log.Fatal(err)
}
myUser.Registration = reg
```

Persist the user (with its registration resource) somewhere so you can reuse it next time.

Most CAs (including Let's Encrypt) have terms of service. You will have to agree to them if you haven't already or they changed since the last agreement:

```go
err = client.AgreeToTOS()
if err != nil {
    log.Fatal(err)
}
```

Now with all the paperwork out of the way, we can finally get to the exciting stuff.


## Obtaining Certificates

You can obtain one certificate per domain name (`client.ObtainCertificates`), or you can obtain a single SAN certificate (`client.ObtainSANCertificate`) which is valid for multiple domain names. SAN certificates are most commonly used for the "www." variant of the domain name. Typically, you'll want different certificates for different sites.

```go
certificates, failures := client.ObtainCertificates([]string{"mydomain.com", "www.mydomain.com"}, true)
if len(failures) > 0 {
    // At least one cert request failed; but some may have succeeded.
    // Make sure to save the certs and keys for those that succeeded.
    for domain, err := range failures {
        log.Printf("[%s] %v", domain, err)
    }
}
```

The second argument is whether lego should bundle the intermediate certificates for us. Usually this is `true` unless you have a good reason.

The returned certificates come with their private keys (lego generated them for us). Save the certificates and private keys somewhere safe.


## ACME Challenges

The ACME spec defines several *challenges* a client may solve to prove ownership of the domain. These are http-01, tls-sni-01, dns-01, and proofOfPosession-01 (the 01 is just a version number). At time of writing, only http-01 and tls-sni-01 are implemented by Let's Encrypt, but lego already supports those in addition to dns-01.

These challenges are solved for you by lego, but you should be aware how they work.

The **http-01** challenge requires the client to provide a special token value at a certain URL. This exchange must occur on port 80.

The **tls-sni-01** challenge requires the client to add a special token hostname to the TLS handshake. This exchange must occur on port 443.

The **dns-01** challenge requires a DNS record to be provisioned with a special token value. Thanks to [an awesome PR](https://github.com/xenolf/lego/pull/38), lego will soon be able to do this for you for AWS, CloudFlare, and RFC2136-compliant DNS providers. The benefit of this challenge is that there is no need to connect to your machine directly.

The **proofOfPosession-01** challenge requires you to prove ownership of a private key used in association with a public key already known to the server. This will be available in the future.

Right now, the challenge is chosen at random, but the next release of lego will allow you to specify which challenges you can do.

## Using Your Certificates

The certificates are returned in PEM format, ready to be written to disk or for use with net/http. For example, after writing the cert chain to cert.pem and the private key to key.pem:

```go
err := http.ListenAndServeTLS(":10443", "cert.pem", "key.pem", nil)
if err != nil {
    log.Fatal(err)
}
```

## Renewing Certificates

When a certificate is nearing its expiration, you should renew it.

```go
newCerts, err := client.RenewCertificate(certificate, false, true)
if err != nil {
    log.Fatal(err)
}
```

Pass in the old certificate resource, whether to revoke the old one (only necessary if its key was compromised), and whether the new certificate should come bundled. Only revoke a certificate if your domain, server, or private key was compromised.


## Common Problems

If you haven't noticed, there's a lot of moving parts here. Usually the failure point is your network configuration. Here are a few common troubleshooting tips, in order:

- Is your domain a registered, public domain name? Let's Encrypt does not issue certificates for internal domain names or IP addresses. (Adding a domain to your hosts file doesn't count.)

- Does your domain resolve to the machine you're running the client on? You must configure your domain name to point to the IP of the machine you're solving the challenge on.

- Are you behind a load balancer? SSL termination will break the tls-sni-01 challenge, and load balancing to a machine other than the one solving the challenge will break http-01.

- Can you bind to ports 80 and 443? The http-01 and tls-sni-01 challenges _must_ be completed on those ports. You're welcome to forward them to a higher port that your client uses (that'd be the optional alternate port I mentioned earlier), but those challenges require those ports, period.

- Are you on an IPv6-only network? Let's Encrypt's infrastructure only supports IPv4 at this time. Hopefully this changes soon.


## Stay Encrypted

Let's Encrypt and the ACME protocol make free, automatic TLS more accessible than ever. It's our responsibility as programmers to keep our applications secure, private, and reliable, and now we have no excuse not to use TLS for this purpose.

If you have questions about ACME or run into technical difficulties, try searching the [Let's Encrypt Community](https://community.letsencrypt.org) forum before opening an issue.

Also, if you find lego useful, consider [donating to Sebastian](https://www.paypal.com/cgi-bin/webscr?cmd=_s-xclick&hosted_button_id=UW82U7AYWAL96) to let him know it's awesome!
