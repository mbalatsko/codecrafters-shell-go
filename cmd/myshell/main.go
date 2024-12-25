package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	fmt.Fprint(os.Stdout, "$ ")

	// Wait for user input
	commandWithArgs, err := bufio.NewReader(os.Stdin).ReadString('\n')
	commandWithArgs = commandWithArgs[:len(commandWithArgs) - 1]
	if err != nil {
		fmt.Printf("Failed to read input: %s", err.Error())
		os.Exit(1)
	}

	command := strings.Split(commandWithArgs, " ")[0]
	fmt.Printf("%s: command not found", command)
}
