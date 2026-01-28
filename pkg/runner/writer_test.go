package runner

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/cyberspacesec/go-snir/pkg/models"
)

// createTestResult creates a test result for all writer tests
func createTestResult() *models.Result {
	return &models.Result{
		URL:            "https://example.com",
		Title:          "Example Domain",
		Filename:       "test_screenshot.png",
		ResponseCode:   200,
		ResponseReason: "OK",
		ProbedAt:       models.Now(),
		FinalURL:       "https://example.com",
	}
}

// TestNewJSONLWriter tests the creation of a new JSONLWriter
func TestNewJSONLWriter(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "jsonl_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "test.jsonl")

	// Create a new JSONLWriter
	writer, err := NewJSONLWriter(filePath)
	if err != nil {
		t.Fatalf("NewJSONLWriter returned error: %v", err)
	}
	defer writer.Close()

	// Verify writer is not nil
	if writer == nil {
		t.Fatal("NewJSONLWriter should return non-nil writer")
	}

	// Verify file was created
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Expected file %s to be created", filePath)
	}
}

// TestJSONLWriterWrite tests writing a result to a JSONL file
func TestJSONLWriterWrite(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "jsonl_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "test.jsonl")

	// Create a new JSONLWriter
	writer, err := NewJSONLWriter(filePath)
	if err != nil {
		t.Fatalf("NewJSONLWriter returned error: %v", err)
	}

	// Create a test result
	result := createTestResult()

	// Write the result to the file
	if err := writer.Write(result); err != nil {
		t.Fatalf("JSONLWriter.Write returned error: %v", err)
	}

	// Close the writer
	if err := writer.Close(); err != nil {
		t.Fatalf("JSONLWriter.Close returned error: %v", err)
	}

	// Verify the file contains the expected data
	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	// Read the content
	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// Parse the JSON
	var parsedResult models.Result
	if err := json.Unmarshal(content, &parsedResult); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify the result
	if parsedResult.URL != result.URL {
		t.Errorf("Expected URL %s, got %s", result.URL, parsedResult.URL)
	}
	if parsedResult.Title != result.Title {
		t.Errorf("Expected Title %s, got %s", result.Title, parsedResult.Title)
	}
}

// TestNewCSVWriter tests the creation of a new CSVWriter
func TestNewCSVWriter(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "csv_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "test.csv")

	// Create a new CSVWriter
	writer, err := NewCSVWriter(filePath)
	if err != nil {
		t.Fatalf("NewCSVWriter returned error: %v", err)
	}
	defer writer.Close()

	// Verify writer is not nil
	if writer == nil {
		t.Fatal("NewCSVWriter should return non-nil writer")
	}

	// Verify file was created
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Expected file %s to be created", filePath)
	}
}

// TestCSVWriterWrite tests writing a result to a CSV file
func TestCSVWriterWrite(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "csv_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "test.csv")

	// Create a new CSVWriter
	writer, err := NewCSVWriter(filePath)
	if err != nil {
		t.Fatalf("NewCSVWriter returned error: %v", err)
	}

	// Create a test result
	result := createTestResult()

	// Write the result to the file
	if err := writer.Write(result); err != nil {
		t.Fatalf("CSVWriter.Write returned error: %v", err)
	}

	// Close the writer
	if err := writer.Close(); err != nil {
		t.Fatalf("CSVWriter.Close returned error: %v", err)
	}

	// Verify the file contains the expected data
	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	// Read CSV records
	csvReader := csv.NewReader(file)
	records, err := csvReader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	// Verify header exists
	if len(records) < 2 {
		t.Fatalf("Expected at least 2 rows (header + data), got %d", len(records))
	}

	// Verify the result data (second row)
	resultRow := records[1]
	if resultRow[0] != result.URL {
		t.Errorf("Expected URL %s, got %s", result.URL, resultRow[0])
	}
	if resultRow[1] != result.Title {
		t.Errorf("Expected Title %s, got %s", result.Title, resultRow[1])
	}
}

// TestStdoutWriter tests the stdout writer
func TestStdoutWriter(t *testing.T) {
	// Create a new StdoutWriter
	writer := NewStdoutWriter()

	// Verify writer is not nil
	if writer == nil {
		t.Fatal("NewStdoutWriter should return non-nil writer")
	}

	// Create a test result
	result := createTestResult()

	// Write the result to stdout (can't easily verify output, just check for no errors)
	if err := writer.Write(result); err != nil {
		t.Fatalf("StdoutWriter.Write returned error: %v", err)
	}

	// Close the writer
	if err := writer.Close(); err != nil {
		t.Fatalf("StdoutWriter.Close returned error: %v", err)
	}
}

// TestCreateWriters tests the CreateWriters function
func TestCreateWriters(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "writers_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test cases
	tests := []struct {
		name          string
		options       *Options
		expectedCount int
	}{
		{
			name: "All writers enabled",
			options: &Options{
				DB: struct {
					Enable bool
					Path   string
				}{
					Enable: true,
					Path:   ":memory:",
				},
				Writer: struct {
					Db        bool
					DbURI     string
					DbDebug   bool
					Jsonl     bool
					JsonlFile string
					Csv       bool
					CsvFile   string
					Stdout    bool
				}{
					Jsonl:     true,
					JsonlFile: filepath.Join(tempDir, "test.jsonl"),
					Csv:       true,
					CsvFile:   filepath.Join(tempDir, "test.csv"),
					Stdout:    true,
				},
			},
			expectedCount: 4, // DB, JSONL, CSV, Stdout
		},
		{
			name: "No writers enabled (defaults to stdout)",
			options: &Options{
				DB: struct {
					Enable bool
					Path   string
				}{
					Enable: false,
				},
				Writer: struct {
					Db        bool
					DbURI     string
					DbDebug   bool
					Jsonl     bool
					JsonlFile string
					Csv       bool
					CsvFile   string
					Stdout    bool
				}{
					Jsonl:  false,
					Csv:    false,
					Stdout: false,
				},
			},
			expectedCount: 1, // Just Stdout
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writers, err := CreateWriters(tt.options)
			if err != nil {
				t.Fatalf("CreateWriters returned error: %v", err)
			}

			if len(writers) != tt.expectedCount {
				t.Errorf("Expected %d writers, got %d", tt.expectedCount, len(writers))
			}

			// Clean up
			for _, writer := range writers {
				writer.Close()
			}
		})
	}
}
