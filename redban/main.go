package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

func main() {
	stateFile := flag.String("state", "redban-state.json", "where token balance lives")
	perMin := flag.Float64("rpm", 30, "Reddit requests per minute (stay under ~60)")
	burst := flag.Float64("burst", 5, "max burst size")
	tokens := flag.Float64("n", 1, "tokens to take")
	flag.Parse()

	b, err := Open(*stateFile, *perMin, *burst)
	if err != nil {
		log.Fatal(err)
	}

	wait := b.Take(*tokens)
	if wait > 0 {
		fmt.Fprintf(os.Stderr, "rate limited, sleeping %s (%s)\n", wait.Round(time.Millisecond), b)
		time.Sleep(wait)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("ok", b)
		return
	}
	if args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		log.Fatal("nothing to run after --")
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}
