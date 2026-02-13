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
	"github.com/jjudge-oj/apiserver/internal/utils"
)

type testcasePair struct {
	in  []byte
	out []byte
}

// GetTestcaseBundleFromFiles verifies testcase files and returns their bundle metadata.
func (s *ProblemService) GetTestcaseBundleFromFiles(ctx context.Context, problemID int, files map[string][]byte, tcGroups []types.TestcaseGroup) (types.TestcaseBundle, error) {
	if problemID < 1 {
		return types.TestcaseBundle{}, errors.New("invalid problem id")
	}
	if len(files) == 0 {
		return types.TestcaseBundle{}, errors.New("testcase files are required")
	}
	if s.storage == nil {
		return types.TestcaseBundle{}, errors.New("object storage is not configured")
	}

	updatedGroups, archiveFiles, err := readTestcasesFromFiles(problemID, files, tcGroups)
	if err != nil {
		return types.TestcaseBundle{}, err
	}

	hash, err := hashTestcaseFiles(archiveFiles)
	if err != nil {
		return types.TestcaseBundle{}, err
	}

	archiveData, err := utils.BuildTarGz(archiveFiles)
	if err != nil {
		return types.TestcaseBundle{}, err
	}

	objectKey := fmt.Sprintf("problems/%d/testcases/%s.tar.gz", problemID, hash)
	if err := s.storage.Put(ctx, objectKey, bytes.NewReader(archiveData), int64(len(archiveData)), "application/gzip"); err != nil {
		return types.TestcaseBundle{}, fmt.Errorf("failed to upload testcase bundle: %w", err)
	}

	tcBundle := types.TestcaseBundle{
		ObjectKey:      objectKey,
		SHA256:         hash,
		TestcaseGroups: updatedGroups,
	}
	return tcBundle, nil
}

func hashTestcaseFiles(files map[string][]byte) (string, error) {
	keys := make([]string, 0, len(files))
	for key := range files {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	hasher := sha256.New()
	for _, key := range keys {
		if _, err := hasher.Write([]byte(key)); err != nil {
			return "", err
		}
		if _, err := hasher.Write([]byte{0}); err != nil {
			return "", err
		}
		if _, err := hasher.Write(files[key]); err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func readTestcasesFromFiles(problemID int, files map[string][]byte, tcGroups []types.TestcaseGroup) ([]types.TestcaseGroup, map[string][]byte, error) {
	groupOrders := make([]map[int]*testcasePair, len(tcGroups))
	for i := range tcGroups {
		groupOrders[i] = make(map[int]*testcasePair)
	}

	keySeen := make(map[string]struct{})
	for groupIndex := range tcGroups {
		groupOrdinal := tcGroups[groupIndex].Ordinal
		if groupOrdinal < 0 {
			return nil, nil, fmt.Errorf("invalid testcase group ordinal: %d", groupOrdinal)
		}

		for testcaseIndex := range tcGroups[groupIndex].Testcases {
			testcase := tcGroups[groupIndex].Testcases[testcaseIndex]
			if testcase.Ordinal < 0 {
				return nil, nil, fmt.Errorf("invalid testcase ordinal: %d", testcase.Ordinal)
			}
			if strings.TrimSpace(testcase.InKey) == "" || strings.TrimSpace(testcase.OutKey) == "" {
				return nil, nil, fmt.Errorf("testcase %d_%d must include in_key and out_key", groupOrdinal, testcase.Ordinal)
			}
			if _, ok := keySeen[testcase.InKey]; ok {
				return nil, nil, fmt.Errorf("duplicate in_key: %s", testcase.InKey)
			}
			if _, ok := keySeen[testcase.OutKey]; ok {
				return nil, nil, fmt.Errorf("duplicate out_key: %s", testcase.OutKey)
			}
			keySeen[testcase.InKey] = struct{}{}
			keySeen[testcase.OutKey] = struct{}{}

			inData, ok := files[testcase.InKey]
			if !ok {
				return nil, nil, fmt.Errorf("missing testcase input for key: %s", testcase.InKey)
			}
			outData, ok := files[testcase.OutKey]
			if !ok {
				return nil, nil, fmt.Errorf("missing testcase output for key: %s", testcase.OutKey)
			}

			p := groupOrders[groupIndex][testcase.Ordinal]
			if p == nil {
				p = &testcasePair{}
				groupOrders[groupIndex][testcase.Ordinal] = p
			}
			if p.in != nil || p.out != nil {
				return nil, nil, fmt.Errorf("duplicate testcase ordinal: %d_%d", groupOrdinal, testcase.Ordinal)
			}
			p.in = inData
			p.out = outData
		}
	}

	for groupIndex, orders := range groupOrders {
		if len(orders) == 0 {
			continue
		}

		testcaseOrdinals := make([]int, 0, len(orders))
		for ordinal, pair := range orders {
			if pair.in == nil || pair.out == nil {
				return nil, nil, fmt.Errorf("testcase %d_%d must have both .in and .out files", tcGroups[groupIndex].Ordinal, ordinal)
			}
			testcaseOrdinals = append(testcaseOrdinals, ordinal)
		}

		sort.Ints(testcaseOrdinals)
		for expected, ordinal := range testcaseOrdinals {
			if ordinal != expected {
				return nil, nil, fmt.Errorf("testcase ordinals must be consecutive in group %d", tcGroups[groupIndex].Ordinal)
			}
		}
	}

	archiveFiles, err := buildTestcaseArchiveFiles(problemID, tcGroups, groupOrders)
	if err != nil {
		return nil, nil, err
	}

	return tcGroups, archiveFiles, nil
}

func buildTestcaseArchiveFiles(problemID int, tcGroups []types.TestcaseGroup, groupOrders []map[int]*testcasePair) (map[string][]byte, error) {
	archiveFiles := make(map[string][]byte)
	for groupIndex := range groupOrders {
		orders := make([]int, 0, len(groupOrders[groupIndex]))
		for order := range groupOrders[groupIndex] {
			orders = append(orders, order)
		}
		sort.Ints(orders)
		for _, order := range orders {
			pair := groupOrders[groupIndex][order]
			if pair == nil || pair.in == nil || pair.out == nil {
				return nil, fmt.Errorf("testcase %d_%d must have both .in and .out files", tcGroups[groupIndex].Ordinal, order)
			}
			base := fmt.Sprintf("%d_%d_%d", problemID, tcGroups[groupIndex].Ordinal, order)
			inName := base + ".in"
			outName := base + ".out"
			if _, exists := archiveFiles[inName]; exists {
				return nil, fmt.Errorf("duplicate testcase filename: %s", inName)
			}
			if _, exists := archiveFiles[outName]; exists {
				return nil, fmt.Errorf("duplicate testcase filename: %s", outName)
			}
			archiveFiles[inName] = pair.in
			archiveFiles[outName] = pair.out
		}
	}
	if len(archiveFiles) == 0 {
		return nil, errors.New("testcase files are required")
	}
	return archiveFiles, nil
}
