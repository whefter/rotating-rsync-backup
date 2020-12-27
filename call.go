package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
)

func sshCall(options *Options, sshCmd string, logger *log.Logger) ([]string, []string, int, error) {
	args := []string{}

	args = append(args, options.SSHOptions()...)
	args = append(args, options.targetHost)
	args = append(args, sshCmd)

	return call("ssh", args, "ssh", logger)
}

func call(command string, args []string, logLabel string, logger *log.Logger) ([]string, []string, int, error) {
	if logLabel == "" {
		logLabel = "exec"
	}

	Log.Debug.Printf("call: Full command line: %s %v", command, args)

	cmd := exec.Command(command, args...)

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

	go handleCallStream("stdout", logLabel, stdout, &fullStdout, logger)
	go handleCallStream("stderr", logLabel, stderr, &fullStderr, logger)

	err = cmd.Wait()

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}

	Log.Debug.Printf("call: Command finished with error: %v", err)
	// if err != nil {
	// 	panic(fmt.Sprintf("sshCall: returned with error %v", err))
	// }

	// Log.Debug.Println("call: All stdout lines", fullStdout)
	// Log.Debug.Println("call: All stderr lines", fullStderr)

	return fullStdout, fullStderr, exitCode, err
}

func handleCallStream(streamName string, logLabel string, stdout io.ReadCloser, stash *[]string, logger *log.Logger) {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		*stash = append(*stash, line)

		logLine := fmt.Sprintf("[ %s %s ] %s", logLabel, streamName, line)
		logger.Println(logLine)
	}
}
