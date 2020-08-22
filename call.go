package main

import (
	"fmt"
	"os/exec"
)

func sshCall(options *Options, sshCmd string) ([]string, []string, error) {
	args := []string{}

	args = append(args, options.SSHOptions()...)
	args = append(args, options.targetHost)
	args = append(args, sshCmd)

	return call("ssh", args)
}

func call(command string, args []string) ([]string, []string, error) {
	Log.Debug.Printf("call: Full command line: %s %v", command, args)

	cmd := exec.Command(command, args...)
	Log.Debug.Printf("test? %s %v", command, cmd.Args)

	// newCommand := "bash"
	// newArgs := []string{"-c", shellescape.Quote(command + " " + strings.Join(args, " "))}
	// Log.Debug.Printf("FOOOOO %s %s", newCommand, strings.Join(newArgs, " "))
	// cmd := exec.Command(newCommand, newArgs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(fmt.Sprintf("call: Could not get StdoutPipe: %v", err))
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(fmt.Sprintf("call: Could not get Stderr: %v", err))
	}

	if err := cmd.Start(); err != nil {
		panic(fmt.Sprintf("call: could not Start() cmd: %v", err))
	}

	var fullStdout []string
	var fullStderr []string

	go printStdout(stdout, &fullStdout)
	go printStderr(stderr, &fullStderr)

	err = cmd.Wait()

	Log.Debug.Printf("call: Command finished with error: %v", err)
	// if err != nil {
	// 	panic(fmt.Sprintf("sshCall: returned with error %v", err))
	// }

	Log.Debug.Println("call: All stdout lines", fullStdout)
	Log.Debug.Println("call: All stderr lines", fullStderr)

	return fullStdout, fullStderr, err
}
