package services

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/jjudge-oj/api/types"
)

type testcasePair struct {
	in  []byte
	out []byte
}

// ProcessTestcaseFiles verifies testcase files, uploads them to storage, and returns updated groups with testcase metadata.
func (s *ProblemService) ProcessTestcaseFiles(ctx context.Context, problemID int, files map[string][]byte, tcGroups []types.TestcaseGroup) ([]types.TestcaseGroup, error) {
	if problemID < 1 {
		return nil, errors.New("invalid problem id")
	}
	if len(files) == 0 {
		return nil, errors.New("testcase files are required")
	}
	if s.storage == nil {
		return nil, errors.New("object storage is not configured")
	}

	// Validate testcase structure
	if _, err := readAndValidateTestcases(files, tcGroups); err != nil {
		return nil, err
	}

	// Process each group and update testcases with storage keys and hashes
	updatedGroups := make([]types.TestcaseGroup, len(tcGroups))
	for groupIndex, group := range tcGroups {
		updatedGroups[groupIndex] = group
		updatedTestcases := make([]types.Testcase, len(group.Testcases))

		for testcaseIndex, tc := range group.Testcases {
			// Get the input and output data
			inData, ok := files[tc.InKey]
			if !ok {
				return nil, fmt.Errorf("missing input file for testcase %d_%d (key: %s)", group.Ordinal, tc.Ordinal, tc.InKey)
			}
			outData, ok := files[tc.OutKey]
			if !ok {
				return nil, fmt.Errorf("missing output file for testcase %d_%d (key: %s)", group.Ordinal, tc.Ordinal, tc.OutKey)
			}

			// Generate storage keys with standard naming
			inStorageKey := fmt.Sprintf("testcases/%d/%d_%d_%d.in", problemID, problemID, group.Ordinal, tc.Ordinal)
			outStorageKey := fmt.Sprintf("testcases/%d/%d_%d_%d.out", problemID, problemID, group.Ordinal, tc.Ordinal)

			// Upload input file
			if err := s.storage.Put(ctx, inStorageKey, bytes.NewReader(inData), int64(len(inData)), "application/octet-stream"); err != nil {
				return nil, fmt.Errorf("failed to upload input file %s: %w", inStorageKey, err)
			}

			// Upload output file
			if err := s.storage.Put(ctx, outStorageKey, bytes.NewReader(outData), int64(len(outData)), "application/octet-stream"); err != nil {
				return nil, fmt.Errorf("failed to upload output file %s: %w", outStorageKey, err)
			}

			// Compute hash of input + output
			hash := computeTestcaseHash(inData, outData)

			// Update testcase with storage keys and hash
			updatedTestcases[testcaseIndex] = types.Testcase{
				Ordinal:         tc.Ordinal,
				TestcaseGroupID: tc.TestcaseGroupID,
				Input:           tc.Input,  // May be empty for hidden testcases
				Output:          tc.Output, // May be empty for hidden testcases
				InKey:           inStorageKey,
				OutKey:          outStorageKey,
				Hash:            hash,
				IsHidden:        tc.IsHidden,
			}
		}

		updatedGroups[groupIndex].Testcases = updatedTestcases
	}

	return updatedGroups, nil
}

// computeTestcaseHash computes SHA256 hash of input and output data concatenated.
func computeTestcaseHash(inData, outData []byte) string {
	hasher := sha256.New()
	hasher.Write(inData)
	hasher.Write([]byte{0}) // Separator
	hasher.Write(outData)
	return hex.EncodeToString(hasher.Sum(nil))
}

// ProcessTestcasesFromZip processes a ZIP archive whose entries follow the naming convention
// <subtask_ordinal>_<testcase_ordinal>.in / .out. groupsMeta provides the name and points for
// each subtask group (matched by ordinal); groups not present in groupsMeta get empty names and
// zero points. Files are uploaded to storage and the fully-populated groups are returned.
func (s *ProblemService) ProcessTestcasesFromZip(ctx context.Context, problemID int, zipData []byte, groupsMeta []types.TestcaseGroup) ([]types.TestcaseGroup, error) {
	if problemID < 1 {
		return nil, errors.New("invalid problem id")
	}
	if s.storage == nil {
		return nil, errors.New("object storage is not configured")
	}

	pairs, err := parseArchiveTestcases(zipData)
	if err != nil {
		return nil, err
	}

	// Collect and validate subtask ordinals are consecutive from 0
	subtaskOrdinals := make([]int, 0, len(pairs))
	for ord := range pairs {
		subtaskOrdinals = append(subtaskOrdinals, ord)
	}
	sort.Ints(subtaskOrdinals)
	for i, ord := range subtaskOrdinals {
		if ord != i {
			return nil, fmt.Errorf("subtask ordinals must be consecutive starting from 0 (expected %d, got %d)", i, ord)
		}
	}

	// Build lookup for group metadata by ordinal
	groupByOrdinal := make(map[int]types.TestcaseGroup, len(groupsMeta))
	for _, g := range groupsMeta {
		groupByOrdinal[g.Ordinal] = g
	}

	updatedGroups := make([]types.TestcaseGroup, 0, len(subtaskOrdinals))
	for _, subtaskOrd := range subtaskOrdinals {
		testcasePairs := pairs[subtaskOrd]

		tcOrdinals := make([]int, 0, len(testcasePairs))
		for tcOrd := range testcasePairs {
			tcOrdinals = append(tcOrdinals, tcOrd)
		}
		sort.Ints(tcOrdinals)
		for i, ord := range tcOrdinals {
			if ord != i {
				return nil, fmt.Errorf("testcase ordinals in subtask %d must be consecutive starting from 0 (expected %d, got %d)", subtaskOrd, i, ord)
			}
		}

		group := types.TestcaseGroup{Ordinal: subtaskOrd}
		if meta, ok := groupByOrdinal[subtaskOrd]; ok {
			group.Name = meta.Name
			group.Points = meta.Points
		}

		updatedTestcases := make([]types.Testcase, 0, len(tcOrdinals))
		for _, tcOrd := range tcOrdinals {
			pair := testcasePairs[tcOrd]

			inKey := fmt.Sprintf("testcases/%d/%d_%d_%d.in", problemID, problemID, subtaskOrd, tcOrd)
			outKey := fmt.Sprintf("testcases/%d/%d_%d_%d.out", problemID, problemID, subtaskOrd, tcOrd)

			if err := s.storage.Put(ctx, inKey, bytes.NewReader(pair.in), int64(len(pair.in)), "application/octet-stream"); err != nil {
				return nil, fmt.Errorf("failed to upload %s: %w", inKey, err)
			}
			if err := s.storage.Put(ctx, outKey, bytes.NewReader(pair.out), int64(len(pair.out)), "application/octet-stream"); err != nil {
				return nil, fmt.Errorf("failed to upload %s: %w", outKey, err)
			}

			updatedTestcases = append(updatedTestcases, types.Testcase{
				Ordinal: tcOrd,
				InKey:   inKey,
				OutKey:  outKey,
				Hash:    computeTestcaseHash(pair.in, pair.out),
			})
		}

		group.Testcases = updatedTestcases
		updatedGroups = append(updatedGroups, group)
	}

	return updatedGroups, nil
}

// parseArchiveTestcases detects the archive format by magic bytes and delegates accordingly.
func parseArchiveTestcases(data []byte) (map[int]map[int]*testcasePair, error) {
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		return parseTarGzTestcases(data)
	}
	return parseZipTestcases(data)
}

// parseTarGzTestcases reads a .tar.gz archive and returns a map[subtaskOrd][testcaseOrd]*testcasePair.
func parseTarGzTestcases(data []byte) (map[int]map[int]*testcasePair, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("invalid gzip stream: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	pairs := make(map[int]map[int]*testcasePair)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading tar archive: %w", err)
		}
		if hdr.Typeflag == tar.TypeDir {
			continue
		}

		name := path.Base(hdr.Name)
		subtaskOrd, tcOrd, ext, ok := parseTestcaseFilename(name)
		if !ok {
			return nil, fmt.Errorf("invalid testcase filename %q (expected <subtask>_<testcase>.in or .out)", name)
		}

		entryData, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("failed to read tar entry %s: %w", hdr.Name, err)
		}

		if pairs[subtaskOrd] == nil {
			pairs[subtaskOrd] = make(map[int]*testcasePair)
		}
		if pairs[subtaskOrd][tcOrd] == nil {
			pairs[subtaskOrd][tcOrd] = &testcasePair{}
		}
		p := pairs[subtaskOrd][tcOrd]
		if ext == "in" {
			if p.in != nil {
				return nil, fmt.Errorf("duplicate input file for testcase %d_%d", subtaskOrd, tcOrd)
			}
			p.in = entryData
		} else {
			if p.out != nil {
				return nil, fmt.Errorf("duplicate output file for testcase %d_%d", subtaskOrd, tcOrd)
			}
			p.out = entryData
		}
	}

	if len(pairs) == 0 {
		return nil, errors.New("archive contains no valid testcase files")
	}
	for subtaskOrd, testcases := range pairs {
		for tcOrd, p := range testcases {
			if p.in == nil {
				return nil, fmt.Errorf("missing .in file for testcase %d_%d", subtaskOrd, tcOrd)
			}
			if p.out == nil {
				return nil, fmt.Errorf("missing .out file for testcase %d_%d", subtaskOrd, tcOrd)
			}
		}
	}

	return pairs, nil
}

// parseZipTestcases reads a zip archive and returns a map[subtaskOrd][testcaseOrd]*testcasePair.
func parseZipTestcases(zipData []byte) (map[int]map[int]*testcasePair, error) {
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("invalid zip file: %w", err)
	}

	pairs := make(map[int]map[int]*testcasePair)

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		name := path.Base(f.Name)
		subtaskOrd, tcOrd, ext, ok := parseTestcaseFilename(name)
		if !ok {
			return nil, fmt.Errorf("invalid testcase filename %q (expected <subtask>_<testcase>.in or .out)", name)
		}

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open zip entry %s: %w", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read zip entry %s: %w", f.Name, err)
		}

		if pairs[subtaskOrd] == nil {
			pairs[subtaskOrd] = make(map[int]*testcasePair)
		}
		if pairs[subtaskOrd][tcOrd] == nil {
			pairs[subtaskOrd][tcOrd] = &testcasePair{}
		}
		p := pairs[subtaskOrd][tcOrd]
		if ext == "in" {
			if p.in != nil {
				return nil, fmt.Errorf("duplicate input file for testcase %d_%d", subtaskOrd, tcOrd)
			}
			p.in = data
		} else {
			if p.out != nil {
				return nil, fmt.Errorf("duplicate output file for testcase %d_%d", subtaskOrd, tcOrd)
			}
			p.out = data
		}
	}

	if len(pairs) == 0 {
		return nil, errors.New("zip contains no valid testcase files")
	}

	for subtaskOrd, testcases := range pairs {
		for tcOrd, p := range testcases {
			if p.in == nil {
				return nil, fmt.Errorf("missing .in file for testcase %d_%d", subtaskOrd, tcOrd)
			}
			if p.out == nil {
				return nil, fmt.Errorf("missing .out file for testcase %d_%d", subtaskOrd, tcOrd)
			}
		}
	}

	return pairs, nil
}

// parseTestcaseFilename parses a filename like "1_2.in" into (subtask=1, testcase=2, ext="in").
func parseTestcaseFilename(name string) (subtask, testcase int, ext string, ok bool) {
	dot := strings.LastIndex(name, ".")
	if dot < 0 {
		return 0, 0, "", false
	}
	ext = strings.ToLower(name[dot+1:])
	if ext != "in" && ext != "out" {
		return 0, 0, "", false
	}
	base := name[:dot]
	underscore := strings.Index(base, "_")
	if underscore < 0 {
		return 0, 0, "", false
	}
	subtask, err1 := strconv.Atoi(base[:underscore])
	testcase, err2 := strconv.Atoi(base[underscore+1:])
	if err1 != nil || err2 != nil || subtask < 0 || testcase < 0 {
		return 0, 0, "", false
	}
	return subtask, testcase, ext, true
}

// readAndValidateTestcases validates the structure of testcase groups and files.
func readAndValidateTestcases(files map[string][]byte, tcGroups []types.TestcaseGroup) (map[string]*testcasePair, error) {
	testcaseData := make(map[string]*testcasePair)
	keySeen := make(map[string]struct{})

	for _, group := range tcGroups {
		if group.Ordinal < 0 {
			return nil, fmt.Errorf("invalid testcase group ordinal: %d", group.Ordinal)
		}

		// Track ordinals to ensure they're consecutive
		ordinals := make(map[int]bool)

		for _, tc := range group.Testcases {
			if tc.Ordinal < 0 {
				return nil, fmt.Errorf("invalid testcase ordinal: %d in group %d", tc.Ordinal, group.Ordinal)
			}

			// Check for duplicate ordinals
			if ordinals[tc.Ordinal] {
				return nil, fmt.Errorf("duplicate testcase ordinal: %d in group %d", tc.Ordinal, group.Ordinal)
			}
			ordinals[tc.Ordinal] = true

			// Validate keys are provided
			if strings.TrimSpace(tc.InKey) == "" || strings.TrimSpace(tc.OutKey) == "" {
				return nil, fmt.Errorf("testcase %d_%d must include in_key and out_key", group.Ordinal, tc.Ordinal)
			}

			// Check for duplicate keys across all testcases
			if _, ok := keySeen[tc.InKey]; ok {
				return nil, fmt.Errorf("duplicate in_key: %s", tc.InKey)
			}
			if _, ok := keySeen[tc.OutKey]; ok {
				return nil, fmt.Errorf("duplicate out_key: %s", tc.OutKey)
			}
			keySeen[tc.InKey] = struct{}{}
			keySeen[tc.OutKey] = struct{}{}

			// Verify files exist
			inData, ok := files[tc.InKey]
			if !ok {
				return nil, fmt.Errorf("missing testcase input for key: %s", tc.InKey)
			}
			outData, ok := files[tc.OutKey]
			if !ok {
				return nil, fmt.Errorf("missing testcase output for key: %s", tc.OutKey)
			}

			testcaseData[tc.InKey] = &testcasePair{in: inData, out: outData}
		}

		// Verify ordinals are consecutive starting from 0
		ordinalList := make([]int, 0, len(ordinals))
		for ord := range ordinals {
			ordinalList = append(ordinalList, ord)
		}
		sort.Ints(ordinalList)

		for expected, actual := range ordinalList {
			if expected != actual {
				return nil, fmt.Errorf("testcase ordinals must be consecutive starting from 0 in group %d (expected %d, got %d)", group.Ordinal, expected, actual)
			}
		}
	}

	return testcaseData, nil
}
