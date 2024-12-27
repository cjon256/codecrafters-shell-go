package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
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
	if arg0 == "exit" || arg0 == "echo" || arg0 == "type" || arg0 == "pwd" || arg0 == "cd" {
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

func callCmd(cmd string, args []string) string {
	cmdName, err := find_file(cmd)
	if err != nil {
		return fmt.Sprintf("%s: command not found\n", cmd)
	} else {
		result := exec.Command(cmdName, args...)
		output, err := result.Output()
		if err != nil {
			// we don't actually do anything different here, but probably should be
			// a separate path
			return string(output)
		} else {
			return string(output)
		}
	}
}

func pwdCmd() string {
	path, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
	}
	return path
}

func cdCmd(args []string) error {
	if len(args) == 0 {
		// not handling this yet
		return nil
	}
	if len(args) > 1 {
		return errors.New("chdir too many arguments")
	}
	arg0 := args[0]
	err := os.Chdir(arg0)
	if err != nil {
		var pathError *fs.PathError
		if errors.As(err, &pathError) {
			capitalizeFirst := func(str string) string {
				if len(str) == 0 {
					return str
				}
				return strings.ToUpper(str[:1]) + str[1:]
			}

			err_message := capitalizeFirst(fmt.Sprintf("%s", pathError.Err))
			return fmt.Errorf("%s: %s", arg0, err_message)
		} else {
			return err
		}
	}
	// check to see if arg0 is a directory that exists
	// "%s: No such file or directory"
	// cd: README.md: Not a directory
	// ok, error situations handled... now actually cd
	return nil
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
			// should probably do more to handle, or print an error message?
			break
		}

		fields := strings.Fields(cmdLine)
		cmd := fields[0]
		args := fields[1:]

		switch cmd {
		case "exit":
			return
		case "echo":
			echoing := strings.Join(args, " ")
			fmt.Fprintf(os.Stdout, "%s\n", echoing)
		case "type":
			fmt.Fprint(os.Stdout, typeCmd(args[0]))
		case "pwd":
			fmt.Fprintln(os.Stdout, pwdCmd())
		case "cd":
			err := cdCmd(args)
			if err != nil {
				fmt.Printf("cd: %s\n", err)
			}
		default:
			fmt.Fprint(os.Stdout, callCmd(cmd, args))
		}
	}
}
