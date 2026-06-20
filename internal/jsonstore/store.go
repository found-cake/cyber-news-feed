package jsonstore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/found-cake/cyber-news-feed/pkg/rssjson"
)

func Load(outputDir string, source string) (rssjson.Document, error) {
	path := filepath.Join(outputDir, source+".json")
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return rssjson.Document{
			SchemaVersion: rssjson.SchemaVersion,
			Source:        source,
			Articles:      []rssjson.Article{},
		}, nil
	}
	if err != nil {
		return rssjson.Document{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() {
		_ = file.Close()
	}()

	var doc rssjson.Document
	if err := json.NewDecoder(file).Decode(&doc); err != nil {
		return rssjson.Document{}, fmt.Errorf("decode %s: %w", path, err)
	}
	return doc, nil
}

func Write(outputDir string, doc rssjson.Document) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", outputDir, err)
	}
	path := filepath.Join(outputDir, doc.Source+".json")
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(doc); err != nil {
		return fmt.Errorf("encode %s: %w", doc.Source, err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
