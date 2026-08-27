package rules

import "testing"

func TestHasFilters(t *testing.T) {
	if (Rule{Name: "x", AlbumName: "y"}).HasFilters() {
		t.Fatal("empty rule should have no filters")
	}
	if !(Rule{City: "Krakow"}).HasFilters() {
		t.Fatal("city is a filter")
	}
	if !(Rule{CameraMake: "FUJIFILM"}).HasFilters() {
		t.Fatal("camera make is a filter")
	}
}

func TestValidate(t *testing.T) {
	err := Rule{Name: "r", AlbumName: "a"}.Validate()
	if err == nil {
		t.Fatal("expected error without filters")
	}
	err = Rule{Name: "r", AlbumName: "a", Country: "Poland"}.Validate()
	if err != nil {
		t.Fatal(err)
	}
	err = Rule{AlbumName: "a", City: "x"}.Validate()
	if err == nil {
		t.Fatal("expected error without name")
	}
}
