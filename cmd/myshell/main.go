package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/shlex"
)

type Executor func(*ShellCtx, []string) error
type ShellCtx struct {
	Builtins    map[string]Executor
	PathFolders []string
	CurrentDir  string
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

func EchoExecutor(_ *ShellCtx, args []string) error {
	message := strings.Join(args, " ")
	fmt.Println(message)
	return nil
}

func TypeExecutor(shellCtx *ShellCtx, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("exit command takes exactly 1 argument of type string")
	}
	command := args[0]
	_, found := shellCtx.Builtins[command]
	if found {
		fmt.Printf("%s is a shell builtin\n", command)
	} else {
		execPath, found := SearchExecInPathFolders(command, shellCtx.PathFolders)

		if found {
			fmt.Printf("%s is %s\n", command, execPath)
		} else {
			fmt.Printf("%s: not found\n", command)
		}
	}
	return nil
}

func PwdExecutor(shellCtx *ShellCtx, _ []string) error {
	fmt.Println(shellCtx.CurrentDir)
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
		fmt.Printf("cd: %s: No such file or directory\n", destPath)
	} else {
		shellCtx.CurrentDir = destPath
	}
	return nil
}

func RunExternalCommand(command string, args []string) error {
	cmd := exec.Command(command, args...)
	output, err := cmd.Output()
	if err != nil {
		serr, ok := err.(*exec.ExitError)
		if ok {
			output = serr.Stderr
		} else {
			return err
		}
	}
	fmt.Print(string(output))
	return nil
}

func ParseInput(input string) ([]string, error) {
	return shlex.Split(input)
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
		fmt.Fprint(os.Stdout, "$ ")

		// Wait for user input
		commandWithArgs, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Printf("Failed to read input: %s\n", err.Error())
			os.Exit(1)
		}
		commandWithArgs = commandWithArgs[:len(commandWithArgs)-1]
		parsedCommand, err := ParseInput(commandWithArgs)
		if err != nil {
			fmt.Printf("Failed to parse input: %s\n", err.Error())
			os.Exit(1)
		}

		args := make([]string, 0)
		command := parsedCommand[0]
		if len(parsedCommand) > 0 {
			args = parsedCommand[1:]
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
				err := RunExternalCommand(execPath, args)
				if err != nil {
					fmt.Printf("Failed execute external command %s with args %s: %s\n", execPath, args, err.Error())
				}
			} else {
				fmt.Printf("%s: command not found\n", command)
			}
		}
	}
}
