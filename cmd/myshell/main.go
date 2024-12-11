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
	var cmd_line string
	var err error

	for {
		fmt.Fprint(os.Stdout, "$ ")
		// Wait for user input
		cmd_line, err = reader.ReadString('\n')
		if err != nil {
			break
		}
		cmd := strings.Fields(cmd_line)[0]
		if cmd == "exit" {
			break
		} else {
			fmt.Fprintf(os.Stdout, "%s: not found\n", cmd)
		}
	}
}
