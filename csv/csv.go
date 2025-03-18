package csv

import (
	"encoding/csv"
	"fmt"
	"os"
)

// Writer handles writing data to CSV files
type Writer struct {
	file   *os.File
	writer *csv.Writer
	path   string
}

const outputDir = "out"

// NewWriter creates a new CSV writer for the given file name
func NewWriter(filename string) (*Writer, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("error creating output directory: %v", err)
	}

	// Create the CSV file
	file, err := os.Create(outputDir + "/" + filename)
	if err != nil {
		return nil, fmt.Errorf("error creating CSV file: %v", err)
	}

	return &Writer{
		file:   file,
		writer: csv.NewWriter(file),
		path:   outputDir + "/" + filename,
	}, nil
}

// WriteHeader writes the header row to the CSV file
func (w *Writer) WriteHeader(header []string) error {
	return w.writer.Write(header)
}

// WriteRow writes a row of data to the CSV file
func (w *Writer) WriteRow(row []string) error {
	return w.writer.Write(row)
}

// Close closes the CSV file and flushes any buffered data
func (w *Writer) Close() error {
	w.writer.Flush()
	return w.file.Close()
}

// Path returns the path of the CSV file
func (w *Writer) Path() string {
	return w.path
}
