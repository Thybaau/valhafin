package utils

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func ExportData(data interface{}, filename string, format string) error {
	switch strings.ToLower(format) {
	case "json":
		return exportJSON(data, filename)
	case "csv":
		return exportCSV(data, filename)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func exportJSON(data interface{}, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func exportCSV(data interface{}, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write UTF-8 BOM for Excel compatibility
	file.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(file)
	writer.Comma = ';'
	defer writer.Flush()

	// Get slice value
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("data must be a slice")
	}

	if v.Len() == 0 {
		return nil
	}

	// Get headers from struct tags
	firstItem := v.Index(0)
	headers := getCSVHeaders(firstItem.Type())
	writer.Write(headers)

	// Write rows
	for i := 0; i < v.Len(); i++ {
		row := getCSVRow(v.Index(i), headers)
		writer.Write(row)
	}

	return nil
}

func getCSVHeaders(t reflect.Type) []string {
	var headers []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("csv")
		if tag != "" && tag != "-" {
			headers = append(headers, tag)
		}
	}
	return headers
}

func getCSVRow(v reflect.Value, headers []string) []string {
	row := make([]string, len(headers))
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("csv")
		if tag == "" || tag == "-" {
			continue
		}

		// Find index in headers
		idx := -1
		for j, h := range headers {
			if h == tag {
				idx = j
				break
			}
		}

		if idx == -1 {
			continue
		}

		fieldValue := v.Field(i)
		row[idx] = formatValue(fieldValue)
	}

	return row
}

func formatValue(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Float32, reflect.Float64:
		val := v.Float()
		// Format with comma as decimal separator (French format)
		str := strconv.FormatFloat(val, 'f', -1, 64)
		return strings.Replace(str, ".", ",", 1)
	case reflect.Bool:
		if v.Bool() {
			return "True"
		}
		return "False"
	default:
		return ""
	}
}
