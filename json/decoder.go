package json

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
)

// Decoder reads and decodes JSON values from an input stream with case-sensitive field matching.
type Decoder struct {
	decoder               *json.Decoder
	disallowUnknownFields bool
	allowTrailingData     bool
}

// NewDecoder returns a new decoder that reads from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		decoder: json.NewDecoder(r),
	}
}

// Decode reads the next JSON-encoded value from its input and stores it in the value pointed to by v.
// Unlike standard json.Decoder, this decoder is case-sensitive and validates that the entire
// input is valid JSON without trailing data (unless AllowTrailingData is called).
// v can be a pointer to a struct or a pointer to a slice of structs.
func (d *Decoder) Decode(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("v must be a non-nil pointer")
	}

	elem := rv.Elem()

	// Handle slice targets (JSON arrays)
	if elem.Kind() == reflect.Slice && elem.Type().Elem().Kind() == reflect.Struct {
		return d.decodeSlice(rv)
	}

	// Handle struct targets (JSON objects)
	var rawMap map[string]json.RawMessage
	if err := d.decoder.Decode(&rawMap); err != nil {
		return err
	}

	if err := d.checkTrailingData(); err != nil {
		return err
	}

	if d.disallowUnknownFields {
		if err := d.checkUnknownFields(rawMap, v); err != nil {
			return err
		}
	}

	return unmarshalFromMap(rawMap, v)
}

// decodeSlice handles decoding JSON arrays into slices of structs.
func (d *Decoder) decodeSlice(slicePtr reflect.Value) error {
	var rawSlice []json.RawMessage
	if err := d.decoder.Decode(&rawSlice); err != nil {
		return err
	}

	if err := d.checkTrailingData(); err != nil {
		return err
	}

	sliceElem := slicePtr.Elem()
	elemType := sliceElem.Type().Elem()
	result := reflect.MakeSlice(sliceElem.Type(), 0, len(rawSlice))

	for i, rawItem := range rawSlice {
		var rawMap map[string]json.RawMessage
		if err := json.Unmarshal(rawItem, &rawMap); err != nil {
			return fmt.Errorf("failed to parse JSON array element %d: %w", i, err)
		}

		if d.disallowUnknownFields {
			itemPtr := reflect.New(elemType)
			if err := d.checkUnknownFields(rawMap, itemPtr.Interface()); err != nil {
				return err
			}
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

// checkTrailingData validates there's no trailing data after the JSON value.
func (d *Decoder) checkTrailingData() error {
	if d.allowTrailingData {
		return nil
	}

	if d.decoder.More() {
		return fmt.Errorf("json: invalid character after top-level value")
	}

	_, err := d.decoder.Token()
	if err == nil {
		return fmt.Errorf("json: invalid character after top-level value")
	}
	if err != io.EOF {
		return err
	}

	return nil
}

// DisallowUnknownFields causes the Decoder to return an error when the destination
// is a struct and the input contains object keys which do not match any
// non-ignored, exported fields in the destination (case-sensitive).
func (d *Decoder) DisallowUnknownFields() *Decoder {
	d.disallowUnknownFields = true
	return d
}

// AllowTrailingData allows the Decoder to accept input with data after the JSON value.
// By default, the decoder rejects trailing data to prevent injection attacks.
// Use this only when you need to parse multiple JSON values from a stream.
func (d *Decoder) AllowTrailingData() *Decoder {
	d.allowTrailingData = true
	return d
}

// checkUnknownFields validates that all JSON keys match struct fields (case-sensitive)
func (d *Decoder) checkUnknownFields(rawMap map[string]json.RawMessage, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return nil
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return nil
	}

	rt := rv.Type()
	fieldMap := cachedBuildFieldMap(rt)

	// Check for unknown fields
	for jsonKey := range rawMap {
		if _, exists := fieldMap[jsonKey]; !exists {
			return fmt.Errorf("json: unknown field %q", jsonKey)
		}
	}

	return nil
}

// UseNumber causes the Decoder to unmarshal a number into an interface{} as a
// Number instead of as a float64.
func (d *Decoder) UseNumber() *Decoder {
	d.decoder.UseNumber()
	return d
}

// More returns true if there is another element in the current array or object being parsed.
func (d *Decoder) More() bool {
	return d.decoder.More()
}

// Token returns the next JSON token in the input stream.
func (d *Decoder) Token() (json.Token, error) {
	return d.decoder.Token()
}
