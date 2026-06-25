package rssjson

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func (m SourceMetadata) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, source := range m.sources {
		if i > 0 {
			buf.WriteByte(',')
		}
		name, err := marshalJSONString(source.Name)
		if err != nil {
			return nil, fmt.Errorf("marshal metadata source name: %w", err)
		}
		object, err := source.Object.MarshalJSON()
		if err != nil {
			return nil, err
		}
		buf.Write(name)
		buf.WriteByte(':')
		buf.Write(object)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func (m *SourceMetadata) UnmarshalJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := consumeObjectStart(decoder); err != nil {
		return err
	}

	next := SourceMetadata{}
	for decoder.More() {
		key, err := consumeObjectKey(decoder)
		if err != nil {
			return err
		}
		switch key {
		case "guid_is_permalink":
			if err := decoder.Decode(&next.legacyGUID); err != nil {
				return fmt.Errorf("decode legacy guid_is_permalink: %w", err)
			}
		case "post_id":
			if err := decoder.Decode(&next.legacyPostID); err != nil {
				return fmt.Errorf("decode legacy post_id: %w", err)
			}
		default:
			var object MetadataObject
			if err := decoder.Decode(&object); err != nil {
				return fmt.Errorf("decode source metadata %s: %w", key, err)
			}
			next.sources = append(next.sources, MetadataSource{Name: key, Object: object})
		}
	}
	if err := consumeObjectEnd(decoder); err != nil {
		return err
	}
	*m = next
	return nil
}

func (o MetadataObject) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, field := range o {
		if i > 0 {
			buf.WriteByte(',')
		}
		name, err := marshalJSONString(field.Name)
		if err != nil {
			return nil, fmt.Errorf("marshal metadata field name: %w", err)
		}
		value, err := field.Value.MarshalJSON()
		if err != nil {
			return nil, err
		}
		buf.Write(name)
		buf.WriteByte(':')
		buf.Write(value)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func (o *MetadataObject) UnmarshalJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := consumeObjectStart(decoder); err != nil {
		return err
	}

	fields := MetadataObject{}
	for decoder.More() {
		key, err := consumeObjectKey(decoder)
		if err != nil {
			return err
		}
		var value MetadataValue
		if err := decoder.Decode(&value); err != nil {
			return fmt.Errorf("decode metadata field %s: %w", key, err)
		}
		fields = append(fields, MetadataField{Name: key, Value: value})
	}
	if err := consumeObjectEnd(decoder); err != nil {
		return err
	}
	*o = fields
	return nil
}

func (v MetadataValue) MarshalJSON() ([]byte, error) {
	if v.hasObject {
		return v.object.MarshalJSON()
	}
	return marshalJSONString(v.text)
}

func (v *MetadataValue) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return fmt.Errorf("empty metadata value")
	}
	if trimmed[0] == '{' {
		var object MetadataObject
		if err := json.Unmarshal(trimmed, &object); err != nil {
			return err
		}
		*v = MetadataObjectValue(object)
		return nil
	}
	var text string
	if err := json.Unmarshal(trimmed, &text); err != nil {
		return err
	}
	*v = MetadataString(text)
	return nil
}

func marshalJSONString(value string) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}

func consumeObjectStart(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '{' {
		return fmt.Errorf("metadata value must be an object")
	}
	return nil
}

func consumeObjectKey(decoder *json.Decoder) (string, error) {
	token, err := decoder.Token()
	if err != nil {
		return "", err
	}
	key, ok := token.(string)
	if !ok {
		return "", fmt.Errorf("metadata object key must be a string")
	}
	return key, nil
}

func consumeObjectEnd(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '}' {
		return fmt.Errorf("metadata object is not closed")
	}
	return nil
}
