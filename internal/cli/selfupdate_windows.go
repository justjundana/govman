//go:build windows

package cli

import (
	"fmt"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func startWindowsUpdateHelper(helperPath string, arguments []string) error {
	powerShell, err := exec.LookPath("powershell.exe")
	if err != nil {
		powerShell, err = exec.LookPath("pwsh.exe")
		if err != nil {
			return fmt.Errorf("PowerShell is required to complete self-update: %w", err)
		}
	}
	commandArguments := []string{"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", helperPath}
	commandArguments = append(commandArguments, arguments...)
	command := exec.Command(powerShell, commandArguments...)
	command.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
		HideWindow:    true,
	}
	if err := command.Start(); err != nil {
		return err
	}
	return command.Process.Release()
}
