package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Fprint

func main() {
	reader := bufio.NewReader(os.Stdin)
	var cmd string
	var err error

	for {
		fmt.Fprint(os.Stdout, "$ ")
		// Wait for user input
		cmd, err = reader.ReadString('\n')
		if err != nil {
			break
		}
		fmt.Fprintf(os.Stdout, "%s: not found\n", strings.TrimSpace(cmd))
	}
}
