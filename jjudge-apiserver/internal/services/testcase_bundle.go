package services

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jjudge-oj/apiserver/types"
)

var testcaseFilenamePattern = regexp.MustCompile(`^\d+_\d+\.(in|out)$`)

const testcaseExtractDirEnv = "JJUDGE_TESTCASE_EXTRACT_DIR"

// GetTestcaseBundleFromArchive verifies the testcase bundle data and returns its SHA-256 hash.
func (s *ProblemService) GetTestcaseBundleFromArchive(filename string, data []byte, tcGroups []types.TestcaseGroup) (types.TestcaseBundle, error) {
	if len(data) == 0 {
		return types.TestcaseBundle{}, errors.New("empty bundle data")
	}

	hash := sha256.Sum256(data)
	actual := hex.EncodeToString(hash[:])

	tcBundle := types.TestcaseBundle{}
	tcBundle.ObjectKey = filename
	tcBundle.SHA256 = actual

	lower := strings.ToLower(strings.TrimSpace(filename))
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return types.TestcaseBundle{}, errors.New("zip bundles are not supported")
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		gr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return types.TestcaseBundle{}, errors.New("invalid tar.gz bundle")
		}
		defer gr.Close()

		tr := tar.NewReader(gr)
		updatedGroups, err := readTestcaseFromTarGz(tr, tcGroups)
		if err != nil {
			return types.TestcaseBundle{}, err
		}
		tcBundle.TestcaseGroups = updatedGroups
		return tcBundle, nil
	default:
		return types.TestcaseBundle{}, errors.New("unsupported bundle format")
	}
}

func readTestcaseFromTarGz(tr *tar.Reader, tcGroups []types.TestcaseGroup) ([]types.TestcaseGroup, error) {
	extractBase := strings.TrimSpace(os.Getenv(testcaseExtractDirEnv))
	if extractBase == "" {
		extractBase = "."
	}

	tempDir, err := os.MkdirTemp(extractBase, "testcase-bundle-")
	if err != nil {
		return nil, fmt.Errorf("failed to create bundle extract directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	type pair struct {
		in  bool
		out bool
	}

	groupOrders := make([]map[int]*pair, len(tcGroups))
	for i := range tcGroups {
		groupOrders[i] = make(map[int]*pair)
	}

	count := 0
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, errors.New("invalid tar.gz bundle")
		}
		if header.FileInfo().IsDir() {
			continue
		}
		if !header.FileInfo().Mode().IsRegular() {
			return nil, errors.New("bundle contains unsupported entries")
		}
		if err := validateBundleFilename(header.Name); err != nil {
			return nil, err
		}

		base := path.Base(path.Clean(header.Name))
		groupOrder, testcaseOrder, ext, err := parseTestcaseFilename(base)
		if err != nil {
			return nil, err
		}
		if groupOrder < 0 || groupOrder >= len(tcGroups) {
			return nil, fmt.Errorf("testcase group %d does not exist", groupOrder)
		}

		p := groupOrders[groupOrder][testcaseOrder]
		if p == nil {
			p = &pair{}
			groupOrders[groupOrder][testcaseOrder] = p
		}
		switch ext {
		case "in":
			if p.in {
				return nil, fmt.Errorf("duplicate testcase input: %d_%d.in", groupOrder, testcaseOrder)
			}
			p.in = true
		case "out":
			if p.out {
				return nil, fmt.Errorf("duplicate testcase output: %d_%d.out", groupOrder, testcaseOrder)
			}
			p.out = true
		default:
			return nil, fmt.Errorf("invalid testcase filename: %s", base)
		}

		dst := filepath.Join(tempDir, base)
		outFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return nil, fmt.Errorf("failed to extract testcase: %w", err)
		}
		if _, err := io.Copy(outFile, tr); err != nil {
			_ = outFile.Close()
			return nil, fmt.Errorf("failed to extract testcase: %w", err)
		}
		if err := outFile.Close(); err != nil {
			return nil, fmt.Errorf("failed to extract testcase: %w", err)
		}
		count++
	}

	if count == 0 {
		return nil, errors.New("bundle has no testcases")
	}

	for groupOrder, orders := range groupOrders {
		if len(orders) == 0 {
			continue
		}

		testcaseOrders := make([]int, 0, len(orders))
		for order, pair := range orders {
			if !pair.in || !pair.out {
				return nil, fmt.Errorf("testcase %d_%d must have both .in and .out files", groupOrder, order)
			}
			testcaseOrders = append(testcaseOrders, order)
		}

		sort.Ints(testcaseOrders)
		for expected, order := range testcaseOrders {
			if order != expected {
				return nil, fmt.Errorf("testcase order must be consecutive in group %d", groupOrder)
			}
		}

		for _, order := range testcaseOrders {
			tcGroups[groupOrder].Testcases = append(tcGroups[groupOrder].Testcases, types.Testcase{
				OrderID: order,
			})
		}
	}

	return tcGroups, nil
}

func parseTestcaseFilename(base string) (int, int, string, error) {
	ext := strings.TrimPrefix(path.Ext(base), ".")
	name := strings.TrimSuffix(base, "."+ext)
	parts := strings.Split(name, "_")
	if ext == "" || len(parts) != 2 {
		return 0, 0, "", fmt.Errorf("invalid testcase filename: %s", base)
	}
	groupOrder, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, "", fmt.Errorf("invalid testcase filename: %s", base)
	}
	testcaseOrder, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, "", fmt.Errorf("invalid testcase filename: %s", base)
	}
	if groupOrder < 0 || testcaseOrder < 0 {
		return 0, 0, "", fmt.Errorf("invalid testcase filename: %s", base)
	}
	return groupOrder, testcaseOrder, ext, nil
}

func validateBundleFilename(name string) error {
	clean := path.Clean(name)
	if clean == "." {
		return errors.New("invalid testcase filename")
	}
	base := path.Base(clean)
	if base != clean {
		return errors.New("bundle must not contain directories")
	}
	if strings.Contains(base, `\`) {
		return errors.New("invalid testcase filename")
	}
	if !testcaseFilenamePattern.MatchString(base) {
		return fmt.Errorf("invalid testcase filename: %s", base)
	}
	return nil
}
