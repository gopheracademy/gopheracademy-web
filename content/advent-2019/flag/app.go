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
	flag.Usage = func() { // [1]
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s check|run\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(os.Args) < 2 { // [2]
		log.Fatalf("error: wrong number of arguments")
	}

	var err error
	switch os.Args[1] { // [3]
	case "run":
		err = runHTTPD()
	case "check":
		err = checkHTTPD()
	default:
		err = fmt.Errorf("error: unknown command - %s", os.Args[1])
	}

	if err != nil {
		log.Fatalf("error: %s", err)
	}

}

func checkHTTPD() error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError) // [4]
	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), checkUsage, os.Args[0])
		fs.PrintDefaults()
	}

	if err := fs.Parse(os.Args[2:]); err != nil { // [5]
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("error: wrong number of arguments")
	}

	url := fs.Arg(0) // [6]
	resp, err := http.Get(url)
	switch {
	case err != nil:
		return err
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("error: bad status - %s", resp.Status)
	}

	return nil
}

func runHTTPD() error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.Var(PortVar(&config.port), "port", "port to listen on")
	fs.StringVar(&config.host, "host", config.host, "host to listen on")
	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), httpdUsage, os.Args[0])
		fs.PrintDefaults()
	}

	if err := fs.Parse(os.Args[2:]); err != nil {
		return err
	}

	http.HandleFunc("/", handler)
	addr := fmt.Sprintf("%s:%d", config.host, config.port)
	fmt.Printf("server ready on %s\n", addr)
	return http.ListenAndServe(addr, nil)
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
