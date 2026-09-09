package appdir

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDBPathIsInsideUserConfigDir(t *testing.T) {
	path, err := DBPath()
	if err != nil {
		t.Fatalf("DBPath: %v", err)
	}
	if filepath.Base(path) != "data.db" {
		t.Errorf("fisierul = %q, vrem data.db", filepath.Base(path))
	}
	if !strings.Contains(path, "nota-de-receptie") {
		t.Errorf("drumul %q nu contine numele aplicatiei", path)
	}
}

func TestDBPathCreatesTheDirectory(t *testing.T) {
	path, err := DBPath()
	if err != nil {
		t.Fatalf("DBPath: %v", err)
	}
	info, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("directorul nu a fost creat: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("%q nu este director", filepath.Dir(path))
	}
}
