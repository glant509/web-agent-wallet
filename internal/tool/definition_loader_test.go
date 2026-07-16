package tool

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefinition(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "market.json")
	if err := os.WriteFile(path, []byte(`{
  "name":"market_token_overview",
  "description":"Get market data",
  "input_schema":{
    "type":"object",
    "properties":{"vs_currency":{"type":"string"}}
  }
}`), 0o644); err != nil {
		t.Fatalf("write definition: %v", err)
	}

	definition, err := LoadDefinition(path)
	if err != nil {
		t.Fatalf("load definition: %v", err)
	}
	if definition.Name != "market_token_overview" {
		t.Fatalf("unexpected name: %q", definition.Name)
	}
	if len(definition.InputSchema) == 0 {
		t.Fatal("expected input schema")
	}
}
