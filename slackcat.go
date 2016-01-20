package main

import (
	"flag"
	"io"
	"os"
	"os/exec"

	"github.com/crewjam/wx/slackcat/slackio"
)

var SlackToken = flag.String("slack-token", "", "Slack Token")
var SlackURL = flag.String("slack-url", "", "Slack URL (may be empty)")
var Channel = flag.String("channel", "general", "The name of the slack channel")

func main() {
	flag.Parse()

	s, err := slackio.New(*SlackURL, *SlackToken, *Channel)
	if err != nil {
		panic(err)
	}
	defer s.Close()

	// copy from stdin to slack and slack to stdout
	if len(flag.Args()) == 0 {
		go io.Copy(os.Stdout, s)
		io.Copy(s, os.Stdin)
		return
	}

	// execute the program and bind to stdin/stdout
	cmd := exec.Command(flag.Args()[0], flag.Args()[1:]...)
	cmd.Stdout = s
	cmd.Stderr = s
	cmd.Stdin = s
	err = cmd.Run()
	if err != nil {
		panic(err)
	}
}
