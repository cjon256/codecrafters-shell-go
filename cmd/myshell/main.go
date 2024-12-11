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
	var cmdLine string
	var err error

	for {
		fmt.Fprint(os.Stdout, "$ ")
		// Wait for user input
		cmdLine, err = reader.ReadString('\n')
		if err != nil {
			break
		}
		fields := strings.Fields(cmdLine)
		cmd := fields[0]
		if cmd == "exit" {
			break
		} else if cmd == "echo" {
			echoing := strings.Join(fields[1:], " ")
			fmt.Fprintf(os.Stdout, "%s\n", echoing)
		} else if cmd == "type" {
			typeCmd := fields[1]
			if typeCmd == "exit" || typeCmd == "echo" || typeCmd == "type" {
				fmt.Fprintf(os.Stdout, "%s is a shell builtin\n", typeCmd)
			} else {
				fmt.Fprintf(os.Stdout, "%s: not found\n", cmd)
			}
		} else {
			fmt.Fprintf(os.Stdout, "%s: not found\n", cmd)
		}
	}
}
