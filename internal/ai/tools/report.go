package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Severity levels for findings
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// Finding represents a single issue or suggestion
type Finding struct {
	Severity    Severity `json:"severity"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	FilePath    string   `json:"file_path,omitempty"`
	LineNumber  int      `json:"line_number,omitempty"`
	Category    string   `json:"category"`
}

// ReviewReport contains the complete review results
type ReviewReport struct {
	Timestamp time.Time `json:"timestamp"`
	Findings  []Finding `json:"findings"`
	Summary   string    `json:"summary"`
	FileCount int       `json:"file_count"`
}

// ReportFormatter defines the interface for report formatters
type ReportFormatter interface {
	Format(report ReviewReport) (string, error)
}

// TextFormatter formats reports as plain text
type TextFormatter struct{}

func (f *TextFormatter) Format(report ReviewReport) (string, error) {
	var out strings.Builder

	out.WriteString("=== Code Review Report ===\n\n")
	out.WriteString(fmt.Sprintf("Generated: %s\n", report.Timestamp.Format(time.RFC3339)))
	out.WriteString(fmt.Sprintf("Files analyzed: %d\n", report.FileCount))
	out.WriteString(fmt.Sprintf("Total findings: %d\n\n", len(report.Findings)))

	if report.Summary != "" {
		out.WriteString("Summary:\n")
		out.WriteString(report.Summary)
		out.WriteString("\n\n")
	}

	if len(report.Findings) > 0 {
		out.WriteString("Findings:\n\n")
		for i, finding := range report.Findings {
			out.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, finding.Severity, finding.Title))
			if finding.FilePath != "" {
				out.WriteString(fmt.Sprintf("   File: %s", finding.FilePath))
				if finding.LineNumber > 0 {
					out.WriteString(fmt.Sprintf(":%d", finding.LineNumber))
				}
				out.WriteString("\n")
			}
			out.WriteString(fmt.Sprintf("   %s\n\n", finding.Description))
		}
	}

	return out.String(), nil
}

// MarkdownFormatter formats reports as Markdown
type MarkdownFormatter struct{}

func (f *MarkdownFormatter) Format(report ReviewReport) (string, error) {
	var out strings.Builder

	out.WriteString("# Code Review Report\n\n")
	out.WriteString(fmt.Sprintf("**Generated:** %s\n\n", report.Timestamp.Format(time.RFC3339)))
	out.WriteString(fmt.Sprintf("**Files analyzed:** %d\n\n", report.FileCount))
	out.WriteString(fmt.Sprintf("**Total findings:** %d\n\n", len(report.Findings)))

	if report.Summary != "" {
		out.WriteString("## Summary\n\n")
		out.WriteString(report.Summary)
		out.WriteString("\n\n")
	}

	if len(report.Findings) > 0 {
		out.WriteString("## Findings\n\n")
		for i, finding := range report.Findings {
			severity := strings.ToUpper(string(finding.Severity))
			out.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, finding.Title))
			out.WriteString(fmt.Sprintf("**Severity:** `%s`\n\n", severity))
			if finding.FilePath != "" {
				out.WriteString(fmt.Sprintf("**Location:** `%s", finding.FilePath))
				if finding.LineNumber > 0 {
					out.WriteString(fmt.Sprintf(":%d", finding.LineNumber))
				}
				out.WriteString("`\n\n")
			}
			out.WriteString(finding.Description)
			out.WriteString("\n\n---\n\n")
		}
	}

	return out.String(), nil
}

// JSONFormatter formats reports as JSON
type JSONFormatter struct{}

func (f *JSONFormatter) Format(report ReviewReport) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// SARIFFormatter formats reports as SARIF (Static Analysis Results Interchange Format)
type SARIFFormatter struct{}

type sarifReport struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string `json:"name"`
	InformationUri string `json:"informationUri,omitempty"`
	Version        string `json:"version"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations,omitempty"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           *sarifRegion          `json:"region,omitempty"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}

func (f *SARIFFormatter) Format(report ReviewReport) (string, error) {
	results := make([]sarifResult, 0, len(report.Findings))

	for _, finding := range report.Findings {
		level := "note"
		switch finding.Severity {
		case SeverityWarning:
			level = "warning"
		case SeverityError:
			level = "error"
		}

		result := sarifResult{
			RuleID: finding.Category,
			Level:  level,
			Message: sarifMessage{
				Text: fmt.Sprintf("%s: %s", finding.Title, finding.Description),
			},
		}

		if finding.FilePath != "" {
			location := sarifLocation{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{
						URI: finding.FilePath,
					},
				},
			}
			if finding.LineNumber > 0 {
				location.PhysicalLocation.Region = &sarifRegion{
					StartLine: finding.LineNumber,
				}
			}
			result.Locations = []sarifLocation{location}
		}

		results = append(results, result)
	}

	sarif := sarifReport{
		Version: "2.1.0",
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Runs: []sarifRun{
			{
				Tool: sarifTool{
					Driver: sarifDriver{
						Name:    "Critica AI",
						Version: "1.0.0",
					},
				},
				Results: results,
			},
		},
	}

	data, err := json.MarshalIndent(sarif, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal SARIF: %w", err)
	}
	return string(data), nil
}

// GetFormatter returns the appropriate formatter for the given format
func GetFormatter(format string) ReportFormatter {
	switch format {
	case "json":
		return &JSONFormatter{}
	case "md", "markdown":
		return &MarkdownFormatter{}
	case "sarif":
		return &SARIFFormatter{}
	default:
		return &TextFormatter{}
	}
}
