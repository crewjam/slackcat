
## slackcat

This is a simple command that you can use to pipe stdout and stderr to other places.

Examples:

    (for i in 0 1 2 3 4 5 6 7 8 9; do echo $i; sleep 1; done) | slackcat -token=$SLACK_TOKEN

If you provide additional arguments, they are a command to execute with the standard handles redirected to slack. Like, maybe a shell:

    slackcat -token=$SLACK_TOKEN bash -i

Log everything that is said on a channel:

    slackcat -token=$SLACK_TOKEN | tee /var/log/slack.log

Run a command and share the output with your team:

    make deploy ENV=production 2>&1 | slackcat -tee -token=$SLACK_TOKEN -channel=chatops

## slackio

package slackio provides a reader and writer interface to Slack.

 Example:

    slack := NewReaderWriter("", slackToken, "greetings")
    defer slack.Close()
    fmt.Fprintf(slack, "Hello, World!")
    line, _, _ := bufio.NewReader(slack).ReadLine()
    fmt.Fprintf(slack, "You said %q", string(line))
