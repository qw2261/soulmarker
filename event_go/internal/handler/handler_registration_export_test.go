package handler

import "testing"

func TestSpreadsheetSafeCellBlocksFormulaInjection(t *testing.T) {
	tests := map[string]string{
		"=SUM(A1:A2)":  "'=SUM(A1:A2)",
		" +cmd|' /C":   "' +cmd|' /C",
		"\t@malicious": "'\t@malicious",
		"safe value":   "safe value",
		"":             "",
	}
	for input, expected := range tests {
		if actual := spreadsheetSafeCell(input); actual != expected {
			t.Errorf("spreadsheetSafeCell(%q)=%q want %q", input, actual, expected)
		}
	}
}
