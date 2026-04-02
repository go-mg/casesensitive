package json

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// Unmarshal parses the JSON-encoded data and stores the result in the value pointed to by v.
// Unlike standard json.Unmarshal, this function is case-sensitive at all nesting levels
// and will only match exact field names (respecting json tags).
// v can be a pointer to a struct or a pointer to a slice of structs.
func Unmarshal(data []byte, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("v must be a non-nil pointer")
	}

	elem := rv.Elem()

	// Handle slice of structs
	if elem.Kind() == reflect.Slice {
		return unmarshalSlice(data, rv)
	}

	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("v must be a pointer to a struct or a slice of structs")
	}

	// Unmarshal single object
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	return unmarshalFromMap(rawMap, v)
}

// unmarshalSlice handles JSON arrays by unmarshaling each element case-sensitively.
func unmarshalSlice(data []byte, slicePtr reflect.Value) error {
	var rawSlice []json.RawMessage
	if err := json.Unmarshal(data, &rawSlice); err != nil {
		return fmt.Errorf("failed to parse JSON array: %w", err)
	}

	sliceElem := slicePtr.Elem()
	elemType := sliceElem.Type().Elem()

	// For non-struct element types, fall back to standard unmarshal
	if elemType.Kind() != reflect.Struct {
		return json.Unmarshal(data, slicePtr.Interface())
	}

	result := reflect.MakeSlice(sliceElem.Type(), 0, len(rawSlice))

	for i, rawItem := range rawSlice {
		var rawMap map[string]json.RawMessage
		if err := json.Unmarshal(rawItem, &rawMap); err != nil {
			return fmt.Errorf("failed to parse JSON array element %d: %w", i, err)
		}
		itemPtr := reflect.New(elemType)
		if err := unmarshalFromMap(rawMap, itemPtr.Interface()); err != nil {
			return fmt.Errorf("failed to unmarshal array element %d: %w", i, err)
		}
		result = reflect.Append(result, itemPtr.Elem())
	}

	sliceElem.Set(result)
	return nil
}

// unmarshalFromMap unmarshals from a map into a struct with case-sensitive field matching.
// Nested struct fields are also unmarshaled case-sensitively (recursive).
func unmarshalFromMap(rawMap map[string]json.RawMessage, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("v must be a non-nil pointer")
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("v must be a pointer to a struct")
	}

	rt := rv.Type()
	fieldMap := cachedBuildFieldMap(rt)

	for jsonKey, rawValue := range rawMap {
		fi, exists := fieldMap[jsonKey]
		if !exists {
			continue
		}

		// Initialize nil pointer fields along the index path (for embedded pointer-to-struct)
		field := initAndGetField(rv, fi.index)
		if !field.CanSet() {
			continue
		}

		// Recursively handle nested structs and slices of structs case-sensitively
		fieldType := field.Type()
		if err := unmarshalFieldCaseSensitive(fieldType, field, rawValue, fi.name); err != nil {
			return err
		}
	}

	return nil
}

// unmarshalFieldCaseSensitive unmarshals a single field, recursing into nested structs
// and slices of structs to maintain case-sensitivity at all levels.
func unmarshalFieldCaseSensitive(fieldType reflect.Type, field reflect.Value, rawValue json.RawMessage, fieldName string) error {
	// Dereference pointer types
	actualType := fieldType
	if actualType.Kind() == reflect.Ptr {
		actualType = actualType.Elem()
	}

	switch actualType.Kind() {
	case reflect.Struct:
		// Nested struct: unmarshal case-sensitively
		var nestedMap map[string]json.RawMessage
		if err := json.Unmarshal(rawValue, &nestedMap); err != nil {
			// Could be null or a type that json.Unmarshal handles (e.g. time.Time)
			// Fall back to standard unmarshal
			return unmarshalStandard(field, rawValue, fieldName)
		}
		return unmarshalNestedStruct(fieldType, field, nestedMap, fieldName)

	case reflect.Slice:
		elemType := actualType.Elem()
		// Only recurse for slices of structs (or pointer-to-struct)
		if derefKind(elemType) == reflect.Struct {
			return unmarshalNestedSlice(fieldType, field, rawValue, fieldName)
		}
		return unmarshalStandard(field, rawValue, fieldName)

	default:
		return unmarshalStandard(field, rawValue, fieldName)
	}
}

// unmarshalNestedStruct handles case-sensitive unmarshal for a nested struct field.
func unmarshalNestedStruct(fieldType reflect.Type, field reflect.Value, nestedMap map[string]json.RawMessage, fieldName string) error {
	if fieldType.Kind() == reflect.Ptr {
		// Pointer to struct: allocate and set
		ptr := reflect.New(fieldType.Elem())
		if err := unmarshalFromMap(nestedMap, ptr.Interface()); err != nil {
			return fmt.Errorf("failed to unmarshal field %s: %w", fieldName, err)
		}
		field.Set(ptr)
	} else {
		// Value struct: use addressable copy
		ptr := reflect.New(fieldType)
		if err := unmarshalFromMap(nestedMap, ptr.Interface()); err != nil {
			return fmt.Errorf("failed to unmarshal field %s: %w", fieldName, err)
		}
		field.Set(ptr.Elem())
	}
	return nil
}

// unmarshalNestedSlice handles case-sensitive unmarshal for a slice of structs field.
func unmarshalNestedSlice(fieldType reflect.Type, field reflect.Value, rawValue json.RawMessage, fieldName string) error {
	var rawSlice []json.RawMessage
	if err := json.Unmarshal(rawValue, &rawSlice); err != nil {
		return fmt.Errorf("failed to unmarshal field %s: %w", fieldName, err)
	}

	sliceType := fieldType
	if sliceType.Kind() == reflect.Ptr {
		sliceType = sliceType.Elem()
	}
	elemType := sliceType.Elem()

	result := reflect.MakeSlice(sliceType, 0, len(rawSlice))

	for i, rawItem := range rawSlice {
		var itemMap map[string]json.RawMessage
		if err := json.Unmarshal(rawItem, &itemMap); err != nil {
			return fmt.Errorf("failed to unmarshal field %s[%d]: %w", fieldName, i, err)
		}

		var itemVal reflect.Value
		if elemType.Kind() == reflect.Ptr {
			itemPtr := reflect.New(elemType.Elem())
			if err := unmarshalFromMap(itemMap, itemPtr.Interface()); err != nil {
				return fmt.Errorf("failed to unmarshal field %s[%d]: %w", fieldName, i, err)
			}
			itemVal = itemPtr
		} else {
			itemPtr := reflect.New(elemType)
			if err := unmarshalFromMap(itemMap, itemPtr.Interface()); err != nil {
				return fmt.Errorf("failed to unmarshal field %s[%d]: %w", fieldName, i, err)
			}
			itemVal = itemPtr.Elem()
		}
		result = reflect.Append(result, itemVal)
	}

	if fieldType.Kind() == reflect.Ptr {
		ptr := reflect.New(sliceType)
		ptr.Elem().Set(result)
		field.Set(ptr)
	} else {
		field.Set(result)
	}
	return nil
}

// unmarshalStandard falls back to encoding/json for non-struct types.
func unmarshalStandard(field reflect.Value, rawValue json.RawMessage, fieldName string) error {
	fieldPtr := reflect.New(field.Type())
	if err := json.Unmarshal(rawValue, fieldPtr.Interface()); err != nil {
		return fmt.Errorf("failed to unmarshal field %s: %w", fieldName, err)
	}
	field.Set(fieldPtr.Elem())
	return nil
}

// derefKind returns the Kind of t, dereferencing one pointer level if needed.
func derefKind(t reflect.Type) reflect.Kind {
	if t.Kind() == reflect.Ptr {
		return t.Elem().Kind()
	}
	return t.Kind()
}

// initAndGetField traverses the index path on a reflect.Value, initializing
// any nil pointer-to-struct fields along the way (needed for embedded pointer-to-struct).
func initAndGetField(rv reflect.Value, index []int) reflect.Value {
	for _, idx := range index[:len(index)-1] {
		rv = rv.Field(idx)
		if rv.Kind() == reflect.Ptr {
			if rv.IsNil() {
				rv.Set(reflect.New(rv.Type().Elem()))
			}
			rv = rv.Elem()
		}
	}
	return rv.Field(index[len(index)-1])
}

// fieldInfo holds information about a struct field
type fieldInfo struct {
	name  string // struct field name
	index []int  // field index for reflect.Value.FieldByIndex
}

// fieldMapCache caches the result of buildFieldMap per reflect.Type to avoid
// rebuilding via reflection on every Unmarshal/Decode call.
var fieldMapCache sync.Map // map[reflect.Type]map[string]fieldInfo

// cachedBuildFieldMap returns a cached field map for the given type.
func cachedBuildFieldMap(t reflect.Type) map[string]fieldInfo {
	if cached, ok := fieldMapCache.Load(t); ok {
		return cached.(map[string]fieldInfo)
	}
	fm := buildFieldMap(t)
	fieldMapCache.Store(t, fm)
	return fm
}

// buildFieldMap creates a map of JSON field names to struct field information.
// It supports embedded (anonymous) struct fields, flattening them into the parent map.
func buildFieldMap(t reflect.Type) map[string]fieldInfo {
	fieldMap := make(map[string]fieldInfo)
	buildFieldMapRecursive(t, nil, fieldMap)
	return fieldMap
}

// buildFieldMapRecursive walks struct fields including exported embedded structs.
// Direct fields are processed before embedded fields so they take precedence on name conflicts.
func buildFieldMapRecursive(t reflect.Type, parentIndex []int, fieldMap map[string]fieldInfo) {
	// First pass: direct (non-anonymous) exported fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if field.Anonymous {
			continue
		}

		if !field.IsExported() {
			continue
		}

		index := make([]int, len(parentIndex)+1)
		copy(index, parentIndex)
		index[len(parentIndex)] = i

		jsonName := getJSONFieldName(field)
		if jsonName == "-" {
			continue
		}

		if _, exists := fieldMap[jsonName]; !exists {
			fieldMap[jsonName] = fieldInfo{
				name:  field.Name,
				index: index,
			}
		}
	}

	// Second pass: embedded (anonymous) exported structs
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if !field.Anonymous || !field.IsExported() {
			continue
		}

		ft := field.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		if ft.Kind() != reflect.Struct {
			continue
		}

		index := make([]int, len(parentIndex)+1)
		copy(index, parentIndex)
		index[len(parentIndex)] = i

		buildFieldMapRecursive(ft, index, fieldMap)
	}
}

// getJSONFieldName extracts the JSON field name from a struct field
func getJSONFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}

	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])

	if name == "" {
		return field.Name
	}

	return name
}
