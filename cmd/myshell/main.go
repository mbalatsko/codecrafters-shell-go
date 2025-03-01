package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type Executor func(*ShellCtx, []string) error
type ShellCtx struct {
	Builtins    map[string]Executor
	PathFolders []string
	CurrentDir  string
	Serr        string
	Sout        string
}

func (ctx *ShellCtx) Reset() {
	ctx.Serr = ""
	ctx.Sout = ""
}

func IsExecAny(mode os.FileMode) bool {
	return mode&0111 != 0
}

func SearchExecInPathFolders(command string, pathFolders []string) (string, bool) {
	for _, folder := range pathFolders {
		files, err := os.ReadDir(folder)
		if err != nil {
			continue
		}

		for _, file := range files {
			fileInfo, err := file.Info()
			if err != nil {
				continue
			}

			if IsExecAny(fileInfo.Mode()) && file.Name() == command {
				return filepath.Join(folder, file.Name()), true
			}
		}
	}
	return "", false
}

func ExitExecutor(_ *ShellCtx, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("exit command takes exactly 1 argument of type int")
	}
	code, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("exit command failed to parse exit code: %s", err.Error())
	}
	os.Exit(code)
	return nil
}

func EchoExecutor(shellCtx *ShellCtx, args []string) error {
	message := strings.Join(args, " ")
	shellCtx.Sout = message + "\n"
	return nil
}

func TypeExecutor(shellCtx *ShellCtx, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("exit command takes exactly 1 argument of type string")
	}
	command := args[0]
	_, found := shellCtx.Builtins[command]
	if found {
		shellCtx.Sout = fmt.Sprintf("%s is a shell builtin\n", command)
	} else {
		execPath, found := SearchExecInPathFolders(command, shellCtx.PathFolders)

		if found {
			shellCtx.Sout = fmt.Sprintf("%s is %s\n", command, execPath)
		} else {
			shellCtx.Serr = fmt.Sprintf("%s: not found\n", command)
		}
	}
	return nil
}

func PwdExecutor(shellCtx *ShellCtx, _ []string) error {
	shellCtx.Sout = fmt.Sprintln(shellCtx.CurrentDir)
	return nil
}

func ChangeDirExecutor(shellCtx *ShellCtx, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("cd command takes exactly 1 argument of type string")
	}

	destPath := args[0]
	if destPath[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		destPath = strings.Replace(destPath, "~", homeDir, 1)
	} else if destPath[0] == '.' {
		destPath = filepath.Join(shellCtx.CurrentDir, destPath)
	}

	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		shellCtx.Serr = fmt.Sprintf("cd: %s: No such file or directory\n", destPath)
	} else {
		shellCtx.CurrentDir = destPath
	}
	return nil
}

func RunExternalCommand(command string, args []string, shellCtx *ShellCtx) error {
	cmd := exec.Command(command, args...)
	output, err := cmd.Output()
	if err != nil {
		serr, ok := err.(*exec.ExitError)
		if ok {
			shellCtx.Serr = string(serr.Stderr)
		} else {
			return err
		}
	}
	shellCtx.Sout = string(output)
	return nil
}

func ParseArgs(input string) []string {
	input = strings.TrimSpace(input)
	args := []string{}
	doubleQuotedSpecialCharacters := []rune{'$', '\\', '"'}
	const (
		isSingleQouted = iota
		isDoubleQouted
		isEscaped
	)
	currentState := isEscaped
	skipNext := false
	buffer := ""
	for i, arg := range input {
		if skipNext {
			skipNext = false
			continue
		}
		if i == 0 {
			if arg == '"' {
				currentState = isDoubleQouted
				continue
			} else if arg == '\'' {
				currentState = isSingleQouted
				continue
			}
		}
		switch arg {
		case '"':
			if currentState == isEscaped {
				currentState = isDoubleQouted
				buffer += string(input[i+1])
				skipNext = true
			} else if currentState == isDoubleQouted {
				currentState = isEscaped
			} else {
				buffer += string(arg)
			}
		case '\'':
			if currentState == isEscaped {
				currentState = isSingleQouted
				buffer += string(input[i+1])
				skipNext = true
			} else if currentState == isSingleQouted {
				currentState = isEscaped
			} else {
				buffer += string(arg)
			}
		case '\\':
			if currentState == isEscaped {
				buffer += string(input[i+1])
				skipNext = true
			} else if currentState == isDoubleQouted {
				contains := slices.Contains(doubleQuotedSpecialCharacters, rune(input[i+1]))
				if contains {
					buffer += string(input[i+1])
					skipNext = true
				} else {
					buffer += string(arg)
				}
			} else if currentState == isSingleQouted {
				buffer += string(arg)
			}
		case ' ':
			if currentState == isEscaped {
				args = append(args, buffer)
				buffer = ""
			} else {
				buffer += string(arg)
			}
		default:
			buffer += string(arg)
		}
	}
	if len(buffer) > 0 {
		args = append(args, buffer)
	}
	res := []string{}
	for _, arg := range args {
		if len(arg) > 0 {
			res = append(res, arg)
		}
	}

	return res
}

func main() {
	var builtins = map[string]Executor{
		"exit": ExitExecutor,
		"echo": EchoExecutor,
		"type": TypeExecutor,
		"pwd":  PwdExecutor,
		"cd":   ChangeDirExecutor,
	}

	var pathFolders []string
	path := os.Getenv("PATH")
	if len(path) > 0 {
		pathFolders = strings.Split(path, ":")
	} else {
		pathFolders = make([]string, 0)
	}

	currentDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	shellCtx := &ShellCtx{Builtins: builtins, PathFolders: pathFolders, CurrentDir: currentDir}
	for {
		shellCtx.Serr = ""
		shellCtx.Sout = ""

		fmt.Fprint(os.Stdout, "$ ")

		// Wait for user input
		commandWithArgs, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Printf("Failed to read input: %s\n", err.Error())
			os.Exit(1)
		}
		commandWithArgs = commandWithArgs[:len(commandWithArgs)-1]
		parsedCommand := ParseArgs(commandWithArgs)

		if len(parsedCommand) == 0 {
			continue
		}

		args := make([]string, 0)
		command := parsedCommand[0]

		sOut := os.Stdout
		sErr := os.Stderr

		if len(parsedCommand) > 0 {
			args = parsedCommand[1:]

			cutIdx := -1
			for i := range args {
				if args[i] == ">" || args[i] == "1>" {
					sOut, err = os.OpenFile(args[i+1], os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
					if err != nil {
						panic(err)
					}
					if cutIdx == -1 {
						cutIdx = i
					}
				} else if args[i] == ">>" || args[i] == "1>>" {
					sOut, err = os.OpenFile(args[i+1], os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
					if err != nil {
						panic(err)
					}
					if cutIdx == -1 {
						cutIdx = i
					}
				} else if args[i] == "2>" {
					sErr, err = os.OpenFile(args[i+1], os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
					if err != nil {
						panic(err)
					}
					if cutIdx == -1 {
						cutIdx = i
					}
				} else if args[i] == "2>>" {
					sErr, err = os.OpenFile(args[i+1], os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
					if err != nil {
						panic(err)
					}
					if cutIdx == -1 {
						cutIdx = i
					}
				}
			}

			if cutIdx != -1 {
				args = args[:cutIdx]
			}
		}

		executor, found := shellCtx.Builtins[command]
		if found {
			err = executor(shellCtx, args)
			if err != nil {
				fmt.Printf("Failed execute command %s with args %s: %s\n", command, args, err.Error())
			}
		} else {
			execPath, found := SearchExecInPathFolders(command, shellCtx.PathFolders)
			if found {
				err := RunExternalCommand(execPath, args, shellCtx)
				if err != nil {
					fmt.Printf("Failed execute external command %s with args %s: %s\n", execPath, args, err.Error())
				}
			} else {
				fmt.Printf("%s: command not found\n", command)
			}
		}

		if _, err := io.Copy(sOut, strings.NewReader(shellCtx.Sout)); err != nil {
			fmt.Printf("Failed to copy to stdout: %s", err.Error())
		}

		if _, err := io.Copy(sErr, strings.NewReader(shellCtx.Serr)); err != nil {
			fmt.Printf("Failed to copy to stderr: %s", err.Error())
		}

		if sOut != os.Stdout {
			sOut.Close()
		}

		if sErr != os.Stderr {
			sErr.Close()
		}
	}
}
