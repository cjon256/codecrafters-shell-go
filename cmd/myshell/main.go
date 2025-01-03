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

var (
	builtins = make(map[string]cmdFunc)
	path     []string
	cmdLine  string
)

type commandEnviroment struct {
	Cmd    string
	Args   []string
	Stdout string
	Stderr string
}

type cmdFunc func(commandEnviroment) (stdout, stderr string)

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

func typeCmd(cmdargs commandEnviroment) (string, string) {
	stdout, stderr := "", ""
	if len(cmdargs.Args) == 0 {
		return "", ""
	}
	for _, cmd := range cmdargs.Args {
		_, isBuiltin := builtins[cmd]
		if isBuiltin {
			stdout += fmt.Sprintf("%s is a shell builtin\n", cmd)
		} else {
			loc, err := findFile(cmd)
			if err == nil {
				stdout += fmt.Sprintf("%s is %s\n", cmd, loc)
			} else {
				stderr += fmt.Sprintf("%s: not found\n", cmd)
			}
		}
	}
	return stdout, stderr
}

func callCmd(cmdargs commandEnviroment) (string, string) {
	// need path to the actual command
	cmd := cmdargs.Cmd
	args := cmdargs.Args
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

func pwdCmd(_ commandEnviroment) (string, string) {
	errString := ""
	path, err := os.Getwd()
	if err != nil {
		errString = fmt.Sprintln(err)
	}
	outString := path + "\n"
	return outString, errString
}

func capitalizeFirst(str string) string {
	if len(str) == 0 {
		return str
	}
	return strings.ToUpper(str[:1]) + str[1:]
}

func cdCmd(cmdargs commandEnviroment) (string, string) {
	doCd := func(arg0 string) (string, string) {
		err := os.Chdir(arg0)
		if err != nil {
			var pathError *fs.PathError
			if errors.As(err, &pathError) {
				err_message := capitalizeFirst(fmt.Sprintf("%s", pathError.Err))
				return "", fmt.Sprintf("%s: %s\n", arg0, err_message)
			}
			// this is unlikely, Chdir errors are almost always PathError
			return "", fmt.Sprintf("%s\n", err)
		}
		// cd succeeded
		return "", ""
	}

	cdHome := func() (string, string) {
		home := os.Getenv("HOME")
		return doCd(home)
	}

	args := cmdargs.Args
	if len(args) == 0 {
		return cdHome()
	}
	if len(args) > 1 {
		return "", "chdir too many arguments"
	}
	arg0 := args[0]
	if arg0 == "~" {
		return cdHome()
	}
	return doCd(arg0)
}

func parse(s string) (*commandEnviroment, error) {
	parseSingleQuoted := func(s *string, index *int, currentString *bytes.Buffer) error {
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

	parseDoubleQuoted := func(s *string, index *int, currentString *bytes.Buffer) error {
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

	parseWord := func(s *string, index *int) (string, error) {
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

	getOut := func(s *string, index *int) (string, error) {
		out, err := parseWord(s, index)
		if err != nil {
			return "", err
		}
		if len(out) == 0 {
			return "", errors.New("expected a filename after '>'")
		}
		return out, nil
	}

	fields := []string{}
	index := 0
	rval := commandEnviroment{}

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

func getCmdEnv(reader *bufio.Reader) (*commandEnviroment, error) {
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

func exitCmd(_ commandEnviroment) (string, string) {
	os.Exit(0)
	// silly that this needs this, but...
	return "", ""
}

func echoCmd(cmdargs commandEnviroment) (string, string) {
	echoing := strings.Join(cmdargs.Args, " ") + "\n"
	// fmt.Fprintf(stdout, "%s\n", echoing)
	return echoing, ""
}

func noopCmd(_ commandEnviroment) (string, string) {
	return "", ""
}

func getCommand(cmd string) cmdFunc {
	cmdfunc, ok := builtins[cmd]
	if !ok {
		// not a builtin, try calling the command
		return callCmd
	}
	return cmdfunc
}

func main() {
	builtins["type"] = typeCmd
	builtins["exit"] = exitCmd
	builtins["pwd"] = pwdCmd
	builtins["echo"] = echoCmd
	builtins["cd"] = cdCmd
	builtins[""] = noopCmd

	reader := bufio.NewReader(os.Stdin)
	raw_path := os.Getenv("PATH")
	path = strings.Split(raw_path, ":")
	stdout := os.Stdout
	stderr := os.Stderr
	var cmdEnv *commandEnviroment

	for {
		fmt.Fprint(stdout, "$ ")
		// Wait for user input
		var err error
		cmdEnv, err = getCmdEnv(reader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			continue
		}

		if cmdEnv.Stderr != "" {
			var err error
			stderr, err = os.Create(cmdEnv.Stdout)
			if err != nil {
				stderr = os.Stderr
				fmt.Fprintf(stderr, "Err: %s\n", err)
				continue
			}
		}
		if cmdEnv.Stdout != "" {
			var err error
			stdout, err = os.Create(cmdEnv.Stdout)
			if err != nil {
				stdout = os.Stdout
				fmt.Fprintf(stderr, "Err: %s\n", err)
				continue
			}
		}
		outmessage, errmessage := getCommand(cmdEnv.Cmd)(*cmdEnv)
		fmt.Fprint(os.Stderr, errmessage)
		fmt.Fprint(stdout, outmessage)

		stdout = os.Stdout
		stderr = os.Stderr
	}
}
