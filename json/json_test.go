package json_test

import (
	"io"
	"strings"
	"testing"

	"github.com/go-mg/casesensitive/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

// TestCaseSensitiveMatching tests case-sensitive field matching
func TestCaseSensitiveMatching(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  string
		expected User
	}{
		{
			name:     "exact match",
			payload:  `{"name":"john","email":"test@example.com","age":30}`,
			expected: User{Name: "john", Email: "test@example.com", Age: 30},
		},
		{
			name:     "duplicate fields - lowercase wins",
			payload:  `{"name":"john","NAME":"hacker","email":"test@example.com","age":30}`,
			expected: User{Name: "john", Email: "test@example.com", Age: 30},
		},
		{
			name:     "only uppercase - ignored",
			payload:  `{"NAME":"hacker","email":"test@example.com","age":30}`,
			expected: User{Name: "", Email: "test@example.com", Age: 30},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var user User
			err := json.Unmarshal([]byte(tt.payload), &user)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, user)
		})
	}
}

// TestStructWithoutTags tests structs without JSON tags
func TestStructWithoutTags(t *testing.T) {
	t.Parallel()

	type Person struct {
		Name  string
		Email string
	}

	tests := []struct {
		name     string
		payload  string
		expected Person
	}{
		{
			name:     "correct case",
			payload:  `{"Name":"John","Email":"john@example.com"}`,
			expected: Person{Name: "John", Email: "john@example.com"},
		},
		{
			name:     "wrong case - ignored",
			payload:  `{"name":"John","email":"john@example.com"}`,
			expected: Person{Name: "", Email: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var person Person
			err := json.Unmarshal([]byte(tt.payload), &person)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, person)
		})
	}
}

// TestTrailingDataProtection tests protection against trailing data injection
func TestTrailingDataProtection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		payload           string
		allowTrailingData bool
		expectError       bool
	}{
		{
			name:        "valid JSON",
			payload:     `{"name":"john","email":"test@example.com"}`,
			expectError: false,
		},
		{
			name:        "trailing data - rejected by default",
			payload:     `{"name":"john","email":"test@example.com"}extra`,
			expectError: true,
		},
		{
			name:        "leading data - always rejected",
			payload:     `extra{"name":"john","email":"test@example.com"}`,
			expectError: true,
		},
		{
			name:              "trailing data - allowed when configured",
			payload:           `{"name":"john","email":"test@example.com"}extra`,
			allowTrailingData: true,
			expectError:       false,
		},
		{
			name:              "multiple JSONs",
			payload:           `{"name":"john","email":"test1@example.com"}{"name":"jane","email":"test2@example.com"}`,
			allowTrailingData: true,
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var user User
			decoder := json.NewDecoder(strings.NewReader(tt.payload))

			if tt.allowTrailingData {
				decoder.AllowTrailingData()
			}

			err := decoder.Decode(&user)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestDisallowUnknownFields tests strict field validation
func TestDisallowUnknownFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		payload     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid fields",
			payload:     `{"name":"john","email":"test@example.com","age":30}`,
			expectError: false,
		},
		{
			name:        "unknown field - uppercase",
			payload:     `{"name":"john","NAME":"hacker","email":"test@example.com","age":30}`,
			expectError: true,
			errorMsg:    `unknown field "NAME"`,
		},
		{
			name:        "completely unknown field",
			payload:     `{"name":"john","email":"test@example.com","age":30,"unknown":"field"}`,
			expectError: true,
			errorMsg:    `unknown field "unknown"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var user User
			decoder := json.NewDecoder(strings.NewReader(tt.payload))
			decoder.DisallowUnknownFields()
			err := decoder.Decode(&user)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestSpecialCharactersInValues tests that special characters are preserved
// Note: json does NOT sanitize values - that's the application's responsibility
func TestSpecialCharactersInValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  string
		expected string
	}{
		{
			name:     "SQL injection attempt",
			payload:  `{"name":"'; DROP TABLE users; --","email":"test@example.com"}`,
			expected: "'; DROP TABLE users; --",
		},
		{
			name:     "XSS attempt",
			payload:  `{"name":"<script>alert('xss')</script>","email":"test@example.com"}`,
			expected: "<script>alert('xss')</script>",
		},
		{
			name:     "Unicode characters",
			payload:  `{"name":"José 日本語 🎉","email":"test@example.com"}`,
			expected: "José 日本語 🎉",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var user User
			err := json.Unmarshal([]byte(tt.payload), &user)
			require.NoError(t, err, "json should parse valid JSON regardless of content")
			assert.Equal(t, tt.expected, user.Name)
		})
	}
}

// TestInvalidInput tests error handling for invalid input
func TestInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload string
		target  interface{}
	}{
		{
			name:    "invalid JSON",
			payload: `{"name":"john"`,
			target:  &User{},
		},
		{
			name:    "not a pointer",
			payload: `{"name":"john"}`,
			target:  User{},
		},
		{
			name:    "nil pointer",
			payload: `{"name":"john"}`,
			target:  (*User)(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := json.Unmarshal([]byte(tt.payload), tt.target)
			require.Error(t, err)
		})
	}
}

// TestComplexStructures tests nested objects and arrays
func TestComplexStructures(t *testing.T) {
	t.Parallel()

	type Address struct {
		Street string `json:"street"`
		City   string `json:"city"`
	}

	type Person struct {
		Name    string   `json:"name"`
		Address Address  `json:"address"`
		Tags    []string `json:"tags"`
	}

	payload := `{
		"name":"John",
		"address":{"street":"Main St","city":"NYC"},
		"tags":["tag1","tag2"]
	}`

	var person Person
	err := json.Unmarshal([]byte(payload), &person)
	require.NoError(t, err)
	assert.Equal(t, "John", person.Name)
	assert.Equal(t, "Main St", person.Address.Street)
	assert.Equal(t, []string{"tag1", "tag2"}, person.Tags)
}

// TestEmptyAndNullValues tests handling of empty and null values
func TestEmptyAndNullValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  string
		expected User
	}{
		{
			name:     "empty strings",
			payload:  `{"name":"","email":""}`,
			expected: User{Name: "", Email: ""},
		},
		{
			name:     "null values",
			payload:  `{"name":null,"email":null}`,
			expected: User{Name: "", Email: ""},
		},
		{
			name:     "missing fields",
			payload:  `{}`,
			expected: User{Name: "", Email: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var user User
			err := json.Unmarshal([]byte(tt.payload), &user)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, user)
		})
	}
}

// TestIgnoredFields tests that fields with json:"-" are ignored
func TestIgnoredFields(t *testing.T) {
	t.Parallel()

	type UserWithIgnored struct {
		Name     string `json:"name"`
		Password string `json:"-"`
		Email    string `json:"email"`
	}

	payload := `{"name":"John","password":"secret123","email":"john@example.com"}`

	var user UserWithIgnored
	err := json.Unmarshal([]byte(payload), &user)
	require.NoError(t, err)
	assert.Equal(t, "John", user.Name)
	assert.Equal(t, "", user.Password) // Should be ignored
	assert.Equal(t, "john@example.com", user.Email)
}

// TestEmbeddedStructs tests embedded (anonymous) struct field handling
func TestEmbeddedStructs(t *testing.T) {
	t.Parallel()

	type Base struct {
		ID    int    `json:"id"`
		Label string `json:"label"`
	}

	type Extended struct {
		Base
		Name string `json:"name"`
	}

	t.Run("embedded fields are flattened", func(t *testing.T) {
		var e Extended
		err := json.Unmarshal([]byte(`{"id":1,"label":"test","name":"john"}`), &e)
		require.NoError(t, err)
		assert.Equal(t, 1, e.ID)
		assert.Equal(t, "test", e.Label)
		assert.Equal(t, "john", e.Name)
	})

	t.Run("embedded fields are case-sensitive", func(t *testing.T) {
		var e Extended
		err := json.Unmarshal([]byte(`{"ID":1,"Label":"test","name":"john"}`), &e)
		require.NoError(t, err)
		assert.Equal(t, 0, e.ID)     // "ID" != "id"
		assert.Equal(t, "", e.Label) // "Label" != "label"
		assert.Equal(t, "john", e.Name)
	})
}

// TestEmbeddedStructFieldConflict tests that outer fields take precedence over embedded
func TestEmbeddedStructFieldConflict(t *testing.T) {
	t.Parallel()

	type Base struct {
		Name string `json:"name"`
	}

	type Override struct {
		Base
		Name string `json:"name"`
	}

	var o Override
	err := json.Unmarshal([]byte(`{"name":"outer"}`), &o)
	require.NoError(t, err)
	assert.Equal(t, "outer", o.Name)
	assert.Equal(t, "", o.Base.Name) // embedded field should not be set
}

// TestEmbeddedPointerStruct tests embedded struct via pointer
func TestEmbeddedPointerStruct(t *testing.T) {
	t.Parallel()

	type Base struct {
		ID int `json:"id"`
	}

	type WithPtr struct {
		*Base
		Name string `json:"name"`
	}

	t.Run("pointer embedded is initialized and populated", func(t *testing.T) {
		var w WithPtr
		err := json.Unmarshal([]byte(`{"id":42,"name":"jane"}`), &w)
		require.NoError(t, err)
		assert.Equal(t, "jane", w.Name)
		require.NotNil(t, w.Base)
		assert.Equal(t, 42, w.Base.ID)
	})

	t.Run("pointer embedded stays nil when field not present", func(t *testing.T) {
		var w WithPtr
		err := json.Unmarshal([]byte(`{"name":"jane"}`), &w)
		require.NoError(t, err)
		assert.Equal(t, "jane", w.Name)
		assert.Nil(t, w.Base)
	})
}

// TestUnexportedEmbeddedIgnored tests that unexported embedded structs are ignored
func TestUnexportedEmbeddedIgnored(t *testing.T) {
	t.Parallel()

	// unexported embedded struct defined at package level is not possible in _test
	// but we can test with exported embedded that has unexported fields
	type Base struct {
		Name   string `json:"name"`
		secret string //nolint:unused
	}

	type Outer struct {
		Base
		Email string `json:"email"`
	}

	var o Outer
	err := json.Unmarshal([]byte(`{"name":"john","email":"j@example.com"}`), &o)
	require.NoError(t, err)
	assert.Equal(t, "john", o.Name)
	assert.Equal(t, "j@example.com", o.Email)
}

// TestUnmarshalArray tests JSON array unmarshaling via Unmarshal
func TestUnmarshalArray(t *testing.T) {
	t.Parallel()

	t.Run("array of structs", func(t *testing.T) {
		var users []User
		payload := `[{"name":"john","email":"j@example.com","age":30},{"name":"jane","email":"ja@example.com","age":25}]`
		err := json.Unmarshal([]byte(payload), &users)
		require.NoError(t, err)
		require.Len(t, users, 2)
		assert.Equal(t, "john", users[0].Name)
		assert.Equal(t, "jane", users[1].Name)
	})

	t.Run("array is case-sensitive", func(t *testing.T) {
		var users []User
		payload := `[{"Name":"john","email":"j@example.com","age":30}]`
		err := json.Unmarshal([]byte(payload), &users)
		require.NoError(t, err)
		require.Len(t, users, 1)
		assert.Equal(t, "", users[0].Name) // "Name" != "name"
		assert.Equal(t, "j@example.com", users[0].Email)
	})

	t.Run("empty array", func(t *testing.T) {
		var users []User
		err := json.Unmarshal([]byte(`[]`), &users)
		require.NoError(t, err)
		assert.Empty(t, users)
	})

	t.Run("array of non-struct falls back to standard", func(t *testing.T) {
		var nums []int
		err := json.Unmarshal([]byte(`[1,2,3]`), &nums)
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, nums)
	})
}

// TestDecoderArray tests JSON array decoding via Decoder
func TestDecoderArray(t *testing.T) {
	t.Parallel()

	t.Run("decode array of structs", func(t *testing.T) {
		var users []User
		decoder := json.NewDecoder(strings.NewReader(`[{"name":"john","email":"j@example.com","age":30}]`))
		err := decoder.Decode(&users)
		require.NoError(t, err)
		require.Len(t, users, 1)
		assert.Equal(t, "john", users[0].Name)
	})

	t.Run("decode array with DisallowUnknownFields", func(t *testing.T) {
		var users []User
		decoder := json.NewDecoder(strings.NewReader(`[{"name":"john","email":"j@example.com","age":30,"extra":"bad"}]`))
		decoder.DisallowUnknownFields()
		err := decoder.Decode(&users)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown field "extra"`)
	})

	t.Run("decode array rejects trailing data", func(t *testing.T) {
		var users []User
		decoder := json.NewDecoder(strings.NewReader(`[{"name":"john","email":"j@example.com","age":30}]extra`))
		err := decoder.Decode(&users)
		require.Error(t, err)
	})

	t.Run("decode empty array", func(t *testing.T) {
		var users []User
		decoder := json.NewDecoder(strings.NewReader(`[]`))
		err := decoder.Decode(&users)
		require.NoError(t, err)
		assert.Empty(t, users)
	})
}

// TestNestedCaseSensitive tests that nested structs are case-sensitive at all levels
func TestNestedCaseSensitive(t *testing.T) {
	t.Parallel()

	type Address struct {
		Street string `json:"street"`
		City   string `json:"city"`
	}

	type Person struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	t.Run("wrong case on nested field is ignored", func(t *testing.T) {
		var p Person
		err := json.Unmarshal([]byte(`{"name":"john","address":{"Street":"Main St","city":"NYC"}}`), &p)
		require.NoError(t, err)
		assert.Equal(t, "john", p.Name)
		assert.Equal(t, "", p.Address.Street) // "Street" != "street"
		assert.Equal(t, "NYC", p.Address.City)
	})

	t.Run("correct case on nested field works", func(t *testing.T) {
		var p Person
		err := json.Unmarshal([]byte(`{"name":"john","address":{"street":"Main St","city":"NYC"}}`), &p)
		require.NoError(t, err)
		assert.Equal(t, "Main St", p.Address.Street)
	})

	t.Run("deeply nested case-sensitive", func(t *testing.T) {
		type Country struct {
			Code string `json:"code"`
		}
		type Addr struct {
			Country Country `json:"country"`
		}
		type P struct {
			Addr Addr `json:"addr"`
		}

		var p P
		err := json.Unmarshal([]byte(`{"addr":{"country":{"Code":"BR"}}}`), &p)
		require.NoError(t, err)
		assert.Equal(t, "", p.Addr.Country.Code) // "Code" != "code"

		err = json.Unmarshal([]byte(`{"addr":{"country":{"code":"BR"}}}`), &p)
		require.NoError(t, err)
		assert.Equal(t, "BR", p.Addr.Country.Code)
	})
}

// TestNestedSliceOfStructsCaseSensitive tests that slices of structs inside a struct are case-sensitive
func TestNestedSliceOfStructsCaseSensitive(t *testing.T) {
	t.Parallel()

	type Item struct {
		Name string `json:"name"`
	}

	type Order struct {
		ID    int    `json:"id"`
		Items []Item `json:"items"`
	}

	t.Run("nested slice elements are case-sensitive", func(t *testing.T) {
		var o Order
		err := json.Unmarshal([]byte(`{"id":1,"items":[{"Name":"wrong"},{"name":"right"}]}`), &o)
		require.NoError(t, err)
		require.Len(t, o.Items, 2)
		assert.Equal(t, "", o.Items[0].Name) // "Name" != "name"
		assert.Equal(t, "right", o.Items[1].Name)
	})
}

// TestOmitemptyTag tests that json:",omitempty" tag with empty name uses field name
func TestOmitemptyTag(t *testing.T) {
	t.Parallel()

	type Item struct {
		Value string `json:",omitempty"`
	}

	var item Item
	err := json.Unmarshal([]byte(`{"Value":"test"}`), &item)
	require.NoError(t, err)
	assert.Equal(t, "test", item.Value)

	// lowercase should not match since no explicit name and field is "Value"
	var item2 Item
	err = json.Unmarshal([]byte(`{"value":"test"}`), &item2)
	require.NoError(t, err)
	assert.Equal(t, "", item2.Value)
}

// TestAllFieldsIgnored tests struct where all fields have json:"-"
func TestAllFieldsIgnored(t *testing.T) {
	t.Parallel()

	type Secret struct {
		A string `json:"-"`
		B string `json:"-"`
	}

	var s Secret
	err := json.Unmarshal([]byte(`{"A":"x","B":"y"}`), &s)
	require.NoError(t, err)
	assert.Equal(t, "", s.A)
	assert.Equal(t, "", s.B)
}

// TestDecoderStreamWithAllowTrailingData tests reading multiple JSON objects from a stream
func TestDecoderStreamWithAllowTrailingData(t *testing.T) {
	t.Parallel()

	payload := `{"name":"john","email":"j@example.com","age":30}{"name":"jane","email":"ja@example.com","age":25}`
	decoder := json.NewDecoder(strings.NewReader(payload))
	decoder.AllowTrailingData()

	var user1 User
	err := decoder.Decode(&user1)
	require.NoError(t, err)
	assert.Equal(t, "john", user1.Name)

	// Second decode should read the next JSON object
	// Need a new decoder since the internal state consumed the first object
	// Actually the underlying json.Decoder should handle this
	var user2 User
	err = decoder.Decode(&user2)
	require.NoError(t, err)
	assert.Equal(t, "jane", user2.Name)

	// Third decode should return EOF
	var user3 User
	err = decoder.Decode(&user3)
	require.ErrorIs(t, err, io.EOF)
}

// TestDecoderChainedMethods tests that Decoder methods return *Decoder for chaining
func TestDecoderChainedMethods(t *testing.T) {
	t.Parallel()

	var user User
	decoder := json.NewDecoder(strings.NewReader(`{"name":"john","email":"j@example.com","age":30}`)).
		DisallowUnknownFields().
		AllowTrailingData()

	err := decoder.Decode(&user)
	require.NoError(t, err)
	assert.Equal(t, "john", user.Name)
}
