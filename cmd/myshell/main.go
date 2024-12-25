package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Executor func(*ShellCtx, []string) error
type ShellCtx struct {
	Builtins map[string]Executor
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
	_, found := shellCtx.Builtins[args[0]]
	if found {
		fmt.Printf("%s is a shell builtin\n", args[0])
	} else {
		fmt.Printf("%s: not found\n", args[0])
	}
	return nil
}

func main() {
	var builtins = map[string]Executor{
		"exit": ExitExecutor,
		"echo": EchoExecutor,
		"type": TypeExecutor,
	}
	shellCtx := &ShellCtx{Builtins: builtins}
	for {
		fmt.Fprint(os.Stdout, "$ ")

		// Wait for user input
		commandWithArgs, err := bufio.NewReader(os.Stdin).ReadString('\n')
		commandWithArgs = commandWithArgs[:len(commandWithArgs)-1]
		if err != nil {
			fmt.Printf("Failed to read input: %s\n", err.Error())
			os.Exit(1)
		}

		commandWithArgsParts := strings.Split(commandWithArgs, " ")
		command := commandWithArgsParts[0]

		var args []string
		if len(commandWithArgsParts) > 1 {
			args = commandWithArgsParts[1:]
		} else {
			args = make([]string, 0)
		}

		executor, found := builtins[command]
		if !found {
			fmt.Printf("%s: command not found\n", command)
		} else {
			err = executor(shellCtx, args)
			if err != nil {
				fmt.Printf("Failed execute command %s with args %s: %s\n", command, args, err.Error())
			}
		}
	}
}
