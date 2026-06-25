package rssjson

import "strings"

type SourceMetadata struct {
	sources      []MetadataSource
	legacyGUID   string
	legacyPostID string
}

type MetadataSource struct {
	Name   string
	Object MetadataObject
}

type MetadataObject []MetadataField

type MetadataField struct {
	Name  string
	Value MetadataValue
}

type MetadataValue struct {
	text      string
	object    MetadataObject
	hasObject bool
}

func NewSourceMetadata(source string, object MetadataObject) SourceMetadata {
	if strings.TrimSpace(source) == "" || len(object) == 0 {
		return SourceMetadata{}
	}
	return SourceMetadata{
		sources: []MetadataSource{{
			Name:   source,
			Object: object,
		}},
	}
}

func MetadataText(name string, value string) MetadataField {
	return MetadataField{Name: name, Value: MetadataString(value)}
}

func MetadataNested(name string, object MetadataObject) MetadataField {
	return MetadataField{Name: name, Value: MetadataObjectValue(object)}
}

func MetadataString(value string) MetadataValue {
	return MetadataValue{text: value}
}

func MetadataObjectValue(object MetadataObject) MetadataValue {
	return MetadataValue{object: object, hasObject: true}
}

func (m SourceMetadata) Object(source string) (MetadataObject, bool) {
	for _, item := range m.sources {
		if item.Name == source {
			return item.Object, true
		}
	}
	return nil, false
}

func (m SourceMetadata) IsZero() bool {
	return len(m.sources) == 0 && m.legacyGUID == "" && m.legacyPostID == ""
}

func (o MetadataObject) Text(name string) (string, bool) {
	for _, field := range o {
		if field.Name == name && !field.Value.hasObject {
			return field.Value.text, true
		}
	}
	return "", false
}

func (o MetadataObject) Object(name string) (MetadataObject, bool) {
	for _, field := range o {
		if field.Name == name && field.Value.hasObject {
			return field.Value.object, true
		}
	}
	return nil, false
}

func (m SourceMetadata) forSource(source string, contentEncoded string) SourceMetadata {
	switch source {
	case "cybersecuritynews":
		m = m.withCybersecurityNews(contentEncoded)
	case "darkreading", "bleepingcomputer":
		if _, ok := m.Object(source); !ok && m.legacyGUID != "" {
			m = NewSourceMetadata(source, MetadataObject{
				MetadataText("guid_is_permalink", m.legacyGUID),
			})
		}
	}
	m.legacyGUID = ""
	m.legacyPostID = ""
	return m
}

func (m SourceMetadata) withCybersecurityNews(contentEncoded string) SourceMetadata {
	object, ok := m.Object("cybersecuritynews")
	if !ok && m.legacyGUID == "" && m.legacyPostID == "" && contentEncoded == "" {
		return m
	}
	if !ok {
		object = MetadataObject{
			MetadataText("guid_is_permalink", m.legacyGUID),
			MetadataText("post_id", m.legacyPostID),
		}
	}
	if _, exists := object.Text("content_encoded"); !exists {
		object = append(object, MetadataText("content_encoded", contentEncoded))
	}
	return NewSourceMetadata("cybersecuritynews", object)
}
