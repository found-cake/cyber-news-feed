package jsonstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/found-cake/cyber-news-feed/internal/rssdoc"
)

func Load(outputDir string, source string) (rssdoc.Document, error) {
	path := filepath.Join(outputDir, source+".json")
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return rssdoc.Document{
			SchemaVersion: rssdoc.SchemaVersion,
			Source:        source,
			Articles:      []rssdoc.Article{},
		}, nil
	}
	if err != nil {
		return rssdoc.Document{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() {
		_ = file.Close()
	}()

	var doc rssdoc.Document
	if err := json.NewDecoder(file).Decode(&doc); err != nil {
		return rssdoc.Document{}, fmt.Errorf("decode %s: %w", path, err)
	}
	return doc, nil
}

func Write(outputDir string, doc rssdoc.Document) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", outputDir, err)
	}
	path := filepath.Join(outputDir, doc.Source+".json")
	encoded, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", doc.Source, err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
