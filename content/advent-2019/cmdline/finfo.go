package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	var jsonOut bool
	flag.BoolVar(&jsonOut, "json", false, "output in JSON format")
	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatal("error: wrong number of arguments")
	}

	write := writeText
	if jsonOut {
		write = writeJSON
	}

	fi, err := os.Stat(flag.Arg(0))
	if err != nil {
		log.Fatalf("error: %s\n", err)
	}

	m := map[string]interface{}{
		"size":     fi.Size(),
		"dir":      fi.IsDir(),
		"modified": fi.ModTime(),
		"mode":     fi.Mode(),
	}
	write(m)
}

func writeText(m map[string]interface{}) {
	for k, v := range m {
		fmt.Printf("%s: %v\n", k, v)
	}
}

func writeJSON(m map[string]interface{}) {
	m["mode"] = m["mode"].(os.FileMode).String()
	json.NewEncoder(os.Stdout).Encode(m)
}
