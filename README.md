
## slackcat

This is a simple command that you can use to pipe stdout and stderr to other places.

Examples:

    (for i in 0 1 2 3 4 5 6 7 8 9; do echo $i; sleep 1; done) | slacksay -slack-token=$SLACK_TOKEN

If you provide additional arguments, they are a command to execute with the standard handles redirected to slack. Like, maybe a shell:

    slacksay -slack-token=$SLACK_TOKEN bash -i

Log everything that is said on a channel:

    slacksay -slack-token=$SLACK_TOKEN | tee /var/log/slack.log

## slackio

package slackio provides a reader and writer interface to Slack.

 Example:

    slack := New("", slackToken, "general")
    defer slack.Close()
    fmt.Fprintf(slack, "Hello, World!")
    line, _, _ := bufio.NewReader(slack).ReadLine()
    fmt.Fprintf(slack, "You said %q", string(line))
