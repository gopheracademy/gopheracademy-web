package main

import (
	"flag"
	"fmt"
	"time"
)

var spinChars = `|/-\`

type Spinner struct {
	message string
	i       int
}

func NewSpinner(message string) *Spinner {
	return &Spinner{message: message}
}

func (s *Spinner) Tick() {
	fmt.Printf("%s %c \r", s.message, spinChars[s.i])
	s.i = (s.i + 1) % len(spinChars)
}

func main() {
	flag.Parse()
	s := NewSpinner("working...")
	for i := 0; i < 100; i++ {
		s.Tick()
		time.Sleep(100 * time.Millisecond)
	}

}
