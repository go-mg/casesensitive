package xml

import (
	"encoding/xml"
	"fmt"
	"io"
	"reflect"
)

// Decoder reads and decodes XML values from an input stream with case-sensitive field matching.
type Decoder struct {
	decoder               *xml.Decoder
	disallowUnknownFields bool
}

// NewDecoder returns a new decoder that reads from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		decoder: xml.NewDecoder(r),
	}
}

// Decode reads the next XML-encoded value from its input and stores it in the value pointed to by v.
// Unlike standard xml.Decoder, this decoder is case-sensitive.
// v must be a pointer to a struct.
func (d *Decoder) Decode(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("v must be a non-nil pointer")
	}

	if rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("v must be a pointer to a struct")
	}

	rt := rv.Elem().Type()
	fieldMap := cachedBuildFieldMap(rt)
	attrMap := cachedBuildAttrMap(rt)

	// Find the start element
	for {
		tok, err := d.decoder.Token()
		if err == io.EOF {
			return err
		}
		if err != nil {
			return fmt.Errorf("failed to parse XML: %w", err)
		}

		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		// Process attributes
		for _, attr := range se.Attr {
			fi, exists := attrMap[attr.Name.Local]
			if !exists {
				if d.disallowUnknownFields {
					return fmt.Errorf("xml: unknown attribute %q", attr.Name.Local)
				}
				continue
			}
			field := initAndGetField(rv.Elem(), fi.index)
			if field.CanSet() {
				if err := setFieldFromString(field, attr.Value); err != nil {
					return fmt.Errorf("failed to set attribute %s: %w", fi.name, err)
				}
			}
		}

		// Process children
		if d.disallowUnknownFields {
			return d.decodeChildrenStrict(rv.Elem(), fieldMap)
		}
		return decodeChildren(d.decoder, rv.Elem(), fieldMap)
	}
}

// decodeChildrenStrict reads child elements with unknown field validation.
func (d *Decoder) decodeChildrenStrict(rv reflect.Value, fieldMap map[string]fieldInfo) error {
	for {
		tok, err := d.decoder.Token()
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
				return fmt.Errorf("xml: unknown field %q", t.Name.Local)
			}

			field := initAndGetField(rv, fi.index)
			if !field.CanSet() {
				if err := d.decoder.Skip(); err != nil {
					return err
				}
				continue
			}

			if err := decodeField(d.decoder, field, t); err != nil {
				return fmt.Errorf("failed to unmarshal field %s: %w", fi.name, err)
			}

		case xml.EndElement:
			return nil
		}
	}
}

// DisallowUnknownFields causes the Decoder to return an error when the input
// contains element names which do not match any non-ignored, exported fields
// in the destination (case-sensitive).
func (d *Decoder) DisallowUnknownFields() *Decoder {
	d.disallowUnknownFields = true
	return d
}

// Token returns the next XML token in the input stream.
func (d *Decoder) Token() (xml.Token, error) {
	return d.decoder.Token()
}
