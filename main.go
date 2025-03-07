package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"syscall"
)

func main() {

	var (
		nvramCMD, grepCMD *exec.Cmd
		output            bytes.Buffer
		err               error
	)

	nvramCMD = exec.Command("sudo", "nvram", "-p")

	grepCMD = exec.Command("grep", "BootPreference")

	grepCMD.Stdin, _ = nvramCMD.StdoutPipe()
	grepCMD.Stdout = &output

	// Start the grep command first
	if err = grepCMD.Start(); err != nil {
		log.Fatal("Failed to start grep command:", err)
	}

	// Start the ls command
	if err = nvramCMD.Run(); err != nil {
		log.Fatal("Failed to run ls command:", err)
	}

	// Wait for grep to finish
	err = grepCMD.Wait()
	if err != nil {
		exitErr, isExitError := err.(*exec.ExitError)

		if isExitError {
			status, isWaitStatus := exitErr.Sys().(syscall.WaitStatus)
			if isWaitStatus {
				switch status.ExitStatus() {
				case 1:
					fmt.Println("No matches found.")
				case 2:
					log.Fatal("grep encountered an error:", err)
				default:
					log.Fatal("grep exited with unknown status:", status.ExitStatus())
				}
			}
		} else {
			log.Fatal("Failed to wait for grep command:", err)
		}
	}

	// Print the filtered output
	fmt.Println("Filtered output:")
	fmt.Println(output.String())
}
