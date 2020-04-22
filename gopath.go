package main

import (
	"os/exec"
	"strings"
)

func getGOPATH() (string, error) {
	cmd := exec.Command("go", "env", "GOPATH")
	data, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
