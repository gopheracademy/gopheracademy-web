package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"reflect"
	"sort"

	"github.com/pkg/errors"
)

func alignSize(nums []int) int {
	size := 0
	for _, n := range nums {
		if s := int(math.Log10(float64(n))) + 1; s > size {
			size = s
		}
	}

	return size
}

// Point is a 2D point
type Point struct {
	X int
	Y int
}

var authInfoFields []string

// ACL bits
const (
	ReadACL = 1 << iota
	WriteACL
	AdminACL

	keyMask = "*****"
)

// AuthInfo is authentication information
type AuthInfo struct {
	Login  string // Login user
	ACL    uint   // ACL bitmask
	APIKey string // API key
}

// String implements Stringer interface
func (ai *AuthInfo) String() string {
	return fmt.Sprintf("Login:%s, ACL:%08b, APIKey: %s", ai.Login, ai.ACL, keyMask)
}

// Format implements fmt.Formatter
func (ai *AuthInfo) Format(state fmt.State, verb rune) {
	switch verb {
	case 's', 'q':
		val := ai.String()
		if verb == 'q' {
			val = fmt.Sprintf("%q", val)
		}
		fmt.Fprint(state, val)
	case 'v':
		if state.Flag('#') {
			// Emit type before
			fmt.Fprintf(state, "%T", ai)
		}
		fmt.Fprint(state, "{")
		val := reflect.ValueOf(*ai)
		for i, name := range authInfoFields {
			if state.Flag('#') || state.Flag('+') {
				fmt.Fprintf(state, "%s:", name)
			}
			if name == "APIKey" {
				fmt.Fprint(state, keyMask)
			} else {
				fmt.Fprint(state, val.FieldByName(name))
			}
			if i < len(authInfoFields)-1 {
				fmt.Fprint(state, " ")
			}
		}
		fmt.Fprint(state, "}")
	}
}

func init() {
	typ := reflect.TypeOf(AuthInfo{})
	authInfoFields = make([]string, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		authInfoFields[i] = typ.Field(i).Name
	}
	sort.Strings(authInfoFields)
}

type Config struct{}

func loadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "can't open config file")
	}
	defer file.Close()

	return &Config{}, nil
}

func main() {
	var e interface{} = 2.7182
	fmt.Printf("e = %v (%T)\n", e, e)
	fmt.Printf("%10d\n", 353)
	fmt.Printf("%*d\n", 10, 353)
	nums := []int{12, 237, 3878, 3}
	size := alignSize(nums)
	for i, n := range nums {
		fmt.Printf("%2d %*d\n", i, size, n)
	}

	fmt.Printf("The price of %[1]s was $%[2]d. $%[2]d! imagine that.\n", "carrot", 23)

	p := &Point{1, 2}
	fmt.Printf("%v %+v %#v \n", p, p, p)
	ai := &AuthInfo{
		Login:  "daffy",
		ACL:    ReadACL | WriteACL,
		APIKey: "duck season",
	}
	fmt.Println(ai.String())
	fmt.Printf("ai %%s: %s\n", ai)
	fmt.Printf("ai %%q: %q\n", ai)
	fmt.Printf("ai %%v: %v\n", ai)
	fmt.Printf("ai %%+v: %+v\n", ai)
	fmt.Printf("ai %%#v: %#v\n", ai)

	cfg, err := loadConfig("/no/such/file.toml")
	if err != nil {
		fmt.Printf("error: %s\n", err)
		log.Printf("can't load config\n%+v", err)
	}

	fmt.Println("cfg", cfg)
}
