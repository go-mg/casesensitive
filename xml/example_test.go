package xml_test

import (
	"fmt"
	"strings"

	"github.com/go-mg/casesensitive/xml"
)

func ExampleUnmarshal() {
	type User struct {
		Name  string `xml:"name"`
		Email string `xml:"email"`
	}

	payload := `<User><name>john</name><NAME>hacker</NAME><email>john@example.com</email></User>`

	var user User
	if err := xml.Unmarshal([]byte(payload), &user); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Name: %s\n", user.Name)
	fmt.Printf("Email: %s\n", user.Email)
	// Output:
	// Name: john
	// Email: john@example.com
}

func ExampleDecoder() {
	type User struct {
		Name  string `xml:"name"`
		Email string `xml:"email"`
	}

	payload := `<User><name>john</name><email>john@example.com</email></User>`

	var user User
	decoder := xml.NewDecoder(strings.NewReader(payload))
	if err := decoder.Decode(&user); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Name: %s\n", user.Name)
	fmt.Printf("Email: %s\n", user.Email)
	// Output:
	// Name: john
	// Email: john@example.com
}

func ExampleDecoder_disallowUnknownFields() {
	type User struct {
		Name  string `xml:"name"`
		Email string `xml:"email"`
	}

	payload := `<User><name>john</name><NAME>hacker</NAME><email>john@example.com</email></User>`

	var user User
	decoder := xml.NewDecoder(strings.NewReader(payload))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&user)

	fmt.Printf("Error: %v\n", err)
	// Output:
	// Error: xml: unknown field "NAME"
}
