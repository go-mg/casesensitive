# casesensitive

Case-sensitive JSON unmarshaler for Go.

Unlike `encoding/json`, which performs case-insensitive field matching, this package requires exact field name matches (respecting json tags) at all nesting levels.

## Installation

```bash
go get github.com/go-mg/casesensitive/json
```

## Quick Example

```go
import csjson "github.com/go-mg/casesensitive/json"

type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

payload := `{"name":"john", "NAME":"hacker"}`

var user User
csjson.Unmarshal([]byte(payload), &user)
// user.Name = "john" (only exact "name" match)
```

With `encoding/json`, `user.Name` would be `"hacker"` (last value wins, case-insensitive).

## Packages

- [`json`](json/) — Case-sensitive JSON unmarshaling with trailing data protection

## License

MIT
