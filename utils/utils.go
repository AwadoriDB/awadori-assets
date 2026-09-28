package utils

import (
	"encoding/json"
	"os"
	"path/filepath"

	"awano-winter/rich"
)

func WriteToJsonFile(instance any, dst string) {
	data, err := json.MarshalIndent(instance, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		panic(err)
	}
	rich.Info("Writing json file '%s' done.", dst)
}

func ReadFromJsonFile(src string, v any) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
