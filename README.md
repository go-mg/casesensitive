# casesensitive

[![CI](https://github.com/go-mg/casesensitive/actions/workflows/ci.yml/badge.svg)](https://github.com/go-mg/casesensitive/actions/workflows/ci.yml)

Case-sensitive JSON and XML unmarshaler for Go.

Unlike `encoding/json` and `encoding/xml`, which perform case-insensitive field matching, these packages require exact field name matches (respecting json/xml tags) at all nesting levels.

## Installation

```bash
go get github.com/go-mg/casesensitive/json
go get github.com/go-mg/casesensitive/xml
```

## Quick Example

```go
import (
    csjson "github.com/go-mg/casesensitive/json"
    csxml  "github.com/go-mg/casesensitive/xml"
)

type User struct {
    Name  string `json:"name" xml:"name"`
    Email string `json:"email" xml:"email"`
}

// JSON
jsonPayload := `{"name":"john", "NAME":"hacker"}`
var user1 User
csjson.Unmarshal([]byte(jsonPayload), &user1)
// user1.Name = "john" (only exact "name" match)

// XML
xmlPayload := `<User><name>john</name><NAME>hacker</NAME></User>`
var user2 User
csxml.Unmarshal([]byte(xmlPayload), &user2)
// user2.Name = "john" (only exact "name" match)
```

With `encoding/json` and `encoding/xml`, `Name` would be `"hacker"` (last value wins, case-insensitive).

## Packages

- [`json`](json/) — Case-sensitive JSON unmarshaling with trailing data protection
- [`xml`](xml/) — Case-sensitive XML unmarshaling with attribute support

## License

MIT
