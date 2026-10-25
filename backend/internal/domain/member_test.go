package domain

import "testing"

func TestFirstMemberColorIsCanonicalPaletteHead(t *testing.T) {
	if FirstMemberColor != "#0e7861" {
		t.Fatalf("FirstMemberColor = %q, want frontend palette head #0e7861", FirstMemberColor)
	}
	normalized, err := NormalizeColor(FirstMemberColor)
	if err != nil || normalized != FirstMemberColor {
		t.Fatalf("NormalizeColor(FirstMemberColor) = %q, %v; want canonical color", normalized, err)
	}
}
