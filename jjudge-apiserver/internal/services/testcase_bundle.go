package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
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
