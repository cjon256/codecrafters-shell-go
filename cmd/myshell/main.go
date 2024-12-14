package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func find_file(fname string, paths []string) (string, error) {
	for _, p := range paths {
		loc := filepath.Join(p, fname)
		_, e := os.Stat(loc)
		if e == nil {
			return loc, nil
		}
	}
	return "", errors.New("")
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	var cmdLine string
	var err error
	raw_path := os.Getenv("PATH")
	paths := strings.Split(raw_path, ":")

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
				loc, err := find_file(fields[1], paths)
				if err == nil {
					fmt.Fprintf(os.Stdout, "%s is %s\n", typeCmd, loc)
				} else {
					fmt.Fprintf(os.Stdout, "%s: not found\n", typeCmd)
				}
			}
		} else {
			fmt.Fprintf(os.Stdout, "%s: not found\n", cmd)
		}
	}
}
