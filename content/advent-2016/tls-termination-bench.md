+++
linktitle = "Where to Terminate TLS?"
title = "Go, Nginx, and TLS Termination"
author = [
  "Ian Chiles (@fortytw2)",
]
date = "2016-12-03T00:00:00"
series = ["Advent 2016"]

+++

With the advent of letsencrypt, it's now easier than ever before to ensure all 
of your web applications and services are behind HTTPS. However, many times it's
hard to realize the performance impact and overhead of using HTTPS on your 
applications. Should you terminate in Nginx? Go? Stunnel? ELBs?

Luckily, it's fairly easy to find out with a simple benchmark. We'll put a 
Hello World server, written in Go, behind Nginx, set up as a SSL-terminating
reverse proxy, and compare that to the native `http.ListenAndServeTLS`. 

I set up a very small test case for this, (find it [here](https://github.com/fortytw2/dirty-ssl-bench))
to compare the two. Looking at Nginx 1.10.2 w/ OpenSSL 1.0.2j vs Go 1.7.3, the 
quick (and not very scientific) benchmark using [vegeta](https://github.com/tsenart/vegeta) shows us the following 


Nginx (reverse proxy to Go)
```
fortytw2@fortytw2 ~ % echo "GET https://localhost:8081/" | vegeta attack -duration=30s -insecure | tee results.bin | vegeta report


Requests      [total, rate]            1500, 50.03
Duration      [total, attack, wait]    29.98051442s, 29.979999935s, 514.485µs
Latencies     [mean, 50, 95, 99, max]  682.338µs, 686.243µs, 751.002µs, 1.677588ms, 18.316751ms
Bytes In      [total, mean]            18000, 12.00
Bytes Out     [total, mean]            0, 0.00
Success       [ratio]                  100.00%
Status Codes  [code:count]             200:1500
Error Set:
```

Go (`http.ListenAndServeTLS`)
```
fortytw2@fortytw2 ~ % echo "GET https://localhost:8080/" | vegeta attack -duration=30s -insecure | tee results.bin | vegeta report


Requests      [total, rate]            1500, 50.03
Duration      [total, attack, wait]    29.980323182s, 29.979999924s, 323.258µs
Latencies     [mean, 50, 95, 99, max]  491.133µs, 434.315µs, 735.471µs, 1.18063ms, 19.708664ms
Bytes In      [total, mean]            18000, 12.00
Bytes Out     [total, mean]            0, 0.00
Success       [ratio]                  100.00%
Status Codes  [code:count]             200:1500
Error Set:
```

Admittedly, this is a fairly unfair benchmark, as the Nginx benchmark has the overhead of 
reverse proxying in it. However, this is fair, as we care most about "real world" numbers, 
not arbitrary SSL termination benchmarks, as you can't exactly terminate SSL in Nginx and then
use it for anything if there's no proxying going on (in most cases).

It's very nice to see Go have lower non-tail latencies, as the last time I ran a benchmark like this (Go 1.2ish era)
Go was rather far behind Nginx, even with the reverse proxy overhead. In conclusion,
it's probably no longer worth it to just run Nginx in front of your Go app to terminate SSL and proxy to Go, as 
Go is perfectly capable of doing so performantly (and has a _awesome_ variety of easy to use letsencrypt clients, which
nginx still lacks native support for).

If you have ideas to improve this benchmark (or if I did something horribly wrong...) find me anywhere as fortytw2 :)
