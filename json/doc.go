// Package json provides case-sensitive JSON unmarshaling functionality.
//
// Unlike the standard encoding/json package which performs case-insensitive
// field matching, this package requires exact field name matches (respecting json tags).
//
// This is particularly useful when:
//   - You need strict validation of JSON field names
//   - There's a risk of duplicate fields with different cases
//   - You want to ensure only exact field matches are accepted
//   - You're building APIs that require rigorous validation
//
// Basic Usage:
//
//	import (
//	    stdjson "encoding/json"
//	    csjson "github.com/go-mg/casesensitive/json"
//	)
//
//	type User struct {
//	    Name  string `json:"name"`
//	    Email string `json:"email"`
//	}
//
//	payload := `{"name":"john", "NAME":"doe", "email":"john@example.com"}`
//
//	// Standard encoding/json (case-insensitive)
//	var user1 User
//	stdjson.Unmarshal([]byte(payload), &user1)
//	// Result: user1.Name = "doe" (last value wins)
//
//	// casesensitive/json (case-sensitive)
//	var user2 User
//	csjson.Unmarshal([]byte(payload), &user2)
//	// Result: user2.Name = "john" (only exact "name" match)
//
// Decoder with Strict Validation:
//
//	decoder := csjson.NewDecoder(strings.NewReader(payload))
//	decoder.DisallowUnknownFields()
//	err := decoder.Decode(&user)
//	// Returns error: json: unknown field "NAME"
//
// JSON Tags Support:
//
// The package respects json tags the same way as encoding/json:
//
//	type User struct {
//	    Name     string `json:"name"`          // Maps to "name"
//	    Email    string `json:"email"`         // Maps to "email"
//	    Password string `json:"-"`             // Ignored
//	    Age      int    `json:"age,omitempty"` // Maps to "age"
//	    Nickname string                        // No tag: maps to "Nickname" (exact)
//	}
//
// Performance Considerations:
//
// This package is slightly slower than encoding/json due to additional validation.
// Use encoding/json when performance is critical and case-sensitivity is not required.
package json
