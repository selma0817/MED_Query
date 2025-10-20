package excel

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"

	"fuzzyMed/internal/processor"
)

// ReadMedications loads medications from the first sheet of the workbook and
// returns the values along with the header names that were inspected.
func ReadMedications(path string) ([]string, []string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open excel file: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil, fmt.Errorf("excel contains no sheets")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, nil, fmt.Errorf("read rows: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil, fmt.Errorf("excel contains no data")
	}

	headers := rows[0]
	var (
		medColumnIndices []int
		medColumnNames   []string
	)
	for i, header := range headers {
		headerLower := strings.ToLower(strings.TrimSpace(header))
		if strings.HasPrefix(headerLower, "med_med") {
			medColumnIndices = append(medColumnIndices, i)
			medColumnNames = append(medColumnNames, header)
		}
	}
	if len(medColumnIndices) == 0 {
		return nil, nil, fmt.Errorf("no medication columns found (expected columns starting with 'med_med')")
	}

	var medications []string
	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		for _, colIdx := range medColumnIndices {
			if colIdx < len(row) {
				if medName := strings.TrimSpace(row[colIdx]); medName != "" {
					medications = append(medications, medName)
				}
			}
		}
	}

	return medications, medColumnNames, nil
}

// WriteResults writes the processor outcomes to disk.
func WriteResults(path string, results []processor.Result) error {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Results"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("create sheet: %w", err)
	}

	headers := []string{
		"Original Name",
		"Corrected Name",
		"Was Typo Corrected",
		"Has Match",
		"Matched Name",
		"Matched RxCUI",
		"Score",
		"TTY",
		"TTY Description",
		"Ingredient Name",
		"Ingredient RxCUI",
		"Error",
	}

	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	for rowIdx, result := range results {
		rowNumber := rowIdx + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNumber), result.OriginalName)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNumber), result.CorrectedName)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNumber), result.WasTypoCorrected)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowNumber), result.HasMatch)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowNumber), result.MatchedName)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowNumber), result.MatchedRxCUI)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowNumber), result.Score)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", rowNumber), result.TTY)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", rowNumber), result.TTYDescription)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", rowNumber), result.IngredientName)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", rowNumber), result.IngredientRxCUI)
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", rowNumber), result.Error)
	}

	f.SetActiveSheet(index)
	f.DeleteSheet("Sheet1")

	if err := f.SaveAs(path); err != nil {
		return fmt.Errorf("save excel: %w", err)
	}

	return nil
}
