package main

import (
	"fmt"
	"strings"
)

func main() {
	fmt.Printf("Hello, World!")

}

func cleanInput(text string) []string {
	var result []string
	words := strings.Fields(text)
	for _, word := range words {
		word = strings.ToLower(word)
		result = append(result, word)
	}
	return result
}
