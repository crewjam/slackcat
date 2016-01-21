package main

import (
	"flag"
	"io"
	"os"
	"os/exec"

	"github.com/crewjam/slackcat/slackio"
)

func main() {
	token := flag.String("token", "", "Slack Token")
	endpoint := flag.String("endpoint", "", "Slack URL (may be empty)")
	channel := flag.String("channel", "general", "The name of the slack channel")
	doTee := flag.Bool("tee", false, "tee stdin to both stdout and slack")
	doRead := flag.Bool("read", false, "only read from slack, don't also write. Default is to both read and write.")
	doWrite := flag.Bool("write", false, "only write to slack, don't also read. Default is to both read and write.")

	flag.Parse()
	if !*doRead && !*doWrite {
		*doRead = true
		*doWrite = true
	}
	if *doTee {
		*doWrite = true
		*doRead = false
	}

	s, err := slackio.New(*SlackURL, *SlackToken, *Channel)
	if err != nil {
		panic(err)
	}
	defer s.Close()

	var stdin io.Reader = os.Stdin
	if *doTee {
		stdin = io.TeeReader(os.Stdin, os.Stdout)
	}

	// copy from stdin to slack and slack to stdout
	if len(flag.Args()) == 0 {
		if *doWrite && *doRead {
			go io.Copy(os.Stdout, s)
			io.Copy(s, stdin)
		} else if *doWrite {
			io.Copy(s, stdin)
		} else if *doRead {
			io.Copy(os.Stdout, s)
		}
		return
	}

	// execute the program and bind to stdin/stdout
	cmd := exec.Command(flag.Args()[0], flag.Args()[1:]...)
	if *doWrite {
		cmd.Stdout = s
		cmd.Stderr = s
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if *doRead {
		cmd.Stdin = s
	} else {
		cmd.Stdin = stdin
	}

	err = cmd.Run()
	if err != nil {
		panic(err)
	}
}
