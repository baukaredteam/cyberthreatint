package main

import (
	"os"
	"os/exec"
)

func main() {
	// Запускаем основное приложение мониторинга
	cmd := exec.Command("go", "run", "./cmd/monitor/main.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}