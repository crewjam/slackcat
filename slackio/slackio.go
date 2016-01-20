// package slackio provides a reader and writer interface to Slack.
//
// Example:
//
//    slack := New("", slackToken, "general")
//    defer slack.Close()
//    fmt.Fprintf(slack, "Hello, World!")
//    line, _, _ := bufio.NewReader(slack).ReadLine()
//    fmt.Fprintf(slack, "You said %q", string(line))
//
package slackio

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/bobbytables/slacker"
)

// Slack is implements the io.ReadCloser and io.WriteCloser interfaces
// for slack channel. Writing to this object posts messages to the specified
// slack channel. Reading from this object receives messages from the
// specified slack channel.
type Slack struct {
	URL     string // the URL of the slack endpoint (empty string for the default)
	Token   string // the slack token
	Channel string // the name of the channel to write in / read from.

	channelID  string
	selfUserID string
	client     *slacker.APIClient
	broker     *slacker.RTMBroker

	readPipeReader  *io.PipeReader
	readPipeWriter  *io.PipeWriter
	writePipeReader *io.PipeReader
	writePipeWriter *io.PipeWriter
}

// New returns a new Slack
func New(url, token, channel string) (*Slack, error) {
	s := Slack{
		URL:     url,
		Token:   token,
		Channel: channel,
	}
	if err := s.init(); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *Slack) init() error {
	if s.broker != nil {
		return nil
	}

	s.client = slacker.NewAPIClient(s.Token, s.URL)

	// fetch the channel ID
	s.channelID = ""
	channels, err := s.client.ChannelsList()
	if err != nil {
		return err
	}
	for _, channel := range channels {
		if channel.Name == s.Channel {
			s.channelID = channel.ID
			break
		}
	}
	if s.channelID == "" {
		return fmt.Errorf("cannot find channel %s", s.Channel)
	}

	// figure out what user we are connected as so we can ignore
	// previous messages from that user.
	authBuf, err := s.client.RunMethod("auth.test")
	if err != nil {
		panic(err)
	}
	var auth struct {
		UserID string `json:"user_id"`
	}
	json.Unmarshal(authBuf, &auth)

	rtmStart, err := s.client.RTMStart()
	if err != nil {
		return err
	}
	s.selfUserID = auth.UserID

	s.broker = slacker.NewRTMBroker(rtmStart)
	err = s.broker.Connect()
	if err != nil {
		return err
	}

	return nil
}

func (s *Slack) initWrite() error {
	if err := s.init(); err != nil {
		return err
	}
	return nil
}

func (s *Slack) initRead() error {
	if err := s.init(); err != nil {
		return err
	}
	if s.readPipeWriter == nil {
		s.readPipeReader, s.readPipeWriter = io.Pipe()
		go s.reader()
	}
	return nil
}

func (s *Slack) Write(p []byte) (n int, err error) {
	if err := s.initWrite(); err != nil {
		return 0, err
	}
	err = s.broker.Publish(slacker.RTMMessage{
		Type:    "message",
		Text:    string(p),
		Channel: s.channelID,
	})
	if err != nil {
		return 0, err
	}
	return len(p), err
}

func (s *Slack) Read(p []byte) (n int, err error) {
	if err := s.initRead(); err != nil {
		return 0, err
	}
	n, err = s.readPipeReader.Read(p)
	return n, err
}

func (s *Slack) reader() {
	for {
		event := <-s.broker.Events()
		if event.Type == "message" {
			msg, err := event.Message()
			if err != nil {
				s.readPipeWriter.CloseWithError(err)
			}
			if msg.Channel != s.channelID {
				continue
			}
			if msg.User == s.selfUserID {
				continue
			}
			_, err = fmt.Fprintf(s.readPipeWriter, "%s\n", msg.Text)
			if err != nil {
				s.readPipeWriter.CloseWithError(err)
				return
			}
		}
	}
}

func (s *Slack) Close() error {
	if s.readPipeReader != nil {
		s.readPipeReader.Close()
	}
	if s.readPipeWriter != nil {
		s.readPipeWriter.Close()
	}
	if s.broker != nil {
		s.broker.Close()
	}
	return nil
}

var _ io.ReadCloser = &Slack{}
var _ io.WriteCloser = &Slack{}
