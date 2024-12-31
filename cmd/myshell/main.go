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

func callCmd(cmd string, args []string) (string, string) {
	cmdName, err := find_file(cmd)
	if err != nil {
		return "", fmt.Sprintf("%s: command not found\n", cmd)
	} else {
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
}

func pwdCmd() string {
	path, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
	}
	return path
}

func doCd(arg0 string) error {
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

type commandEnvironment struct {
	Cmd    string
	Args   []string
	Stdout string
}

func parseSingleQuoted(s *string, index *int, currentString *bytes.Buffer) error {
	// fmt.Printf("parseSingleQuote(s: %s, start: %d, currentString: %s)\n", s, start, currentString.String())

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
	// fmt.Printf("parse(s: %s)\n", (*s))
	currentString := bytes.Buffer{}
	escaped := false

	for ; *index < len(*s); (*index)++ {
		if escaped {
			currentString.WriteByte((*s)[*index])
			escaped = false
		} else {
			switch {
			case (*s)[*index] == "'"[0]:
				// fmt.Printf("starting single quote at position %d\n", *index)
				*index++
				err := parseSingleQuoted(s, index, &currentString)
				if err != nil {
					return currentString.String(), err
				}
			case (*s)[*index] == "\""[0]:
				// fmt.Printf("starting double quote at position %d\n", *index)
				*index++
				err := parseDoubleQuoted(s, index, &currentString)
				if err != nil {
					return currentString.String(), err
				}
			case (*s)[*index] == "\\"[0]:
				escaped = true
			case (*s)[*index] == '>':
				currentString.WriteByte((*s)[*index])
				*index++
				return currentString.String(), nil
			case (*s)[*index] == '1':
				nextIndex := *index + 1
				// fmt.Printf("s = %s %q %q\n", *s, (*s)[*index], (*s)[nextIndex])
				// fmt.Printf("currentString = %s\n", currentString.String())

				if nextIndex < len(*s) && (*s)[nextIndex] == '>' {
					if currentString.Len() > 0 {
						return currentString.String(), nil
					} else {
						// previous ended the word and we have 1> at the start of this word
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

func parse(s string) (commandEnvironment, error) {
	// fmt.Printf("parse(s: %s)\n", s)
	fields := []string{}
	// stdout_file := ""
	index := 0
	rval := commandEnvironment{}

	for index < len(s) {
		word, err := parseWord(&s, &index)
		if err != nil {
			return rval, err
		}
		// fmt.Printf("word: %s\n", word)
		if len(word) > 0 {
			if word == ">" {
				out, err := getOut(&s, &index)
				if err != nil {
					return rval, err
				}
				// fmt.Printf("out > %q\n", out)
				rval.Stdout = out
			} else {
				fields = append(fields, word)
			}
		}
	}
	if len(fields) == 0 {
		return rval, nil
	}
	rval.Cmd = fields[0]
	rval.Args = fields[1:]
	// fmt.Printf("rval = %+v\n", rval)
	// fmt.Printf("len([%+v]) == %d\n", fields, len(fields))
	return rval, nil
}

func getCmdEnv(reader *bufio.Reader) (*commandEnvironment, error) {
	cmdLine, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			fmt.Printf("\nexit\n")
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Unrecoverable error: %s\n", err)
		os.Exit(-1)
	}
	cmdEnv := commandEnvironment{}
	if len(cmdLine) == 0 {
		return &cmdEnv, nil
	}

	{
		var err error
		cmdEnv, err = parse(cmdLine)
		if err != nil {
			return &cmdEnv, err
		}
	}
	// fmt.Printf("cmd=%v args=%v\n", cmd, args)
	return &cmdEnv, nil
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
