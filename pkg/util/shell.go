package util

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func Shellout(command string, args ...string) (error, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(command, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("shell error: %w: stdout: %s\n stderr: %s", err, stdout.String(), stderr.String()), stdout.String(), stderr.String()
	}
	return nil, stdout.String(), stderr.String()
}

func Touch(file string) error {
	f, err := os.OpenFile(file, os.O_RDONLY|os.O_CREATE, 0o666)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}
	return nil
}
