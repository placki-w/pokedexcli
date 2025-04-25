package main // Use the same package as your main code

import (
	"testing" // Import testing package
)

func TestCleanInput(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{
			input:    "  hello  world  ",
			expected: "hello  world", // cleanInput trims spaces but won't collapse spaces between words
		},
		{
			input:    "Charmander Bulbasaur PIKACHU",
			expected: "charmander bulbasaur pikachu", // cleanInput lowercase conversion
		},
	}

	for _, c := range cases {
		actual := cleanInput(c.input) // Call the function being tested
		if actual != c.expected {
			t.Errorf("Mismatch for input '%s': got '%s', expected '%s'", c.input, actual, c.expected)
		}
	}
}
