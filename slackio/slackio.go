// Package slackio provides a reader and writer interface to Slack.
//
// Writing to a Writer emits messages to the configured Slack channel.
// Reading from a Reader receives messages, separated by a newline from
// a slack channel.
//
// Of course, these simple interfaces hide quite a bit of the richness
// of the slack interface, but they are useful none the less for simple
// bots, chatops, and whatnot.
//
// Example:
//
//    slack := NewReaderWriter("", slackToken, "general")
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
	"github.com/crewjam/errset"
)

// Reader reads messages from the specified slack channel. Create
// new instances with NewReader()
type Reader struct {
	common     *readerWriterCommon
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
}

// NewReader returns a new io.Reader that receives messages as they
// are posted to a the specified slack channel. URL is the endpoint
// of the slack server, which can be empty to use the default. token
// is your slack token and channel is the name of the channel to
// receive from.
func NewReader(url, token, channel string) (*Reader, error) {
	common, err := newIface(url, token, channel)
	if err != nil {
		return nil, err
	}
	s := &Reader{
		common: common,
	}
	s.init()
	return s, nil
}

func (s *Reader) init() {
	s.pipeReader, s.pipeWriter = io.Pipe()
	go s.reader()
}

func (s *Reader) Read(p []byte) (n int, err error) {
	return s.pipeReader.Read(p)
}

func (s *Reader) reader() {
	for {
		event := <-s.common.broker.Events()
		if event.Type == "message" {
			msg, err := event.Message()
			if err != nil {
				s.pipeWriter.CloseWithError(err)
			}
			if msg.Channel != s.common.channelID {
				continue
			}
			if msg.User == s.common.selfUserID {
				continue
			}
			_, err = fmt.Fprintf(s.pipeWriter, "%s\n", msg.Text)
			if err != nil {
				s.pipeWriter.CloseWithError(err)
				return
			}
		}
	}
}

// Close disconnects from the slack server.
func (s *Reader) Close() error {
	s.pipeReader.Close()
	s.pipeWriter.Close()
	return s.common.Close()
}

// Writer sends messages to the specified slack channel. Create
// new instances with NewWriter()
type Writer struct {
	common *readerWriterCommon
}

// NewWriter returns a new io.WriteCloser that sends message to the
// the specified slack channel. URL is the endpoint of the slack server,
// which can be empty to use the default. token is your slack token and
// channel is the name of the channel to receive from.
func NewWriter(url, token, channel string) (*Writer, error) {
	common, err := newIface(url, token, channel)
	if err != nil {
		return nil, err
	}
	s := &Writer{
		common: common,
	}
	return s, nil
}

func (s *Writer) Write(p []byte) (n int, err error) {
	err = s.common.broker.Publish(slacker.RTMMessage{
		Type:    "message",
		Text:    string(p),
		Channel: s.common.channelID,
	})
	if err != nil {
		return 0, err
	}
	return len(p), err
}

// Close disconnects from the slack server
func (s *Writer) Close() error {
	return s.common.Close()
}

// ReaderWriter is implements the io.ReadCloser and io.WriteCloser interfaces
// for slack channel. Writing to this object posts messages to the specified
// slack channel. Reading from this object receives messages from the
// specified slack channel. Create new instances with NewReaderWriter()
type ReaderWriter struct {
	common *readerWriterCommon
	Reader
	Writer
}

// NewReaderWriter returns a new io.WriteCloser and io.ReadCloser that sends message
// to and receives messages from the specified slack channel. URL is the
// endpoint of the slack server, which can be empty to use the default. token is
// your slack token and channel is the name of the channel to receive from.
func NewReaderWriter(url, token, channel string) (*ReaderWriter, error) {
	common, err := newIface(url, token, channel)
	if err != nil {
		return nil, err
	}
	s := &ReaderWriter{
		common: common,
		Reader: Reader{
			common: common,
		},
		Writer: Writer{
			common: common,
		},
	}
	s.Reader.init()
	return s, nil
}

// Close disconnects from the slack server
func (s *ReaderWriter) Close() error {
	errs := errset.ErrSet{}
	errs = append(errs, s.Reader.Close())
	errs = append(errs, s.common.Close())
	return errs.ReturnValue()
}

type readerWriterCommon struct {
	channelID  string
	selfUserID string
	client     *slacker.APIClient
	broker     *slacker.RTMBroker
}

func newIface(url, token, channelName string) (*readerWriterCommon, error) {
	s := &readerWriterCommon{}
	s.client = slacker.NewAPIClient(token, url)

	// fetch the channel ID
	s.channelID = ""
	channels, err := s.client.ChannelsList()
	if err != nil {
		return nil, err
	}
	for _, channel := range channels {
		if channel.Name == channelName {
			s.channelID = channel.ID
			break
		}
	}
	if s.channelID == "" {
		return nil, fmt.Errorf("cannot find channel %s", channelName)
	}

	// figure out what user we are connected as so we can ignore
	// previous messages from that user.
	authBuf, err := s.client.RunMethod("auth.test")
	if err != nil {
		return nil, err
	}
	var auth struct {
		UserID string `json:"user_id"`
	}
	json.Unmarshal(authBuf, &auth)

	rtmStart, err := s.client.RTMStart()
	if err != nil {
		return nil, err
	}
	s.selfUserID = auth.UserID

	s.broker = slacker.NewRTMBroker(rtmStart)
	err = s.broker.Connect()
	if err != nil {
		return nil, err
	}

	return s, nil
}

// Close disconnects from the slack server
func (s *readerWriterCommon) Close() error {
	return s.broker.Close()
}

var _ io.ReadCloser = &Reader{}
var _ io.WriteCloser = &Writer{}
