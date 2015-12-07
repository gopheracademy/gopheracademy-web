+++
author = ["Brian Ketelsen"]
date = "2015-12-06T08:28:59-05:00"
linktitle = "Superdog"
series = ["Advent 2015"]
title = "Smart Cryptography with Superdog and Vault"

+++

## Superdog - the Crypto library for Vault from Hashicorp

At [XOR Data Exchange](http://xor.exchange) we deal with a lot of sensitive data for our customers.  We needed to be able to support strong encryption with key rotation in a simple and practical way.  So we wrote `superdog` to help with this task. Superdog is a library for managing strong cryptography in both development/test environments.  Superdog provides an elegant wrapper to the [Vault](https://www.vaultproject.io) API that allows you to manage your cryptographic keys using any code that implements the `KeyProvider` interface.  An implemention of the `KeyProvider` interface is provided for Vault, but others could be supported.

Using Vault allows us to have secure and strong separation between the keys and the encrypted text.  Superdog takes that separation a step further by allowing us to use our encryption routines in our Development/Test/Staging environments without worrying about having an instance of Vault available.  We provide this abstraction using a `KeyProvider` interface.  By default, the `DevKeyProvider` is used.  DevKeyProvider is an insecure provider that always uses the same keys and initialization vectors.  It should *never ever* be used in Production.

### Features

-  Versioned Keys - Key version is stored as the first few bytes of the encrypted text
-  Key Rotation - Rotate your keys safely, knowing that you'll always be able to decrypt older versionss
-  Development implementation for tests and local development
-  Versioned and Rotated IV/Salt - `SaltProvider` interface works the same as `KeyProvider` to allow development and testing access to the crypto libraries without requiring a live Key (Vault) server
-  `Reencrypt` function to simplify key rotation, decrypts with given key, reencrypts with latest key

### Cypher Suites

`superdog` supports AES encryption with CFB/CTR/GCM/OFB modes.


### Performance

On Go version 1.5.2 / Linux x86_64 kernel 4.2.5 on a quad-core i7:

```
BenchmarkKeyEncryptCFB-8	 1000000	      2024 ns/op
BenchmarkKeyEncryptCTR-8	  500000	      2748 ns/op
BenchmarkKeyEncryptGCM-8	 1000000	      2381 ns/op
BenchmarkKeyEncryptOFB-8	  500000	      2665 ns/op
BenchmarkKeyDecryptCFB-8	10000000	       215 ns/op
BenchmarkKeyDecryptCTR-8	 2000000	       898 ns/op
BenchmarkKeyDecryptGCM-8	 3000000	       520 ns/op
BenchmarkKeyDecryptOFB-8	 2000000	       817 ns/op
```

### Usage

`go get -u github.com/xordataexchange/superdog/...`

#### Encryption
```go
val := []byte("encrypt me!")

// use a key prefix to delineate different crypto keys
// allowing you to use different keys for different parts of your application
// or different fields of a database table, for example
b, err := superdog.Encrypt("mykeyprefix", val, val)
if err != nil {
	// handle error
}
```
#### Decryption
```go
b := []byte["some crypt cypher text here"]
decrypted, err := superdog.Decrypt([]byte("mykeyprefix", b, b)
if err != nil {
	// handle error
}

```

#### Production Usage
By default, `superdog` uses the `DevKeyProvider` which is a static key with static IV.  This is extremely insecure, and SHOULD NOT ever be used in production.

We reccommend using Go's [build tags](https://golang.org/pkg/go/build/) to enable strong cryptography in production usage.

Create a file with your connection routines in the init() function.  Add the build tag `// +build production` to the top of that file.  Here's an incomplete example:

```go
// +build production

package main
import (
	"github.com/xordataexchange/superdog"
	"github.com/xordataexxchange/superdog/vault/hashi"
	"github.com/hashicorp/vault/api"
)

// Assign each application a unique UUID
// and use Vault's AppID authentication mechanism
const (
	appid = "SOME RANDOM UUID"
)

func init() {
	user := os.Getenv("VAULT_USER")
	vaultaddr := os.Getenv("VAULT_ADDRESS")
	// TEST these for empty strings & handle appropriately in your code

	cfg:= api.DefaultConfig()
	cfg.Address = vaultaddr

	vault, err := hashi.NewVault(cfg)
	if err != nil {
		// handle appropriately
	}
	err = vault.AuthAppID(appid, user)
	if err != nil {
		// handle appropriately
	}

	crypto.DefaultKeyProvider = vault
	crypto.DefaultSaltProvider = vault

}

```
Now compile your program with `go build -tags production` to include this code.  The `KeyProvider` will be set to use Vault.

#### Contributions Welcome
We hope you enjoy using Superdog, and we're always excited to accept contributions from the community.  Superdog helps us support our mission of providing safe and secure access to data using responsible security measures.  The original implementation of Superdog was written by Erik St. Martin.
