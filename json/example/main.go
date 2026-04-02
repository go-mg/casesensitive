package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	zjson "github.com/go-mg/casesensitive/json"
)

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	payload := `{"name":"michael", "NAME":"jackson","email":"whosbad@gmail.com"}`

	fmt.Println("=== Comparison: encoding/json vs zjson ===")

	// Method 1: Standard encoding/json (case-insensitive)
	fmt.Println("\n1. encoding/json.Unmarshal (case-insensitive):")
	var user1 User
	err1 := json.Unmarshal([]byte(payload), &user1)
	if err1 != nil {
		log.Printf("error: %v\n", err1)
	}
	fmt.Printf("   Result: %+v\n", user1)
	fmt.Printf("   Name: %q (last value wins - 'jackson')\n", user1.Name)

	// Method 2: zjson.Unmarshal (case-sensitive)
	fmt.Println("\n2. zjson.Unmarshal (case-sensitive):")
	var user2 User
	err2 := zjson.Unmarshal([]byte(payload), &user2)
	if err2 != nil {
		log.Printf("error: %v\n", err2)
	}
	fmt.Printf("   Result: %+v\n", user2)
	fmt.Printf("   Name: %q (only lowercase 'name' accepted - 'michael')\n", user2.Name)

	// Method 3: zjson.Decoder (case-sensitive)
	fmt.Println("\n3. zjson.NewDecoder (case-sensitive):")
	var user3 User
	decoder := zjson.NewDecoder(strings.NewReader(payload))
	err3 := decoder.Decode(&user3)
	if err3 != nil {
		log.Printf("error: %v\n", err3)
	}
	fmt.Printf("   Result: %+v\n", user3)
	fmt.Printf("   Name: %q\n", user3.Name)

	// Method 4: zjson.Decoder with DisallowUnknownFields
	fmt.Println("\n4. zjson.NewDecoder with DisallowUnknownFields:")
	var user4 User
	decoder2 := zjson.NewDecoder(strings.NewReader(payload))
	decoder2.DisallowUnknownFields()
	err4 := decoder2.Decode(&user4)
	if err4 != nil {
		fmt.Printf("   ✓ Expected error: %v\n", err4)
		fmt.Printf("   (Uppercase 'NAME' field is rejected)\n")
	} else {
		fmt.Printf("   Result: %+v\n", user4)
	}

	// Test with valid payload
	validPayload := `{"name":"michael","email":"whosbad@gmail.com"}`
	fmt.Println("\n5. zjson with valid payload (no duplicate fields):")
	var user5 User
	decoder3 := zjson.NewDecoder(strings.NewReader(validPayload))
	decoder3.DisallowUnknownFields()
	err5 := decoder3.Decode(&user5)
	if err5 != nil {
		log.Printf("error: %v\n", err5)
	}
	fmt.Printf("   Result: %+v\n", user5)
	fmt.Printf("   ✓ Success! All fields match exactly\n")

	// Demonstration with struct without tags
	type Person struct {
		Name  string // No tag: expects "Name" with capital N
		Email string // No tag: expects "Email" with capital E
	}

	fmt.Println("\n6. Struct without JSON tags (case-sensitive on field name):")

	// Payload with correct case
	correctPayload := `{"Name":"John","Email":"john@example.com"}`
	var person1 Person
	zjson.Unmarshal([]byte(correctPayload), &person1)
	fmt.Printf("   Payload: %s\n", correctPayload)
	fmt.Printf("   Result: %+v ✓\n", person1)

	// Payload with incorrect case
	wrongPayload := `{"name":"John","email":"john@example.com"}`
	var person2 Person
	zjson.Unmarshal([]byte(wrongPayload), &person2)
	fmt.Printf("   Payload: %s\n", wrongPayload)
	fmt.Printf("   Result: %+v (empty fields - case doesn't match)\n", person2)

	fmt.Println("\n=== Conclusion ===")
	fmt.Println("✓ zjson ensures exact (case-sensitive) field matching")
	fmt.Println("✓ Prevents bugs caused by duplicate fields with different cases")
	fmt.Println("✓ Supports JSON tags and structs without tags")
	fmt.Println("✓ DisallowUnknownFields rejects fields with incorrect case")
	fmt.Println("✓ Protects against data injection (trailing data)")

	// Demonstration of injection protection
	fmt.Println("\n=== Protection Against Data Injection ===")

	injectionPayload := `haha{"name":"michael","email":"whosbad@gmail.com"}kkkk`

	fmt.Println("\n7. encoding/json.Decoder with leading data:")
	fmt.Printf("   Payload: %s\n", injectionPayload)
	var user6 User
	decoder4 := json.NewDecoder(strings.NewReader(injectionPayload))
	err6 := decoder4.Decode(&user6)
	if err6 != nil {
		fmt.Printf("   ✓ Rejects: %v\n", err6)
	} else {
		fmt.Printf("   ⚠️  VULNERABLE: Accepts payload with extra data\n")
		fmt.Printf("   Result: %+v\n", user6)
	}

	fmt.Println("\n8. zjson.Decoder with leading data:")
	fmt.Printf("   Payload: %s\n", injectionPayload)
	var user7 User
	decoder5 := zjson.NewDecoder(strings.NewReader(injectionPayload))
	err7 := decoder5.Decode(&user7)
	if err7 != nil {
		fmt.Printf("   ✓ PROTECTED: Rejects payload with leading data\n")
		fmt.Printf("   Error: %v\n", err7)
	} else {
		fmt.Printf("   ⚠️  Result: %+v\n", user7)
	}

	// Test with trailing data only
	trailingPayload := `{"name":"michael","email":"whosbad@gmail.com"}extra_data`

	fmt.Println("\n9. Payload with trailing data (no leading):")
	fmt.Printf("   Payload: %s\n", trailingPayload)

	var user8 User
	decoder6 := json.NewDecoder(strings.NewReader(trailingPayload))
	err8 := decoder6.Decode(&user8)
	if err8 != nil {
		fmt.Printf("   encoding/json rejects: %v\n", err8)
	} else {
		fmt.Printf("   ⚠️  encoding/json ACCEPTS (vulnerable): %+v\n", user8)
	}

	var user9 User
	decoder7 := zjson.NewDecoder(strings.NewReader(trailingPayload))
	err9 := decoder7.Decode(&user9)
	if err9 != nil {
		fmt.Printf("   ✓ zjson rejects (default): %v\n", err9)
	} else {
		fmt.Printf("   zjson accepts: %+v\n", user9)
	}

	var user10 User
	decoder8 := zjson.NewDecoder(strings.NewReader(trailingPayload))
	decoder8.AllowTrailingData()
	err10 := decoder8.Decode(&user10)
	if err10 != nil {
		fmt.Printf("   zjson with AllowTrailingData rejects: %v\n", err10)
	} else {
		fmt.Printf("   ⚠️  zjson with AllowTrailingData ACCEPTS: %+v\n", user10)
		fmt.Printf("   (Use AllowTrailingData only for streams with multiple JSONs)\n")
	}

	// Test with multiple JSONs (valid case for AllowTrailingData)
	multipleJSONs := `{"name":"john","email":"test1@example.com"}{"name":"jane","email":"test2@example.com"}`

	fmt.Println("\n10. Multiple JSONs in sequence (valid case for AllowTrailingData):")
	fmt.Printf("    Payload: %s\n", multipleJSONs)

	var user11 User
	decoder9 := zjson.NewDecoder(strings.NewReader(multipleJSONs))
	err11 := decoder9.Decode(&user11)
	if err11 != nil {
		fmt.Printf("    ✓ zjson (default) rejects: %v\n", err11)
	} else {
		fmt.Printf("    zjson accepts: %+v\n", user11)
	}

	var user12 User
	decoder10 := zjson.NewDecoder(strings.NewReader(multipleJSONs))
	decoder10.AllowTrailingData()
	err12 := decoder10.Decode(&user12)
	if err12 != nil {
		fmt.Printf("    zjson with AllowTrailingData rejects: %v\n", err12)
	} else {
		fmt.Printf("    ✓ zjson with AllowTrailingData accepts first JSON: %+v\n", user12)
		fmt.Printf("    (Can continue reading next JSON from stream)\n")
	}
}
