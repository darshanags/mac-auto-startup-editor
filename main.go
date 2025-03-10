package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/manifoldco/promptui"
)

type bootPref struct {
	Value string
	Desc  string
}

func readFwVar(varName string) map[string]string {
	var (
		nvramCMD    *exec.Cmd
		nvramOutput string
		output      bytes.Buffer
		err         error
		regex       *regexp.Regexp
		matches     []string
		valueIndex  int
		val         map[string]string
	)

	val = make(map[string]string)
	val[varName] = ""

	nvramCMD = exec.Command("sudo", "nvram", "-p")

	nvramCMD.Stdout = &output

	if err = nvramCMD.Run(); err != nil {
		log.Fatal("Failed to run nvram command:", err)
	}

	nvramOutput = output.String()

	regex = regexp.MustCompile(`(?m)` + varName + `\b\s+(?P<Value>%\d{2})`)
	matches = regex.FindStringSubmatch(nvramOutput)
	valueIndex = regex.SubexpIndex("Value")

	if len(matches) > 0 {
		val[varName] = matches[valueIndex]
	}

	return val

}

func writeFwVar(fkey string, fval string) error {
	const key string = "BootPreference"
	var nvramCMD *exec.Cmd

	if fkey != key {
		return errors.New("firmware variable mismatch")
	}

	switch fval {
	case "%00", "%01", "%02":
		nvramCMD = exec.Command("sudo", "nvram", fkey+"="+fval)
	case "RESET":
		nvramCMD = exec.Command("sudo", "nvram", "-d", fkey)
	default:
		return errors.New("invalid firmware variable value")
	}

	if err := nvramCMD.Run(); err != nil {
		return err
	}

	return nil
}

func isCompatibleDevice() bool {
	var (
		isCompat, isOSCompat, isDarwin,
		isArm64, isLaptop bool = false, false, false, false, false
		cmd1, cmd2, cmd3, cmd4 *exec.Cmd
		output                 bytes.Buffer
		regexOSVer,
		regexArm,
		regexBat *regexp.Regexp = regexp.MustCompile(`(?m)ProductVersion:\s+([\d]+)`),
			regexp.MustCompile(`ARM64`),
			regexp.MustCompile(`(?m)BatteryInstalled`)
		matches [][]string
	)

	cmd1 = exec.Command("sw_vers")
	cmd2 = exec.Command("uname", "-s")
	cmd3 = exec.Command("uname", "-v")
	cmd4 = exec.Command("ioreg", "-c", "AppleSmartBattery", "-r")

	cmd1.Stdout = &output

	if err := cmd1.Run(); err != nil {
		log.Fatal("Failed to run sw_vers command:", err)
	}

	matches = regexOSVer.FindAllStringSubmatch(output.String(), -1)

	majorVer, err := strconv.ParseInt(matches[0][1], 10, 16)
	if err != nil {
		fmt.Println("Error while getting OS version:", err)
		return isCompat
	}
	if majorVer < 15 {
		return isCompat
	} else {
		isOSCompat = true
	}

	output.Reset()
	cmd2.Stdout = &output

	if err := cmd2.Run(); err != nil {
		log.Fatal("Failed to run uname -s command:", err)
	}

	if str := strings.TrimSpace(output.String()); str == "Darwin" {
		isDarwin = true
	} else {
		return isCompat
	}

	output.Reset()
	cmd3.Stdout = &output

	if err := cmd3.Run(); err != nil {
		log.Fatal("Failed to run uname -v command:", err)
	}

	if str := output.String(); len(regexArm.FindStringIndex(str)) > 0 {
		isArm64 = true
	} else {
		return isCompat
	}

	output.Reset()
	cmd4.Stdout = &output

	if err := cmd4.Run(); err != nil {
		log.Fatal("Failed to run ioreg command:", err)
	}

	if str := output.String(); len(regexBat.FindStringIndex(str)) > 0 {
		isLaptop = true
	}

	if isOSCompat && isDarwin && isArm64 && isLaptop {
		isCompat = true
	}

	return isCompat
}

func main() {
	const key string = "BootPreference"
	var bootPrefs = []bootPref{
		{Value: "%00", Desc: "Prevent startup when opening the lid or connecting to power"},
		{Value: "%01", Desc: "Prevent startup only when opening the lid"},
		{Value: "%02", Desc: "Prevent startup only when connecting to power"},
		{Value: "RESET", Desc: "Reset to default"},
		{Value: "CHECK", Desc: "Check current setting"},
		{Value: "EXIT", Desc: "Exit"},
	}

	var banner string = getBanner()

	fmt.Printf("\033]0;%s\007", "M.A.S.E - Mac Auto Startup Editor")

	fmt.Printf("\033[0;34m%s\033[0m\n", banner)

	fmt.Printf("\033[1;34mWelcome to M.A.S.E - Mac Auto Startup Editor.\033[0m\n\n")

	fmt.Printf("This program utilizes the nvram command to set your auto startup preference - as this command runs under elevated privileges, it will require you to provide your password.\n\n")

	if !isCompatibleDevice() {
		fmt.Printf("\n\033[1;33mYour device is not compatible.\033[0m")
		fmt.Print("\nExiting program...\n")
		return
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}:",
		Active:   "\U0001F536 {{ .Desc | cyan }}",
		Inactive: "  {{ .Desc | cyan }}",
		Selected: "\U00002705 {{ .Desc | green }}",
	}

	prompt := promptui.Select{
		Label:     "Select a startup option",
		Items:     bootPrefs,
		Templates: templates,
	}
PROMPT:
	no, _, err := prompt.Run()

	if err != nil {
		fmt.Printf("Option selection process failed %v\n", err)
		return
	}

	switch bootPrefs[no].Value {
	case "EXIT":
		return
	case "CHECK":
		currentSetting := readFwVar(key)
		if len(currentSetting[key]) > 0 {
			for _, pref := range bootPrefs {
				if pref.Value == currentSetting[key] {
					fmt.Print("\033[1A\033[2K")
					fmt.Printf("\nBootPreference setting is set to\u001B[1;32m %s\033[0m.\n\n", pref.Desc)
				}
			}
		} else {
			fmt.Print("\033[1A\033[2K")
			fmt.Printf("\n\033[1;32mBootPreference setting is currently not set.\033[0m\n\n")
		}
		goto PROMPT
	default:
		err = writeFwVar(key, bootPrefs[no].Value)

		if err == nil {
			if bootPrefs[no].Value != "RESET" {
				fmt.Print("\033[1A\033[2K")
				fmt.Printf("BootPreference setting was set to\u001B[1;32m %s\033[0m.\n", bootPrefs[no].Desc)
			} else {
				fmt.Print("\033[1A\033[2K")
				fmt.Printf("BootPreference setting was\u001B[1;32m reset\033[0m.\n")
			}
			goto PROMPT
		} else {
			log.Fatal("An error occured while setting your BootPreference setting: ", err)
		}
	}
}
