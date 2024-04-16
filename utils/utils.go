package utils

import (
	"bufio"
	"fmt"
	"os"
)

func Show(content []string) {
	for _, str := range content {
		fmt.Fprintf(os.Stderr, "%s\n", str)
	}
}
func ReadFileByLine(path string) (*[]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &lines, nil
}
