package xml_test

import (
	"strings"
	"testing"

	"github.com/go-mg/casesensitive/xml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type User struct {
	Name  string `xml:"name"`
	Email string `xml:"email"`
	Age   int    `xml:"age"`
}

// TestCaseSensitiveMatching tests case-sensitive element matching
func TestCaseSensitiveMatching(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  string
		expected User
	}{
		{
			name:     "exact match",
			payload:  `<User><name>john</name><email>test@example.com</email><age>30</age></User>`,
			expected: User{Name: "john", Email: "test@example.com", Age: 30},
		},
		{
			name:     "duplicate elements - exact match wins",
			payload:  `<User><name>john</name><NAME>hacker</NAME><email>test@example.com</email><age>30</age></User>`,
			expected: User{Name: "john", Email: "test@example.com", Age: 30},
		},
		{
			name:     "only uppercase - ignored",
			payload:  `<User><NAME>hacker</NAME><email>test@example.com</email><age>30</age></User>`,
			expected: User{Name: "", Email: "test@example.com", Age: 30},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var user User
			err := xml.Unmarshal([]byte(tt.payload), &user)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, user)
		})
	}
}

// TestStructWithoutTags tests structs without XML tags
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
			payload:  `<Person><Name>John</Name><Email>john@example.com</Email></Person>`,
			expected: Person{Name: "John", Email: "john@example.com"},
		},
		{
			name:     "wrong case - ignored",
			payload:  `<Person><name>John</name><email>john@example.com</email></Person>`,
			expected: Person{Name: "", Email: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var person Person
			err := xml.Unmarshal([]byte(tt.payload), &person)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, person)
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
			payload:     `<User><name>john</name><email>test@example.com</email><age>30</age></User>`,
			expectError: false,
		},
		{
			name:        "unknown field - uppercase",
			payload:     `<User><name>john</name><NAME>hacker</NAME><email>test@example.com</email><age>30</age></User>`,
			expectError: true,
			errorMsg:    `unknown field "NAME"`,
		},
		{
			name:        "completely unknown field",
			payload:     `<User><name>john</name><email>test@example.com</email><age>30</age><unknown>field</unknown></User>`,
			expectError: true,
			errorMsg:    `unknown field "unknown"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var user User
			decoder := xml.NewDecoder(strings.NewReader(tt.payload))
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
func TestSpecialCharactersInValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  string
		expected string
	}{
		{
			name:     "SQL injection attempt",
			payload:  `<User><name>'; DROP TABLE users; --</name><email>test@example.com</email></User>`,
			expected: "'; DROP TABLE users; --",
		},
		{
			name:     "XML entities",
			payload:  `<User><name>&lt;script&gt;alert(&apos;xss&apos;)&lt;/script&gt;</name><email>test@example.com</email></User>`,
			expected: "<script>alert('xss')</script>",
		},
		{
			name:     "Unicode characters",
			payload:  `<User><name>José 日本語</name><email>test@example.com</email></User>`,
			expected: "José 日本語",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var user User
			err := xml.Unmarshal([]byte(tt.payload), &user)
			require.NoError(t, err)
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
		target  any
	}{
		{
			name:    "invalid XML",
			payload: `<User><name>john`,
			target:  &User{},
		},
		{
			name:    "not a pointer",
			payload: `<User><name>john</name></User>`,
			target:  User{},
		},
		{
			name:    "nil pointer",
			payload: `<User><name>john</name></User>`,
			target:  (*User)(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := xml.Unmarshal([]byte(tt.payload), tt.target)
			require.Error(t, err)
		})
	}
}

// TestComplexStructures tests nested objects
func TestComplexStructures(t *testing.T) {
	t.Parallel()

	type Address struct {
		Street string `xml:"street"`
		City   string `xml:"city"`
	}

	type Person struct {
		Name    string   `xml:"name"`
		Address Address  `xml:"address"`
		Tags    []string `xml:"tag"`
	}

	payload := `<Person><name>John</name><address><street>Main St</street><city>NYC</city></address><tag>tag1</tag><tag>tag2</tag></Person>`

	var person Person
	err := xml.Unmarshal([]byte(payload), &person)
	require.NoError(t, err)
	assert.Equal(t, "John", person.Name)
	assert.Equal(t, "Main St", person.Address.Street)
	assert.Equal(t, []string{"tag1", "tag2"}, person.Tags)
}

// TestEmptyAndMissingValues tests handling of empty and missing values
func TestEmptyAndMissingValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  string
		expected User
	}{
		{
			name:     "empty elements",
			payload:  `<User><name></name><email></email></User>`,
			expected: User{Name: "", Email: ""},
		},
		{
			name:     "missing fields",
			payload:  `<User></User>`,
			expected: User{Name: "", Email: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var user User
			err := xml.Unmarshal([]byte(tt.payload), &user)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, user)
		})
	}
}

// TestIgnoredFields tests that fields with xml:"-" are ignored
func TestIgnoredFields(t *testing.T) {
	t.Parallel()

	type UserWithIgnored struct {
		Name     string `xml:"name"`
		Password string `xml:"-"`
		Email    string `xml:"email"`
	}

	payload := `<User><name>John</name><password>secret123</password><email>john@example.com</email></User>`

	var user UserWithIgnored
	err := xml.Unmarshal([]byte(payload), &user)
	require.NoError(t, err)
	assert.Equal(t, "John", user.Name)
	assert.Equal(t, "", user.Password)
	assert.Equal(t, "john@example.com", user.Email)
}

// TestAttributes tests XML attribute handling
func TestAttributes(t *testing.T) {
	t.Parallel()

	type Item struct {
		ID   string `xml:"id,attr"`
		Name string `xml:"name"`
	}

	t.Run("attribute case-sensitive match", func(t *testing.T) {
		var item Item
		err := xml.Unmarshal([]byte(`<Item id="42"><name>test</name></Item>`), &item)
		require.NoError(t, err)
		assert.Equal(t, "42", item.ID)
		assert.Equal(t, "test", item.Name)
	})

	t.Run("attribute wrong case - ignored", func(t *testing.T) {
		var item Item
		err := xml.Unmarshal([]byte(`<Item ID="42"><name>test</name></Item>`), &item)
		require.NoError(t, err)
		assert.Equal(t, "", item.ID) // "ID" != "id"
		assert.Equal(t, "test", item.Name)
	})
}

// TestEmbeddedStructs tests embedded struct field handling
func TestEmbeddedStructs(t *testing.T) {
	t.Parallel()

	type Base struct {
		ID    int    `xml:"id"`
		Label string `xml:"label"`
	}

	type Extended struct {
		Base
		Name string `xml:"name"`
	}

	t.Run("embedded fields are flattened", func(t *testing.T) {
		var e Extended
		err := xml.Unmarshal([]byte(`<Extended><id>1</id><label>test</label><name>john</name></Extended>`), &e)
		require.NoError(t, err)
		assert.Equal(t, 1, e.ID)
		assert.Equal(t, "test", e.Label)
		assert.Equal(t, "john", e.Name)
	})

	t.Run("embedded fields are case-sensitive", func(t *testing.T) {
		var e Extended
		err := xml.Unmarshal([]byte(`<Extended><ID>1</ID><Label>test</Label><name>john</name></Extended>`), &e)
		require.NoError(t, err)
		assert.Equal(t, 0, e.ID)
		assert.Equal(t, "", e.Label)
		assert.Equal(t, "john", e.Name)
	})
}

// TestEmbeddedPointerStruct tests embedded struct via pointer
func TestEmbeddedPointerStruct(t *testing.T) {
	t.Parallel()

	type Base struct {
		ID int `xml:"id"`
	}

	type WithPtr struct {
		*Base
		Name string `xml:"name"`
	}

	t.Run("pointer embedded is initialized and populated", func(t *testing.T) {
		var w WithPtr
		err := xml.Unmarshal([]byte(`<WithPtr><id>42</id><name>jane</name></WithPtr>`), &w)
		require.NoError(t, err)
		assert.Equal(t, "jane", w.Name)
		require.NotNil(t, w.Base)
		assert.Equal(t, 42, w.Base.ID)
	})

	t.Run("pointer embedded stays nil when field not present", func(t *testing.T) {
		var w WithPtr
		err := xml.Unmarshal([]byte(`<WithPtr><name>jane</name></WithPtr>`), &w)
		require.NoError(t, err)
		assert.Equal(t, "jane", w.Name)
		assert.Nil(t, w.Base)
	})
}

// TestNestedCaseSensitive tests case-sensitivity at all nesting levels
func TestNestedCaseSensitive(t *testing.T) {
	t.Parallel()

	type Address struct {
		Street string `xml:"street"`
		City   string `xml:"city"`
	}

	type Person struct {
		Name    string  `xml:"name"`
		Address Address `xml:"address"`
	}

	t.Run("wrong case on nested field is ignored", func(t *testing.T) {
		var p Person
		err := xml.Unmarshal([]byte(`<Person><name>john</name><address><Street>Main St</Street><city>NYC</city></address></Person>`), &p)
		require.NoError(t, err)
		assert.Equal(t, "john", p.Name)
		assert.Equal(t, "", p.Address.Street) // "Street" != "street"
		assert.Equal(t, "NYC", p.Address.City)
	})

	t.Run("correct case on nested field works", func(t *testing.T) {
		var p Person
		err := xml.Unmarshal([]byte(`<Person><name>john</name><address><street>Main St</street><city>NYC</city></address></Person>`), &p)
		require.NoError(t, err)
		assert.Equal(t, "Main St", p.Address.Street)
	})
}

// TestSliceOfStructs tests repeated elements mapped to a slice of structs
func TestSliceOfStructs(t *testing.T) {
	t.Parallel()

	type Item struct {
		Name string `xml:"name"`
	}

	type Order struct {
		ID    int    `xml:"id"`
		Items []Item `xml:"item"`
	}

	payload := `<Order><id>1</id><item><name>first</name></item><item><name>second</name></item></Order>`

	var order Order
	err := xml.Unmarshal([]byte(payload), &order)
	require.NoError(t, err)
	assert.Equal(t, 1, order.ID)
	require.Len(t, order.Items, 2)
	assert.Equal(t, "first", order.Items[0].Name)
	assert.Equal(t, "second", order.Items[1].Name)
}

// TestOmitemptyTag tests that xml:",omitempty" tag with empty name uses field name
func TestOmitemptyTag(t *testing.T) {
	t.Parallel()

	type Item struct {
		Value string `xml:",omitempty"`
	}

	var item Item
	err := xml.Unmarshal([]byte(`<Item><Value>test</Value></Item>`), &item)
	require.NoError(t, err)
	assert.Equal(t, "test", item.Value)

	var item2 Item
	err = xml.Unmarshal([]byte(`<Item><value>test</value></Item>`), &item2)
	require.NoError(t, err)
	assert.Equal(t, "", item2.Value)
}

// TestAllFieldsIgnored tests struct where all fields have xml:"-"
func TestAllFieldsIgnored(t *testing.T) {
	t.Parallel()

	type Secret struct {
		A string `xml:"-"`
		B string `xml:"-"`
	}

	var s Secret
	err := xml.Unmarshal([]byte(`<Secret><A>x</A><B>y</B></Secret>`), &s)
	require.NoError(t, err)
	assert.Equal(t, "", s.A)
	assert.Equal(t, "", s.B)
}

// TestDecoderBasic tests basic Decoder usage
func TestDecoderBasic(t *testing.T) {
	t.Parallel()

	payload := `<User><name>john</name><email>j@example.com</email><age>30</age></User>`

	var user User
	decoder := xml.NewDecoder(strings.NewReader(payload))
	err := decoder.Decode(&user)
	require.NoError(t, err)
	assert.Equal(t, "john", user.Name)
	assert.Equal(t, "j@example.com", user.Email)
	assert.Equal(t, 30, user.Age)
}

// TestDecoderChainedMethods tests that Decoder methods return *Decoder for chaining
func TestDecoderChainedMethods(t *testing.T) {
	t.Parallel()

	var user User
	decoder := xml.NewDecoder(strings.NewReader(`<User><name>john</name><email>j@example.com</email><age>30</age></User>`)).
		DisallowUnknownFields()

	err := decoder.Decode(&user)
	require.NoError(t, err)
	assert.Equal(t, "john", user.Name)
}

// TestNumericAndBoolTypes tests setFieldFromString for various types
func TestNumericAndBoolTypes(t *testing.T) {
	t.Parallel()

	type AllTypes struct {
		S   string  `xml:"s"`
		I   int     `xml:"i"`
		I8  int8    `xml:"i8"`
		I64 int64   `xml:"i64"`
		U   uint    `xml:"u"`
		U32 uint32  `xml:"u32"`
		F32 float32 `xml:"f32"`
		F64 float64 `xml:"f64"`
		B   bool    `xml:"b"`
	}

	payload := `<Root><s>hello</s><i>42</i><i8>8</i8><i64>999</i64><u>10</u><u32>20</u32><f32>3.14</f32><f64>2.718</f64><b>true</b></Root>`

	var v AllTypes
	err := xml.Unmarshal([]byte(payload), &v)
	require.NoError(t, err)
	assert.Equal(t, "hello", v.S)
	assert.Equal(t, 42, v.I)
	assert.Equal(t, int8(8), v.I8)
	assert.Equal(t, int64(999), v.I64)
	assert.Equal(t, uint(10), v.U)
	assert.Equal(t, uint32(20), v.U32)
	assert.InDelta(t, float32(3.14), v.F32, 0.01)
	assert.InDelta(t, 2.718, v.F64, 0.001)
	assert.True(t, v.B)
}

// TestNestedPointerStruct tests nested struct via pointer (not embedded)
func TestNestedPointerStruct(t *testing.T) {
	t.Parallel()

	type Address struct {
		Street string `xml:"street"`
		City   string `xml:"city"`
	}

	type Person struct {
		Name    string   `xml:"name"`
		Address *Address `xml:"address"`
	}

	t.Run("pointer nested struct is populated", func(t *testing.T) {
		var p Person
		err := xml.Unmarshal([]byte(`<Person><name>john</name><address><street>Main St</street><city>NYC</city></address></Person>`), &p)
		require.NoError(t, err)
		assert.Equal(t, "john", p.Name)
		require.NotNil(t, p.Address)
		assert.Equal(t, "Main St", p.Address.Street)
		assert.Equal(t, "NYC", p.Address.City)
	})

	t.Run("pointer nested struct stays nil when absent", func(t *testing.T) {
		var p Person
		err := xml.Unmarshal([]byte(`<Person><name>john</name></Person>`), &p)
		require.NoError(t, err)
		assert.Equal(t, "john", p.Name)
		assert.Nil(t, p.Address)
	})
}

// TestNestedStructWithAttributes tests nested struct with attributes
func TestNestedStructWithAttributes(t *testing.T) {
	t.Parallel()

	type Item struct {
		ID   string `xml:"id,attr"`
		Name string `xml:"name"`
	}

	type Container struct {
		Item Item `xml:"item"`
	}

	payload := `<Container><item id="42"><name>test</name></item></Container>`

	var c Container
	err := xml.Unmarshal([]byte(payload), &c)
	require.NoError(t, err)
	assert.Equal(t, "42", c.Item.ID)
	assert.Equal(t, "test", c.Item.Name)
}

// TestPointerNestedStructWithAttributes tests pointer-to-struct nested with attributes
func TestPointerNestedStructWithAttributes(t *testing.T) {
	t.Parallel()

	type Item struct {
		ID   string `xml:"id,attr"`
		Name string `xml:"name"`
	}

	type Container struct {
		Item *Item `xml:"item"`
	}

	payload := `<Container><item id="42"><name>test</name></item></Container>`

	var c Container
	err := xml.Unmarshal([]byte(payload), &c)
	require.NoError(t, err)
	require.NotNil(t, c.Item)
	assert.Equal(t, "42", c.Item.ID)
	assert.Equal(t, "test", c.Item.Name)
}

// TestSliceOfStructsWithAttributes tests slice elements with attributes
func TestSliceOfStructsWithAttributes(t *testing.T) {
	t.Parallel()

	type Item struct {
		ID   string `xml:"id,attr"`
		Name string `xml:"name"`
	}

	type Order struct {
		Items []Item `xml:"item"`
	}

	payload := `<Order><item id="1"><name>first</name></item><item id="2"><name>second</name></item></Order>`

	var order Order
	err := xml.Unmarshal([]byte(payload), &order)
	require.NoError(t, err)
	require.Len(t, order.Items, 2)
	assert.Equal(t, "1", order.Items[0].ID)
	assert.Equal(t, "first", order.Items[0].Name)
	assert.Equal(t, "2", order.Items[1].ID)
	assert.Equal(t, "second", order.Items[1].Name)
}

// TestDecoderInvalidInput tests Decoder error handling
func TestDecoderInvalidInput(t *testing.T) {
	t.Parallel()

	t.Run("not a pointer", func(t *testing.T) {
		decoder := xml.NewDecoder(strings.NewReader(`<User><name>john</name></User>`))
		err := decoder.Decode(User{})
		require.Error(t, err)
	})

	t.Run("nil pointer", func(t *testing.T) {
		decoder := xml.NewDecoder(strings.NewReader(`<User><name>john</name></User>`))
		err := decoder.Decode((*User)(nil))
		require.Error(t, err)
	})

	t.Run("not a struct", func(t *testing.T) {
		var s string
		decoder := xml.NewDecoder(strings.NewReader(`<User><name>john</name></User>`))
		err := decoder.Decode(&s)
		require.Error(t, err)
	})

	t.Run("invalid XML", func(t *testing.T) {
		var user User
		decoder := xml.NewDecoder(strings.NewReader(`<User><name>john`))
		err := decoder.Decode(&user)
		require.Error(t, err)
	})
}

// TestDecoderWithAttributes tests Decoder attribute handling
func TestDecoderWithAttributes(t *testing.T) {
	t.Parallel()

	type Item struct {
		ID   string `xml:"id,attr"`
		Name string `xml:"name"`
	}

	var item Item
	decoder := xml.NewDecoder(strings.NewReader(`<Item id="42"><name>test</name></Item>`))
	err := decoder.Decode(&item)
	require.NoError(t, err)
	assert.Equal(t, "42", item.ID)
	assert.Equal(t, "test", item.Name)
}

// TestDecoderDisallowUnknownAttributes tests that unknown attributes are rejected
func TestDecoderDisallowUnknownAttributes(t *testing.T) {
	t.Parallel()

	type Item struct {
		Name string `xml:"name"`
	}

	var item Item
	decoder := xml.NewDecoder(strings.NewReader(`<Item unknown="val"><name>test</name></Item>`))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&item)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown attribute "unknown"`)
}

// TestSliceOfPrimitives tests repeated primitive elements
func TestSliceOfPrimitives(t *testing.T) {
	t.Parallel()

	type Config struct {
		Values []int `xml:"value"`
	}

	payload := `<Config><value>1</value><value>2</value><value>3</value></Config>`

	var c Config
	err := xml.Unmarshal([]byte(payload), &c)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, c.Values)
}

// TestInvalidFieldValue tests error on type mismatch
func TestInvalidFieldValue(t *testing.T) {
	t.Parallel()

	type Item struct {
		Count int `xml:"count"`
	}

	var item Item
	err := xml.Unmarshal([]byte(`<Item><count>not_a_number</count></Item>`), &item)
	require.Error(t, err)
}
