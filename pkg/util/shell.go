package util

import (
	"bytes"
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
	return err, stdout.String(), stderr.String()
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
