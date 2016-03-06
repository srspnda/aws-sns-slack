package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	Name    = "aws-sns-slack"
	Version = "0.1.0"
)

var (
	httpAddr       = flag.String("http-addr", ":8000", "HTTP listen address")
	slackWebhook   = flag.String("slack-webhook", os.Getenv("SLACK_WEBHOOK_URL"), "URL of a Slack Incoming Webhook integration")
	slackChannel   = flag.String("slack-channel", "", "Slack channel to post messages from SNS")
	slackUsername  = flag.String("slack-username", "", "Post messages to Slack as this user")
	slackIconURL   = flag.String("slack-icon-url", "", "URL to an image to use as the icon for messages")
	slackIconEmoji = flag.String("slack-icon-emoji", "", "Emoji to use as the icon for messages")
)

func main() {
	flag.Parse()
	if *slackWebhook == "" {
		log.Fatal("-slack-webhook or SLACK_WEBHOOK_URL is required")
	}
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}

// SNSMessage implements an Amazon SNS notification message.
// Represents a POST message received on the configured HTTP endpoint, where
// a message body contains a JSON document with the following fields.
type SNSMessage struct {
	Message          string
	MessageId        string
	Signature        string
	SignatureVersion string
	SigningCertURL   string
	Subject          string
	SubscribeURL     string
	Timestamp        time.Time
	Token            string
	TopicArn         string
	Type             string
	UnsubscribeURL   string
}

// NewSNSMessage returns an initialized SNS message by decoding a stream of
// bytes representing the JSON document from a SNS message POST, or nil along
// with any error should they occur.
func NewSNSMessage(js []byte) (*SNSMessage, error) {
	s := new(SNSMessage)
	if err := json.Unmarshal(js, s); err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation("Local")
	if err != nil {
		return s, err
	}
	s.Timestamp = s.Timestamp.In(loc)
	return s, nil
}

// ConfirmSubscription submits GET request to the SubscribeURL field for
// confirmation of subscription to an Amazon SNS topic.
func (s *SNSMessage) ConfirmSubscription() error {
	if _, err := http.Get(s.SubscribeURL); err != nil {
		return err
	}
	return nil
}

// String returns a formatted string containing the timestamp (RFC3339),
// subject, and message body of the Amazon SNS message.
func (s *SNSMessage) String() string {
	return fmt.Sprintf(
		"%s [%s] %s",
		s.Timestamp.Format(time.RFC3339),
		s.Subject,
		s.Message,
	)
}

// SlackMessage implements a message sent to the Slack API.
// Represents an HTTP POST with a JSON payload.
type SlackMessage struct {
	Channel   string `json:"channel,omitempty"`
	Username  string `json:"username,omitempty"`
	IconURL   string `json:"icon_url,omitempty"`
	IconEmoji string `json:"icon_emoji,omitempty"`
	Text      string `json:"text,omitempty"`
}

// NewSlackMessage returns an initialized SlackMessage.
func NewSlackMessage(text string) *SlackMessage {
	return &SlackMessage{
		Channel:   *slackChannel,
		Username:  *slackUsername,
		IconURL:   *slackIconURL,
		IconEmoji: *slackIconEmoji,
		Text:      text,
	}
}

// PostMessage submits a POST to the specified Incoming Webhook URL.
func (s *SlackMessage) PostMessage(url string) (*http.Response, error) {
	payload, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	body := strings.NewReader("payload=" + string(payload))
	resp, err := http.Post(url, "application/x-www-form-urlencoded", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return resp, err
}

func handler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body := make([]byte, r.ContentLength)
	if _, err := r.Body.Read(body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sns, err := NewSNSMessage(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	switch sns.Type {
	case "Notification":
		slack := NewSlackMessage(sns.String())
		if _, err := slack.PostMessage(*slackWebhook); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "SubscriptionConfirmation":
		if err := sns.ConfirmSubscription(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
