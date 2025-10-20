package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"github.com/xuri/excelize/v2"
)
const (
	outputResultsPath = "output_results.xlsx"
	rawInputPath      = "inputs/RAW_Med_Update_October_2025.xlsx"
	outputPath        = "raw_generic_name_corrected.xlsx"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("update failed: %v", err)
	}
}

func run() error {
	nameMap, err := buildNameMap(outputResultsPath)
	if err != nil {
		return fmt.Errorf("build name map: %w", err)
	}

	if len(nameMap) == 0 {
		return errors.New("no mappings were generated from output results workbook")
	}

	if err := rewriteMedications(rawInputPath, outputPath, nameMap); err != nil {
		return fmt.Errorf("rewrite medications: %w", err)
	}

	fmt.Printf("wrote updated workbook to %s\n", outputPath)
	return nil
}

func buildNameMap(path string) (map[string]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open workbook %s: %w", path, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.Printf("warning: close workbook %s: %v", path, cerr)
		}
	}()

	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("read rows from %s: %w", sheetName, err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("workbook %s is empty", path)
	}

	header := rows[0]
	colIndex := make(map[string]int, len(header))
	for idx, name := range header {
		colIndex[strings.TrimSpace(name)] = idx
	}

	requiredCols := []string{"Original Name", "Ingredient Name", "Matched Name", "Has Match"}
	for _, col := range requiredCols {
		if _, ok := colIndex[col]; !ok {
			return nil, fmt.Errorf("column %q not found in %s", col, path)
		}
	}

	get := func(row []string, col string) string {
		idx, ok := colIndex[col]
		if !ok || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	nameMap := make(map[string]string, len(rows)-1)
	for _, row := range rows[1:] {
		original := get(row, "Original Name")
		if original == "" {
			continue
		}

		ingredient := get(row, "Ingredient Name")
		matched := get(row, "Matched Name")
		hasMatch := strings.EqualFold(get(row, "Has Match"), "true")

		switch {
		case ingredient != "":
			nameMap[original] = ingredient
		case matched != "":
			nameMap[original] = matched
		case !hasMatch:
			nameMap[original] = original
		default:
			nameMap[original] = original
		}
	}

	return nameMap, nil
}

func rewriteMedications(inputPath, outputPath string, nameMap map[string]string) error {
	f, err := excelize.OpenFile(inputPath)
	if err != nil {
		return fmt.Errorf("open workbook %s: %w", inputPath, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.Printf("warning: close workbook %s: %v", inputPath, cerr)
		}
	}()

	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return fmt.Errorf("read rows from %s: %w", sheetName, err)
	}
	if len(rows) == 0 {
		return fmt.Errorf("workbook %s is empty", inputPath)
	}

	header := rows[0]
	medCols := medicationColumns(header)
	if len(medCols) == 0 {
		return errors.New("no medication columns found (expected columns like med_med, med_med_2, etc.)")
	}

	for rowIdx := 2; rowIdx <= len(rows); rowIdx++ { // 1-based row index; skip header
		for _, col := range medCols {
			cell, err := excelize.CoordinatesToCellName(col.medIdx+1, rowIdx)
			if err != nil {
				return fmt.Errorf("map coordinates (%d,%d) to cell: %w", col.medIdx+1, rowIdx, err)
			}

			current, err := f.GetCellValue(sheetName, cell, excelize.Options{})
			if err != nil {
				return fmt.Errorf("read cell %s: %w", cell, err)
			}

			currentTrimmed := strings.TrimSpace(current)
			if currentTrimmed == "" {
				continue
			}

			if replacement, ok := nameMap[currentTrimmed]; ok && replacement != "" && replacement != current {
				if col.commentIdx >= 0 {
					commentCell, err := excelize.CoordinatesToCellName(col.commentIdx+1, rowIdx)
					if err != nil {
						return fmt.Errorf("map comment coordinates (%d,%d) to cell: %w", col.commentIdx+1, rowIdx, err)
					}
					existingComment, err := f.GetCellValue(sheetName, commentCell, excelize.Options{})
					if err != nil {
						return fmt.Errorf("read comment cell %s: %w", commentCell, err)
					}
					newComment := currentTrimmed
					if trimmedComment := strings.TrimSpace(existingComment); trimmedComment != "" {
						newComment = fmt.Sprintf("%s - %s", currentTrimmed, trimmedComment)
					}
					if err := f.SetCellValue(sheetName, commentCell, newComment); err != nil {
						return fmt.Errorf("set comment cell %s: %w", commentCell, err)
					}
				}
				if err := f.SetCellValue(sheetName, cell, replacement); err != nil {
					return fmt.Errorf("set cell %s: %w", cell, err)
				}
			}
		}
	}

	if err := ensureDir(filepath.Dir(outputPath)); err != nil {
		return err
	}

	if err := f.SaveAs(outputPath); err != nil {
		return fmt.Errorf("save updated workbook: %w", err)
	}

	return nil
}

type medColumn struct {
	medIdx     int
	commentIdx int
}

func medicationColumns(header []string) []medColumn {
	indexByName := make(map[string]int, len(header))
	for idx, col := range header {
		name := strings.TrimSpace(col)
		if name == "" {
			continue
		}
		indexByName[name] = idx
	}

	var cols []medColumn
	for idx, col := range header {
		name := strings.TrimSpace(col)
		if !isMedicationColumn(name) {
			continue
		}
		commentName := commentColumnName(name)
		commentIdx := -1
		if cIdx, ok := indexByName[commentName]; ok {
			commentIdx = cIdx
		}
		cols = append(cols, medColumn{medIdx: idx, commentIdx: commentIdx})
	}
	return cols
}

func isMedicationColumn(name string) bool {
	if name == "med_med" {
		return true
	}
	if strings.HasPrefix(name, "med_med_") {
		return true
	}
	return false
}

func commentColumnName(name string) string {
	if name == "med_med" {
		return "med_comment"
	}
	if strings.HasPrefix(name, "med_med_") {
		return "med_comment" + strings.TrimPrefix(name, "med_med")
	}
	return ""
}

func ensureDir(path string) error {
	if path == "" || path == "." {
		return nil
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", path, err)
	}
	return nil
}
