package generator

import (
	"fmt"
	"testing"
)

const casesDir = "test_cases"

func TestIsGeneratedFile(t *testing.T) {
	type testCase struct {
		input    string
		expected bool
	}
	for testNo, test := range []testCase{
		{"some.go", false},
		{"some.gen.go", true},
	} {
		test := test // Capture
		t.Run(fmt.Sprint(`case `, testNo), func(t *testing.T) {
			t.Log("test: ", test)
			if received := isGeneratedFile(test.input); received != test.expected {
				t.Errorf(`expected: %v, received: %v`, test.expected, received)
			}
		})
	}
}
