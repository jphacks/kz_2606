package main

import (
	"io/fs"
	"testing"
)

func TestStaticFSIncludesBuildDirectory(t *testing.T) {
	files, err := staticFS()
	if err != nil {
		t.Fatalf("create embedded static fs: %v", err)
	}
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		t.Fatalf("read embedded build directory: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("embedded build directory is empty")
	}
}
