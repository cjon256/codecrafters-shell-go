package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var path []string

func find_file(fname string) (string, error) {
	for _, p := range path {
		loc := filepath.Join(p, fname)
		_, e := os.Stat(loc)
		if e == nil {
			return loc, nil
		}
	}
	return "", errors.New("")
}

func typeCmd(arg0 string) string {
	if arg0 == "exit" || arg0 == "echo" || arg0 == "type" {
		return fmt.Sprintf("%s is a shell builtin\n", arg0)
	} else {
		loc, err := find_file(arg0)
		if err == nil {
			return fmt.Sprintf("%s is %s\n", arg0, loc)
		} else {
			return fmt.Sprintf("%s: not found\n", arg0)
		}
	}
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	raw_path := os.Getenv("PATH")
	path = strings.Split(raw_path, ":")

	for {
		fmt.Fprint(os.Stdout, "$ ")
		// Wait for user input
		cmdLine, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		fields := strings.Fields(cmdLine)
		cmd := fields[0]
		args := fields[1:]
		if cmd == "exit" {
			break
		} else if cmd == "echo" {
			echoing := strings.Join(args, " ")
			fmt.Fprintf(os.Stdout, "%s\n", echoing)
		} else if cmd == "type" {
			fmt.Fprint(os.Stdout, typeCmd(args[0]))
		} else {
			cmdName, err := find_file(cmd)
			if err != nil {
				fmt.Fprintf(os.Stdout, "%s: not found\n", cmd)
			} else {
				result := exec.Command(cmdName, args...)
				output, err := result.Output()
				if err != nil {
					// we don't actually do anything different here, but probably should be
					// a separate path
					fmt.Fprintln(os.Stdout, string(output))
				} else {
					fmt.Fprintln(os.Stdout, string(output))
				}
			}
		}
	}
}
