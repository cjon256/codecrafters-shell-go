package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
)

var path []string

func findFile(fname string) (string, error) {
	for _, p := range path {
		loc := filepath.Join(p, fname)
		_, err := os.Stat(loc)
		if err == nil {
			return loc, nil
		}
	}
	return "", errors.New("")
}

func typeCmd(cmd string) string {
	if cmd == "exit" || cmd == "echo" || cmd == "type" || cmd == "pwd" || cmd == "cd" {
		return fmt.Sprintf("%s is a shell builtin\n", cmd)
	} else {
		loc, err := findFile(cmd)
		if err == nil {
			return fmt.Sprintf("%s is %s\n", cmd, loc)
		} else {
			return fmt.Sprintf("%s: not found\n", cmd)
		}
	}
}

func callCmd(cmd string, args []string) (string, string) {
	// need path to the actual command
	cmdName, err := findFile(cmd)
	if err != nil {
		// command not found
		return "", fmt.Sprintf("%s: command not found\n", cmd)
	}
	execCmd := exec.Command(cmdName, args...)
	var outStr, errStr bytes.Buffer
	execCmd.Stdout = &outStr
	execCmd.Stderr = &errStr
	err = execCmd.Run()
	if err != nil {
		// fmt.Println("Error of some kind in callCmd(): ", err)
		// this is where we would set the error status I assume?
	}
	return outStr.String(), errStr.String()
}

func pwdCmd() string {
	// TODO maybe doesn't need a separate function?
	path, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stdout, err)
	}
	return path
}

func capitalizeFirst(str string) string {
	if len(str) == 0 {
		return str
	}
	return strings.ToUpper(str[:1]) + str[1:]
}

func doCd(arg0 string) error {
	err := os.Chdir(arg0)
	if err != nil {
		var pathError *fs.PathError
		if errors.As(err, &pathError) {
			err_message := capitalizeFirst(fmt.Sprintf("%s", pathError.Err))
			return fmt.Errorf("%s: %s", arg0, err_message)
		}
		// this is unlikely, Chdir errors are almost always PathError
		return err
	}
	// cd succeeded
	return nil
}

func cdHome() error {
	home := os.Getenv("HOME")
	return doCd(home)
}

func cdCmd(args []string) error {
	if len(args) == 0 {
		return cdHome()
	}
	if len(args) > 1 {
		return errors.New("chdir too many arguments")
	}
	arg0 := args[0]
	if arg0 == "~" {
		return cdHome()
	}
	return doCd(arg0)
}

func parseSingleQuoted(s *string, index *int, currentString *bytes.Buffer) error {
	for ; *index < len(*s); (*index)++ {
		switch (*s)[(*index)] {
		case '\'':
			// fmt.Printf("found a single quote at index %d\n", *index)
			return nil
		default:
			// fmt.Printf("adding byte %s\n", string((*s)[*index]))
			currentString.WriteByte((*s)[*index])
		}
	}
	return errors.New("unclosed single quote")
}

func parseDoubleQuoted(s *string, index *int, currentString *bytes.Buffer) error {
	escaped := false

	for ; *index < len(*s); *index++ {
		if escaped {
			switch (*s)[*index] {
			case '"':
				currentString.WriteByte((*s)[*index])
				escaped = false
			case '\\':
				currentString.WriteByte((*s)[*index])
				escaped = false
			default:
				currentString.WriteByte("\\"[0])
				currentString.WriteByte((*s)[*index])
				escaped = false
			}
		} else {
			switch (*s)[*index] {
			case '"':
				// fmt.Printf("ending double quote at position %d\n", *index)
				return nil
			case '\\':
				escaped = true
			default:
				currentString.WriteByte((*s)[*index])
				escaped = false
			}
		}
	}
	return errors.New("unclosed double quote")
}

func parseWord(s *string, index *int) (string, error) {
	currentString := bytes.Buffer{}
	escaped := false

	for ; *index < len(*s); (*index)++ {
		if escaped {
			currentString.WriteByte((*s)[*index])
			escaped = false
		} else {
			switch {
			case (*s)[*index] == "'"[0]:
				*index++
				err := parseSingleQuoted(s, index, &currentString)
				if err != nil {
					return currentString.String(), err
				}
			case (*s)[*index] == "\""[0]:
				*index++
				err := parseDoubleQuoted(s, index, &currentString)
				if err != nil {
					return currentString.String(), err
				}
			case (*s)[*index] == "\\"[0]:
				escaped = true
			case (*s)[*index] == '>':
				if currentString.Len() > 0 {
					// we have a word already, so stay on this character and return
					return currentString.String(), nil
				} else {
					// we are not in a word, so return '>'
					currentString.WriteByte((*s)[*index])
					*index++
					return currentString.String(), nil
				}
			case (*s)[*index] == '1':
				nextIndex := *index + 1
				if nextIndex < len(*s) && (*s)[nextIndex] == '>' {
					// so next two characters are '1>'
					if currentString.Len() > 0 {
						// we have a word already, so stay on this character and return
						return currentString.String(), nil
					} else {
						// we are not in a word, so return '>'
						*index++
						currentString.WriteByte((*s)[*index])
						*index++
						return currentString.String(), nil
					}
				} else {
					currentString.WriteByte((*s)[*index])
					escaped = false
				}
			case unicode.IsSpace(rune((*s)[*index])):
				if currentString.Len() > 0 {
					return currentString.String(), nil
				}
			default:
				currentString.WriteByte((*s)[*index])
				escaped = false
			}
		}
	}
	return currentString.String(), nil
}

func getOut(s *string, index *int) (string, error) {
	out, err := parseWord(s, index)
	if err != nil {
		return "", err
	}
	if len(out) == 0 {
		return "", errors.New("expected a filename after '>'")
	}
	return out, nil
}

type commandEnvironment struct {
	Cmd    string
	Args   []string
	Stdout string
}

func parse(s string) (*commandEnvironment, error) {
	fields := []string{}
	index := 0
	rval := commandEnvironment{}

	for index < len(s) {
		word, err := parseWord(&s, &index)
		if err != nil {
			return &rval, err
		}
		if len(word) > 0 {
			if word == ">" {
				out, err := getOut(&s, &index)
				if err != nil {
					return &rval, err
				}
				rval.Stdout = out
			} else {
				fields = append(fields, word)
			}
		}
	}
	if len(fields) == 0 {
		return &rval, nil
	}
	rval.Cmd = fields[0]
	rval.Args = fields[1:]
	return &rval, nil
}

func getCmdEnv(reader *bufio.Reader) (*commandEnvironment, error) {
	// get input from the user
	cmdLine, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			// exit nicely on a ^D
			fmt.Printf("\nexit\n")
			os.Exit(0)
		}
		// it was something else, exit with error
		fmt.Fprintf(os.Stderr, "Unrecoverable error: %s\n", err)
		os.Exit(-1)
	}
	return parse(cmdLine)
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	raw_path := os.Getenv("PATH")
	path = strings.Split(raw_path, ":")
	stdout := os.Stdout
	var cmdEnv *commandEnvironment

	for {
		fmt.Fprint(stdout, "$ ")
		// Wait for user input
		var err error
		cmdEnv, err = getCmdEnv(reader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			continue
		}

		if cmdEnv.Stdout != "" {
			var err error
			stdout, err = os.Create(cmdEnv.Stdout)
			if err != nil {
				stdout = os.Stdout
				fmt.Fprintf(os.Stderr, "Err: %s\n", err)
				continue
			}
		}
		switch cmdEnv.Cmd {
		case "":
			stdout = os.Stdout
			continue
		case "exit":
			return
		case "echo":
			echoing := strings.Join(cmdEnv.Args, " ")
			fmt.Fprintf(stdout, "%s\n", echoing)
		case "type":
			fmt.Fprint(stdout, typeCmd(cmdEnv.Args[0]))
		case "pwd":
			fmt.Fprintln(stdout, pwdCmd())
		case "cd":
			err := cdCmd(cmdEnv.Args)
			if err != nil {
				fmt.Fprintf(os.Stderr, "cd: %s\n", err)
			}
		default:
			outmessage, errmessage := callCmd(cmdEnv.Cmd, cmdEnv.Args)
			if errmessage == "" {
				fmt.Fprint(stdout, outmessage)
			} else {
				fmt.Fprint(os.Stderr, errmessage)
				fmt.Fprint(stdout, outmessage)
			}
		}

		stdout = os.Stdout
	}
}
