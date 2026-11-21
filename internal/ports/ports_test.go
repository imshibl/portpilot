package ports

import (
	"reflect"
	"testing"
)

func TestValidatePort(t *testing.T) {
	for _, port := range []int{1, 65535} {
		if _, err := Validate(port); err != nil {
			t.Errorf("Validate(%d) returned error: %v", port, err)
		}
	}
	for _, port := range []int{-1, 0, 65536} {
		if _, err := Validate(port); err == nil {
			t.Errorf("Validate(%d) accepted an invalid port", port)
		}
	}
}

func TestParseRange(t *testing.T) {
	got, err := ParseRange("3000-4000")
	if err != nil {
		t.Fatal(err)
	}
	if got != (Range{Start: 3000, End: 4000}) {
		t.Fatalf("unexpected range: %+v", got)
	}
	for _, value := range []string{"3000", "4000-3000", "0-80", "1-65536", "a-3"} {
		if _, err := ParseRange(value); err == nil {
			t.Errorf("ParseRange(%q) accepted invalid input", value)
		}
	}
}

func TestFilterRangeAndProcess(t *testing.T) {
	listeners := []Listener{
		{Port: 3001, Process: "Node"},
		{Port: 3002, Process: "python"},
		{Port: 4001, Process: "node"},
	}
	rangeFilter := Range{Start: 3000, End: 3999}
	got := Filter(listeners, &rangeFilter, "NODE")
	want := []Listener{listeners[0]}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Filter() = %#v, want %#v", got, want)
	}
}
