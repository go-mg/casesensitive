# casesensitive/json - Case-Sensitive JSON Parser

JSON parser with case-sensitive validation and protection against trailing data injection.

## Problem

Go's standard `encoding/json` is case-insensitive. When there are duplicate fields with different cases, the last value overwrites:

```go
type User struct {
    Name string `json:"name"`
}

payload := `{"name":"john", "NAME":"hacker"}`
json.Unmarshal([]byte(payload), &user)
// Result: user.Name = "hacker" ❌
```

Additionally, `json.Decoder` accepts trailing data, allowing injection:

```go
payload := `{"name":"john"}malicious_data`
decoder := json.NewDecoder(strings.NewReader(payload))
decoder.Decode(&user) // ✓ Accepts, ignores trailing data ❌
```

## Solution

```go
import csjson "github.com/go-mg/casesensitive/json"

// Case-sensitive: only lowercase "name" is accepted
payload := `{"name":"john", "NAME":"hacker"}`
csjson.Unmarshal([]byte(payload), &user)
// Result: user.Name = "john" ✓

// Rejects trailing data by default
payload := `{"name":"john"}malicious_data`
decoder := csjson.NewDecoder(strings.NewReader(payload))
decoder.Decode(&user) // ✗ Error: invalid character after top-level value ✓
```

## Installation

```bash
go get github.com/go-mg/casesensitive/json
```

## Basic Usage

### Unmarshal

```go
var user User
err := csjson.Unmarshal([]byte(payload), &user)
```

### Decoder

```go
decoder := csjson.NewDecoder(r.Body)
err := decoder.Decode(&user)
```

### Strict Validation

```go
decoder := csjson.NewDecoder(r.Body)
decoder.DisallowUnknownFields() // Rejects fields with incorrect case
err := decoder.Decode(&user)
```

### Multiple JSONs (Streams)

```go
decoder := csjson.NewDecoder(reader)
decoder.AllowTrailingData() // Allows processing multiple JSONs

for {
    var obj MyStruct
    if err := decoder.Decode(&obj); err == io.EOF {
        break
    }
    // process obj
}
```

## Features

| Feature | encoding/json | casesensitive/json |
| --- | --- | --- |
| Field matching | Case-insensitive | Case-sensitive ✓ |
| Duplicate fields | Last wins | Exact match only ✓ |
| Trailing data | Accepts ⚠️ | Rejects ✓ |
| DisallowUnknownFields | Case-insensitive | Case-sensitive ✓ |
| Performance | Faster | 3-4x slower |

## Examples

### Secure API Handler

```go
func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
    // Limit size
    r.Body = http.MaxBytesReader(w, r.Body, 1*1024*1024) // 1MB
    
    // Parse with casesensitive/json
    var user User
    decoder := csjson.NewDecoder(r.Body)
    decoder.DisallowUnknownFields()
    
    if err := decoder.Decode(&user); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }
    
    // Validate content
    if err := validateUser(&user); err != nil {
        http.Error(w, "Invalid data", http.StatusBadRequest)
        return
    }
    
    // Process with prepared statements (prevents SQL injection)
    db.Exec("INSERT INTO users (name, email) VALUES ($1, $2)", 
        user.Name, user.Email)
}
```

### Struct Without Tags

```go
type Person struct {
    Name  string // Expects "Name" (with capital N)
    Email string // Expects "Email" (with capital E)
}

// ✓ Correct
payload := `{"Name":"John","Email":"john@example.com"}`

// ✗ Incorrect (case doesn't match)
payload := `{"name":"John","email":"john@example.com"}`
```

## Detailed Behavior

### Trailing Data

```go
// Valid JSON
`{"name":"john"}` → ✓ Accepts

// Trailing data
`{"name":"john"}extra` → ✗ Rejects (default)
                       → ✓ Accepts (with AllowTrailingData)

// Leading data
`extra{"name":"john"}` → ✗ Always rejects

// Multiple JSONs
`{"name":"john"}{"name":"jane"}` → ✗ Rejects (default)
                                  → ✓ Accepts first (with AllowTrailingData)
```

### Duplicate Fields

```go
type User struct {
    Name string `json:"name"`
}

payload := `{"name":"john", "NAME":"hacker"}`

// encoding/json
json.Unmarshal([]byte(payload), &user)
// user.Name = "hacker" (last wins, case-insensitive)

// casesensitive/json
csjson.Unmarshal([]byte(payload), &user)
// user.Name = "john" (only lowercase "name", case-sensitive)
```

## Responsibilities

### What casesensitive/json DOES ✅

- ✅ Case-sensitive field validation
- ✅ Protection against trailing data
- ✅ Unknown field validation
- ✅ Exact name matching

### What casesensitive/json DOES NOT DO ❌

This is a JSON parser, not a content validator:

- ❌ SQL injection in values (use prepared statements)
- ❌ XSS in values (use HTML escaping)
- ❌ XML injection in values (use XML escaping)
- ❌ Format validation (email, URL - use validator)
- ❌ DoS protection (use size middleware)
- ❌ Business rule validation

## Security Architecture

```
┌─────────────────────────────────────┐
│ 1. Infrastructure                   │
│    - Rate limiting                  │
│    - MaxBytesReader (size)          │
│    - Timeout                        │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│ 2. Parsing (casesensitive/json)     │
│    ✓ JSON structure                 │
│    ✓ Case-sensitive                 │
│    ✓ Trailing data                  │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│ 3. Validation (your application)    │
│    - github.com/go-playground/      │
│      validator                      │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│ 4. Sanitization (your application)  │
│    - Prepared statements (SQL)      │
│    - html/template (HTML)           │
└─────────────────────────────────────┘
```

## Tests

```bash
# All tests
go test -v ./json/

# Coverage
go test -cover ./json/
```

## When to Use

### Use casesensitive/json when

- ✅ Public APIs (strict validation)
- ✅ Sensitive data
- ✅ Compliance/audit requirements
- ✅ Preventing bypass via case manipulation

### Use encoding/json when

- ⚠️ Performance is critical
- ⚠️ Data from trusted sources
- ⚠️ Case-sensitivity doesn't matter

## License

MIT

## Contributing

Pull requests are welcome. For major changes, please open an issue first.
