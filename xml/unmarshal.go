package xml

import (
	"encoding/xml"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// Unmarshal parses the XML-encoded data and stores the result in the value pointed to by v.
// Unlike standard xml.Unmarshal, this function is case-sensitive at all nesting levels
// and will only match exact element names (respecting xml tags).
// v must be a pointer to a struct.
func Unmarshal(data []byte, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("v must be a non-nil pointer")
	}

	if rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("v must be a pointer to a struct")
	}

	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	return decodeStruct(decoder, rv, "")
}

// decodeStruct reads XML tokens and populates the struct pointed to by structPtr.
// wrapperName is the expected wrapper element name (empty to skip/accept any root element).
func decodeStruct(decoder *xml.Decoder, structPtr reflect.Value, wrapperName string) error {
	rv := structPtr
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	rt := rv.Type()
	fieldMap := cachedBuildFieldMap(rt)
	attrMap := cachedBuildAttrMap(rt)

	// Find the start element (root or wrapper)
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to parse XML: %w", err)
		}

		if se, ok := tok.(xml.StartElement); ok {
			// If wrapperName is specified, validate it
			if wrapperName != "" && se.Name.Local != wrapperName {
				return fmt.Errorf("xml: expected element %q, got %q", wrapperName, se.Name.Local)
			}

			// Process attributes case-sensitively
			for _, attr := range se.Attr {
				fi, exists := attrMap[attr.Name.Local]
				if !exists {
					continue
				}
				field := initAndGetField(rv, fi.index)
				if !field.CanSet() {
					continue
				}
				if err := setFieldFromString(field, attr.Value); err != nil {
					return fmt.Errorf("failed to set attribute %s: %w", fi.name, err)
				}
			}

			// Process child elements
			if err := decodeChildren(decoder, rv, fieldMap); err != nil {
				return err
			}
			return nil
		}
	}
}

// decodeChildren reads child elements and maps them to struct fields case-sensitively.
func decodeChildren(decoder *xml.Decoder, rv reflect.Value, fieldMap map[string]fieldInfo) error {
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to parse XML: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			fi, exists := fieldMap[t.Name.Local]
			if !exists {
				// Skip unknown element and its children
				if err := decoder.Skip(); err != nil {
					return err
				}
				continue
			}

			field := initAndGetField(rv, fi.index)
			if !field.CanSet() {
				if err := decoder.Skip(); err != nil {
					return err
				}
				continue
			}

			if err := decodeField(decoder, field, t); err != nil {
				return fmt.Errorf("failed to unmarshal field %s: %w", fi.name, err)
			}

		case xml.EndElement:
			// End of parent element
			return nil
		}
	}
}

// decodeField decodes a single XML element into a struct field.
func decodeField(decoder *xml.Decoder, field reflect.Value, start xml.StartElement) error {
	fieldType := field.Type()

	// Dereference pointer
	actualType := fieldType
	if actualType.Kind() == reflect.Ptr {
		actualType = actualType.Elem()
	}

	switch actualType.Kind() {
	case reflect.Struct:
		return decodeNestedStruct(decoder, field, fieldType, start)

	case reflect.Slice:
		return decodeSliceElement(decoder, field, fieldType, start)

	default:
		// Primitive type: read char data
		content, err := readElementContent(decoder)
		if err != nil {
			return err
		}
		return setFieldFromString(field, content)
	}
}

// decodeNestedStruct handles nested struct elements recursively.
func decodeNestedStruct(decoder *xml.Decoder, field reflect.Value, fieldType reflect.Type, start xml.StartElement) error {
	if fieldType.Kind() == reflect.Ptr {
		ptr := reflect.New(fieldType.Elem())
		ptrRv := ptr.Elem()
		fm := cachedBuildFieldMap(fieldType.Elem())
		am := cachedBuildAttrMap(fieldType.Elem())

		// Process attributes
		for _, attr := range start.Attr {
			fi, exists := am[attr.Name.Local]
			if !exists {
				continue
			}
			f := initAndGetField(ptrRv, fi.index)
			if f.CanSet() {
				if err := setFieldFromString(f, attr.Value); err != nil {
					return err
				}
			}
		}

		if err := decodeChildren(decoder, ptrRv, fm); err != nil {
			return err
		}
		field.Set(ptr)
	} else {
		fm := cachedBuildFieldMap(fieldType)
		am := cachedBuildAttrMap(fieldType)

		// Process attributes
		for _, attr := range start.Attr {
			fi, exists := am[attr.Name.Local]
			if !exists {
				continue
			}
			f := initAndGetField(field, fi.index)
			if f.CanSet() {
				if err := setFieldFromString(f, attr.Value); err != nil {
					return err
				}
			}
		}

		if err := decodeChildren(decoder, field, fm); err != nil {
			return err
		}
	}
	return nil
}

// decodeSliceElement appends a new element to a slice field.
func decodeSliceElement(decoder *xml.Decoder, field reflect.Value, fieldType reflect.Type, start xml.StartElement) error {
	sliceType := fieldType
	if sliceType.Kind() == reflect.Ptr {
		sliceType = sliceType.Elem()
	}
	elemType := sliceType.Elem()

	if derefKind(elemType) == reflect.Struct {
		// Slice of structs
		actualElemType := elemType
		if actualElemType.Kind() == reflect.Ptr {
			actualElemType = actualElemType.Elem()
		}

		itemPtr := reflect.New(actualElemType)
		itemRv := itemPtr.Elem()
		fm := cachedBuildFieldMap(actualElemType)
		am := cachedBuildAttrMap(actualElemType)

		for _, attr := range start.Attr {
			fi, exists := am[attr.Name.Local]
			if !exists {
				continue
			}
			f := initAndGetField(itemRv, fi.index)
			if f.CanSet() {
				if err := setFieldFromString(f, attr.Value); err != nil {
					return err
				}
			}
		}

		if err := decodeChildren(decoder, itemRv, fm); err != nil {
			return err
		}

		var itemVal reflect.Value
		if elemType.Kind() == reflect.Ptr {
			itemVal = itemPtr
		} else {
			itemVal = itemPtr.Elem()
		}
		field.Set(reflect.Append(field, itemVal))
	} else {
		// Slice of primitives
		content, err := readElementContent(decoder)
		if err != nil {
			return err
		}
		elemPtr := reflect.New(elemType)
		if err := setFieldFromString(elemPtr.Elem(), content); err != nil {
			return err
		}
		field.Set(reflect.Append(field, elemPtr.Elem()))
	}
	return nil
}

// readElementContent reads the text content of an XML element until its end tag.
func readElementContent(decoder *xml.Decoder) (string, error) {
	var content strings.Builder
	for {
		tok, err := decoder.Token()
		if err != nil {
			return "", err
		}
		switch t := tok.(type) {
		case xml.CharData:
			content.Write(t)
		case xml.EndElement:
			return content.String(), nil
		case xml.StartElement:
			// Nested element inside a primitive field — skip it
			if err := decoder.Skip(); err != nil {
				return "", err
			}
		}
	}
}

// setFieldFromString sets a reflect.Value from a string, handling common types.
func setFieldFromString(field reflect.Value, s string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		field.SetFloat(n)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		field.SetBool(b)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Type())
	}
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
// any nil pointer-to-struct fields along the way.
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

// fieldMapCache caches the result of buildFieldMap per reflect.Type.
var fieldMapCache sync.Map

// cachedBuildFieldMap returns a cached field map for the given type.
func cachedBuildFieldMap(t reflect.Type) map[string]fieldInfo {
	if cached, ok := fieldMapCache.Load(t); ok {
		return cached.(map[string]fieldInfo)
	}
	fm := buildFieldMap(t)
	fieldMapCache.Store(t, fm)
	return fm
}

// attrMapCache caches the result of buildAttrMap per reflect.Type.
var attrMapCache sync.Map

// cachedBuildAttrMap returns a cached attr map for the given type.
func cachedBuildAttrMap(t reflect.Type) map[string]fieldInfo {
	if cached, ok := attrMapCache.Load(t); ok {
		return cached.(map[string]fieldInfo)
	}
	am := buildAttrMap(t)
	attrMapCache.Store(t, am)
	return am
}

// buildFieldMap creates a map of XML element names to struct field information.
// Fields with ",attr" in their xml tag are excluded (handled by buildAttrMap).
func buildFieldMap(t reflect.Type) map[string]fieldInfo {
	fieldMap := make(map[string]fieldInfo)
	buildFieldMapRecursive(t, nil, fieldMap)
	return fieldMap
}

// buildFieldMapRecursive walks struct fields including exported embedded structs.
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

		xmlName, opts := getXMLFieldName(field)
		if xmlName == "-" {
			continue
		}
		// Skip attribute fields
		if opts.contains("attr") {
			continue
		}

		index := makeIndex(parentIndex, i)

		if _, exists := fieldMap[xmlName]; !exists {
			fieldMap[xmlName] = fieldInfo{
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

		index := makeIndex(parentIndex, i)
		buildFieldMapRecursive(ft, index, fieldMap)
	}
}

// buildAttrMap creates a map of XML attribute names to struct field information.
// Only fields with ",attr" in their xml tag are included.
func buildAttrMap(t reflect.Type) map[string]fieldInfo {
	attrMap := make(map[string]fieldInfo)
	buildAttrMapRecursive(t, nil, attrMap)
	return attrMap
}

// buildAttrMapRecursive walks struct fields for attribute mappings.
func buildAttrMapRecursive(t reflect.Type, parentIndex []int, attrMap map[string]fieldInfo) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if field.Anonymous && field.IsExported() {
			ft := field.Type
			if ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				index := makeIndex(parentIndex, i)
				buildAttrMapRecursive(ft, index, attrMap)
				continue
			}
		}

		if !field.IsExported() {
			continue
		}

		xmlName, opts := getXMLFieldName(field)
		if xmlName == "-" {
			continue
		}
		if !opts.contains("attr") {
			continue
		}

		index := makeIndex(parentIndex, i)

		if _, exists := attrMap[xmlName]; !exists {
			attrMap[xmlName] = fieldInfo{
				name:  field.Name,
				index: index,
			}
		}
	}
}

// tagOptions is a string following the name in a struct tag.
type tagOptions string

// contains checks if a comma-separated option is present.
func (o tagOptions) contains(opt string) bool {
	for o != "" {
		var name string
		if idx := strings.Index(string(o), ","); idx >= 0 {
			name, o = string(o[:idx]), o[idx+1:]
		} else {
			name, o = string(o), ""
		}
		if name == opt {
			return true
		}
	}
	return false
}

// getXMLFieldName extracts the XML field name and options from a struct field.
func getXMLFieldName(field reflect.StructField) (string, tagOptions) {
	tag := field.Tag.Get("xml")
	if tag == "" {
		return field.Name, ""
	}

	if idx := strings.Index(tag, ","); idx >= 0 {
		name := strings.TrimSpace(tag[:idx])
		if name == "" {
			return field.Name, tagOptions(tag[idx+1:])
		}
		return name, tagOptions(tag[idx+1:])
	}

	name := strings.TrimSpace(tag)
	if name == "" {
		return field.Name, ""
	}
	return name, ""
}

// makeIndex creates a new index slice appending i to parentIndex.
func makeIndex(parentIndex []int, i int) []int {
	index := make([]int, len(parentIndex)+1)
	copy(index, parentIndex)
	index[len(parentIndex)] = i
	return index
}
