package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

var config struct {
	port int
	host string
}

const (
	httpdUsage = `usage: %s httpd
Run HTTP server

Options:
`
	checkUsage = "usage: %s check URL\n"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s check|httpd\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(os.Args) < 2 {
		log.Fatalf("error: wrong number of arguments")
	}

	switch os.Args[1] {
	case "httpd":
		runHTTPD()
	case "check":
		runCheck()
	default:
		log.Fatalf("error: unknown command - %s", os.Args[1])
	}

}

func runCheck() {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), checkUsage, os.Args[0])
		fs.PrintDefaults()
	}

	fs.Parse(os.Args[2:])
	if fs.NArg() != 1 {
		log.Fatalf("error: wrong number of arguments")
	}

	url := fs.Arg(0)
	resp, err := http.Get(url)
	switch {
	case err != nil:
		log.Fatalf("error: %s", err)
	case resp.StatusCode != http.StatusOK:
		log.Fatalf("error: bad status - %s", resp.Status)
	}
}

func runHTTPD() {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	fs.Var(PortVar(&config.port), "port", "port to listen on")
	fs.StringVar(&config.host, "host", config.host, "host to listen on")
	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), httpdUsage, os.Args[0])
		fs.PrintDefaults()
	}
	fs.Parse(os.Args[2:])

	http.HandleFunc("/", handler)
	addr := fmt.Sprintf("%s:%d", config.host, config.port)
	fmt.Printf("server ready on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("error: %s", err)
	}
}

func PortVar(port *int) *portVar {
	return &portVar{port}
}

type portVar struct {
	port *int
}

func (p *portVar) String() string {
	if p.port == nil {
		return ""
	}

	return fmt.Sprintf("%d", *p.port)
}

func (p *portVar) Set(s string) error {
	val, err := strconv.Atoi(s)
	if err != nil {
		return err
	}

	const minPort, maxPort = 1, 65535
	if val < minPort || val > maxPort {
		return fmt.Errorf("port %d out of range [%d:%d]", val, minPort, maxPort)
	}

	*p.port = val
	return nil
}

func init() {
	// Set defaults
	s := os.Getenv("HTTPD_PORT")
	p, err := strconv.Atoi(s)
	if err == nil {
		config.port = p
	} else {
		config.port = 8080
	}

	h := os.Getenv("HTTPD_HOST")
	if len(h) > 0 {
		config.host = h
	} else {
		config.host = "localhost"
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello Gophers\n")
}
