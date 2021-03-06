package chat

import (
	"time"

	"github.com/marcelbeumer/go-playground/gochat/internal/util/now"
)

// Event is the interface for all events
type Event interface {
	// When returns time of the event.
	// Important we define *something* more than interface{}
	// for static analysis to work on *Struct{} vs Struct{}
	When() time.Time
}

// EventMeta is a base struct for other events to include
// basic meta data that all events have
type EventMeta struct {
	Time time.Time `json:"time"`
}

// When returns the time of the event.
func (e *EventMeta) When() time.Time {
	return e.Time
}

// NewEventMetaNow returns EventMeta with time set to "now".
func NewEventMetaNow() *EventMeta {
	return &EventMeta{
		Time: now.Now(),
	}
}

// EventConnected is (guaranteed) the first event sent when
// a new connection is made
type EventConnected struct {
	EventMeta
	Users []string `json:"users"`
}

type EventUserListUpdate struct {
	EventMeta
	Users []string `json:"users"`
}

type EventUserEnter struct {
	EventMeta
	Name string `json:"name"`
}

type EventUserLeave struct {
	EventMeta
	Name string `json:"name"`
}

type EventSendMessage struct {
	EventMeta
	Message string `json:"message"`
}

type EventNewMessage struct {
	EventMeta
	Sender  string `json:"sender"`
	Message string `json:"message"`
}
