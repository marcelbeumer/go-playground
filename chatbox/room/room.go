package room

import (
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/marcelbeumer/crispy-octo-goggles/chatbox/base"
)

type Room struct {
	uuid   string
	outCh  chan base.Event
	userCh map[string]chan base.Event
	doneCh chan struct{}
	state  base.RoomState
	l      *sync.RWMutex
}

func (r *Room) Start() {
	if r.doneCh != nil {
		return
	}
	r.doneCh = make(chan struct{})
	go (func() {
		for {
			select {
			case e := <-r.outCh:
				if err := r.handleEvent(e); err != nil {
					fmt.Println("Error handling event", err)
				}
			case <-r.doneCh:
				r.doneCh = nil
				break
			}
		}

	})()
}

func (r *Room) Stop() {
	if r.doneCh != nil {
		close(r.doneCh)
	}
}

func (r *Room) Wait() {
	if r.doneCh != nil {
		<-r.doneCh
	}
}

func (r *Room) handleEvent(e base.Event) error {
	switch e.Name {

	case base.Connect:
		uuid, data, err := unpackUserEvent[base.UserState](*r, e)
		if err != nil {
			return err
		}

		r.l.Lock()
		r.state.Users[uuid] = data
		r.l.Unlock()

		m := base.UserRef{Uuid: uuid, State: data}
		ne, err := base.NewEvent(base.NewUser, m, r.uuid)
		if err != nil {
			return err
		}
		r.sendEventToAll(ne)

	case base.SendMessage:
		uuid, data, err := unpackUserEvent[string](*r, e)
		if err != nil {
			return err
		}

		r.l.Lock()
		name := r.state.Users[uuid].Name
		msg := base.Message{Sender: e.Sender, SenderName: name, Message: data}
		r.state.Messages = append(r.state.Messages, msg)
		r.l.Unlock()

		n1, err := base.NewEvent(base.RoomStateUpdate, r.state, r.uuid)
		if err != nil {
			return err
		}
		n2, err := base.NewEvent(base.NewMessage, msg, r.uuid)
		if err != nil {
			return err
		}
		r.sendEventToAll(n1, n2)

	default:
		return base.UhandledEventError{Event: e, Receiver: r.uuid}
	}
	return nil
}

func (r *Room) HasUuid(uuid string) bool {
	for key := range r.userCh {
		if key == uuid {
			return true
		}
	}
	return false
}

func (r *Room) Connect(uuid string) (in <-chan base.Event, out chan<- base.Event, err error) {
	if found := r.HasUuid(uuid); found {
		return nil, nil, errors.New("uuid already added")
	}

	userCh := make(chan base.Event)

	r.l.Lock()
	r.state.Users[uuid] = base.UserState{}
	r.userCh[uuid] = userCh
	r.l.Unlock()

	in = (<-chan base.Event)(userCh)
	out = (chan<- base.Event)(r.outCh)

	return in, out, nil
}

func (r *Room) sendEventToAll(events ...base.Event) {
	for _, e := range events {
		for _, ch := range r.userCh {
			r.sendEvent(ch, e)
		}
	}
}

func (r *Room) sendEvent(ch chan base.Event, e base.Event) {
	go (func() {
		ch <- e
	})()
}

func NewRoom() *Room {
	r := &Room{
		outCh: make(chan base.Event),
		state: base.RoomState{
			Users:    make(map[string]base.UserState),
			Messages: make([]base.Message, 0),
		},
		userCh: make(map[string]chan base.Event),
		uuid:   "room:" + uuid.NewString(),
		l:      &sync.RWMutex{},
	}
	r.Start()
	return r
}

func unpackUserEvent[T any](r Room, e base.Event) (string, T, error) {
	uuid := e.Sender
	data, err := base.GetData[T](e)
	if err != nil {
		return uuid, data, err
	}
	if found := r.HasUuid(uuid); !found {
		err := base.NewEventError(e, fmt.Sprintf("user %s not found", uuid))
		return uuid, data, err
	}
	return uuid, data, nil
}
