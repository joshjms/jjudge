package utils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"
)

// BuildTarGz creates a deterministic tar.gz archive from the provided files.
func BuildTarGz(files map[string][]byte) ([]byte, error) {
	if len(files) == 0 {
		return nil, errors.New("no files provided")
	}

	keys := make([]string, 0, len(files))
	for name := range files {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzipWriter)

	seen := make(map[string]struct{}, len(keys))
	for _, name := range keys {
		clean := sanitizeTarPath(name)
		if clean == "" {
			return nil, fmt.Errorf("invalid file name: %q", name)
		}
		if _, exists := seen[clean]; exists {
			return nil, fmt.Errorf("duplicate file name after sanitization: %s", clean)
		}
		seen[clean] = struct{}{}

		data := files[name]
		header := &tar.Header{
			Name:    clean,
			Mode:    0o644,
			Size:    int64(len(data)),
			ModTime: time.Unix(0, 0),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, fmt.Errorf("failed to write tar header: %w", err)
		}
		if _, err := tarWriter.Write(data); err != nil {
			return nil, fmt.Errorf("failed to write tar data: %w", err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close tar writer: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

func sanitizeTarPath(name string) string {
	if name == "" {
		return ""
	}
	clean := path.Clean(strings.ReplaceAll(name, "\\", "/"))
	if clean == "." || strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return ""
	}
	return clean
}
