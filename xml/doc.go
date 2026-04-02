// Package xml provides case-sensitive XML unmarshaling functionality.
//
// Unlike the standard encoding/xml package which performs case-insensitive
// field matching, this package requires exact element name matches (respecting xml tags).
//
// This is particularly useful when:
//   - You need strict validation of XML element names
//   - There's a risk of duplicate elements with different cases
//   - You want to ensure only exact element matches are accepted
//   - You're building APIs that require rigorous validation
//
// Basic Usage:
//
//	import (
//	    stdxml "encoding/xml"
//	    csxml "github.com/go-mg/casesensitive/xml"
//	)
//
//	type User struct {
//	    Name  string `xml:"name"`
//	    Email string `xml:"email"`
//	}
//
//	payload := `<User><name>john</name><NAME>doe</NAME><email>john@example.com</email></User>`
//
//	// Standard encoding/xml (case-insensitive)
//	var user1 User
//	stdxml.Unmarshal([]byte(payload), &user1)
//	// Result: user1.Name = "doe" (last value wins)
//
//	// casesensitive/xml (case-sensitive)
//	var user2 User
//	csxml.Unmarshal([]byte(payload), &user2)
//	// Result: user2.Name = "john" (only exact "name" match)
//
// XML Tags Support:
//
// The package respects xml tags the same way as encoding/xml:
//
//	type User struct {
//	    Name     string `xml:"name"`          // Maps to "name"
//	    Email    string `xml:"email"`         // Maps to "email"
//	    Password string `xml:"-"`             // Ignored
//	    Age      int    `xml:"age,omitempty"` // Maps to "age"
//	    Nickname string                       // No tag: maps to "Nickname" (exact)
//	}
//
// Performance Considerations:
//
// This package is slightly slower than encoding/xml due to additional validation.
// Use encoding/xml when performance is critical and case-sensitivity is not required.
package xml
