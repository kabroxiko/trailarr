package internal

import (
	"encoding/json"
	"os"
)

// ReadJSONFile reads a JSON file and unmarshals into the provided destination
func ReadJSONFile(path string, dest interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// WriteJSONFile marshals the source and writes to the given path
func WriteJSONFile(path string, src interface{}) error {
	data, err := json.MarshalIndent(src, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
