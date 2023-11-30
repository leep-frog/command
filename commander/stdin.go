package commander

import (
	"bufio"
	"os"
)

// ReadStdin executes lineFn on each line read from stdin.
func ReadStdin(lineFn func(string) error) error {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		lineFn(scanner.Text())
	}
	return scanner.Err()
}
