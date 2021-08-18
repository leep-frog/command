package command

import (
	"bufio"
	"os"
)

func ReadStdin(lineFn func(string) error) error {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		lineFn(scanner.Text())
	}
	return scanner.Err()
}
