package main

import (
	"bufio"
	"fmt"
	medusa "medusa/meducore"
	"os"
	"time"
)

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func writeFileAsync(w *bufio.Writer, found *medusa.Found) {

	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {

		found.Lock()
		for key, line := range found.List {
			if !line.OutputOK {
				line.OutputOK = true
				found.List[key] = line
				line := fmt.Sprintf("[%d] %s - %d len", line.StatusCode, line.URL, len(line.Body))
				fmt.Fprintln(w, line)
			}
		}
		found.Unlock()
		w.Flush()
	}
}
