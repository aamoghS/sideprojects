package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: movie-ctl apply -f <task.json>")
		os.Exit(1)
	}

	if os.Args[1] == "apply" && len(os.Args) == 4 && os.Args[2] == "-f" {
		filePath := os.Args[3]
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			os.Exit(1)
		}

		resp, err := http.Post("http://127.0.0.1:5555/tasks", "application/json", bytes.NewBuffer(fileContent))
		if err != nil {
			fmt.Printf("Failed to connect to orchestrator API: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusCreated {
			fmt.Printf("Successfully created task: %s\n", string(body))
		} else {
			fmt.Printf("Failed to create task (status %d): %s\n", resp.StatusCode, string(body))
		}
	} else {
		fmt.Println("Unknown command")
	}
}
