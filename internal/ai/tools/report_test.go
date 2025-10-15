package tools

import (
	"testing"
	"time"
)

func TestTextFormatter(t *testing.T) {
	report := ReviewReport{
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		FileCount: 2,
		Summary:   "Test summary",
		Findings: []Finding{
			{
				Severity:    SeverityWarning,
				Title:       "Test Finding",
				Description: "This is a test",
				FilePath:    "test.go",
				LineNumber:  10,
				Category:    "test",
			},
		},
	}

	formatter := &TextFormatter{}
	output, err := formatter.Format(report)
	if err != nil {
		t.Fatalf("TextFormatter failed: %v", err)
	}

	if output == "" {
		t.Error("TextFormatter returned empty output")
	}

	if !contains(output, "Test Finding") {
		t.Error("Output missing test finding title")
	}
}

func TestJSONFormatter(t *testing.T) {
	report := ReviewReport{
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		FileCount: 1,
		Summary:   "JSON test",
		Findings: []Finding{
			{
				Severity:    SeverityError,
				Title:       "Error Test",
				Description: "Test error",
				Category:    "test",
			},
		},
	}

	formatter := &JSONFormatter{}
	output, err := formatter.Format(report)
	if err != nil {
		t.Fatalf("JSONFormatter failed: %v", err)
	}

	if !contains(output, `"title"`) || !contains(output, `"Error Test"`) {
		t.Error("JSON output missing expected fields")
	}
}

func TestMarkdownFormatter(t *testing.T) {
	report := ReviewReport{
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		FileCount: 1,
		Summary:   "Markdown test",
		Findings: []Finding{
			{
				Severity:    SeverityInfo,
				Title:       "Info Test",
				Description: "Test info",
				Category:    "test",
			},
		},
	}

	formatter := &MarkdownFormatter{}
	output, err := formatter.Format(report)
	if err != nil {
		t.Fatalf("MarkdownFormatter failed: %v", err)
	}

	if !contains(output, "# Code Review Report") {
		t.Error("Markdown output missing header")
	}
}

func TestSARIFFormatter(t *testing.T) {
	report := ReviewReport{
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		FileCount: 1,
		Summary:   "SARIF test",
		Findings: []Finding{
			{
				Severity:    SeverityWarning,
				Title:       "Warning Test",
				Description: "Test warning",
				FilePath:    "src/main.go",
				LineNumber:  42,
				Category:    "code-quality",
			},
		},
	}

	formatter := &SARIFFormatter{}
	output, err := formatter.Format(report)
	if err != nil {
		t.Fatalf("SARIFFormatter failed: %v", err)
	}

	if !contains(output, `"version": "2.1.0"`) {
		t.Error("SARIF output missing version")
	}

	if !contains(output, "code-quality") {
		t.Error("SARIF output missing category")
	}
}

func TestGetFormatter(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"text", "*tools.TextFormatter"},
		{"json", "*tools.JSONFormatter"},
		{"md", "*tools.MarkdownFormatter"},
		{"markdown", "*tools.MarkdownFormatter"},
		{"sarif", "*tools.SARIFFormatter"},
		{"unknown", "*tools.TextFormatter"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			formatter := GetFormatter(tt.format)
			typeName := getTypeName(formatter)
			if typeName != tt.expected {
				t.Errorf("GetFormatter(%q) = %v, want %v", tt.format, typeName, tt.expected)
			}
		})
	}
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != substr && len(s) >= len(substr) &&
		(s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func getTypeName(v interface{}) string {
	switch v.(type) {
	case *TextFormatter:
		return "*tools.TextFormatter"
	case *JSONFormatter:
		return "*tools.JSONFormatter"
	case *MarkdownFormatter:
		return "*tools.MarkdownFormatter"
	case *SARIFFormatter:
		return "*tools.SARIFFormatter"
	default:
		return "unknown"
	}
}
