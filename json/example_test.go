package json_test

import (
	"fmt"
	"strings"

	"github.com/go-mg/casesensitive/json"
)

func ExampleUnmarshal() {
	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	payload := `{"name":"john", "NAME":"hacker", "email":"john@example.com"}`

	var user User
	json.Unmarshal([]byte(payload), &user)

	fmt.Printf("Name: %s\n", user.Name)
	fmt.Printf("Email: %s\n", user.Email)
	// Output:
	// Name: john
	// Email: john@example.com
}

func ExampleUnmarshal_array() {
	type Item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	payload := `[{"id":1,"name":"first"},{"id":2,"name":"second"}]`

	var items []Item
	json.Unmarshal([]byte(payload), &items)

	for _, item := range items {
		fmt.Printf("ID: %d, Name: %s\n", item.ID, item.Name)
	}
	// Output:
	// ID: 1, Name: first
	// ID: 2, Name: second
}

func ExampleDecoder() {
	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	payload := `{"name":"john","email":"john@example.com"}`

	var user User
	decoder := json.NewDecoder(strings.NewReader(payload))
	err := decoder.Decode(&user)
	if err != nil {
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
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	payload := `{"name":"john","NAME":"hacker","email":"john@example.com"}`

	var user User
	decoder := json.NewDecoder(strings.NewReader(payload))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&user)

	fmt.Printf("Error: %v\n", err)
	// Output:
	// Error: json: unknown field "NAME"
}

func ExampleDecoder_trailingDataRejected() {
	type User struct {
		Name string `json:"name"`
	}

	payload := `{"name":"john"}extra_data`

	var user User
	decoder := json.NewDecoder(strings.NewReader(payload))
	err := decoder.Decode(&user)

	fmt.Printf("Error: %v\n", err)
	// Output:
	// Error: json: invalid character after top-level value
}
