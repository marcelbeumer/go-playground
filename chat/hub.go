package chat

import (
	"fmt"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/marcelbeumer/crispy-octo-goggles/chat/kvstore"
	"github.com/marcelbeumer/crispy-octo-goggles/chat/log"
	"github.com/marcelbeumer/crispy-octo-goggles/chat/queue"
)

// Use simple int increment for ids.
type hubId = int

// hubUser encapsulates a user in the hub.
type hubUser struct {
	name   string
	conn   Connection
	events *queue.Queue[Event]
}

// Hub is the chat hub/room where users can connect to.
type Hub struct {
	logger  log.Logger
	users   *kvstore.KVStore[hubId, *hubUser]
	usersMu sync.RWMutex
	idInc   hubId
}

func (h *Hub) Connect(username string, conn Connection) (hubId, error) {
	userId, err := h.newUser(username, conn)
	if err != nil {
		return userId, err
	}
	go func() {
		if err := h.pumpFromUser(userId); err != nil {
			h.Disconnect(userId)
		}
	}()
	go func() {
		if err := h.pumpToUser(userId); err != nil {
			h.Disconnect(userId)
		}
	}()
	return userId, nil
}

func (h *Hub) Disconnect(userId hubId) error {
	return h.disconnectUser(userId, false)
}

func (h *Hub) genId() hubId {
	h.idInc++
	return h.idInc
}

func (h *Hub) newUser(username string, conn Connection) (hubId, error) {
	h.usersMu.Lock()
	for _, user := range h.users.Values() {
		if user.name == username {
			return 0, &ErrUsernameExists{username: username}
		}
	}
	events := queue.NewQueue[Event]()
	events.Add(&EventConnected{ // first event
		EventMeta: *NewEventMetaNow(),
		Users:     h.userList(username),
	})

	userId := h.genId()
	h.users.Set(userId, &hubUser{
		name:   username,
		conn:   conn,
		events: events,
	})
	defer h.usersMu.Unlock()

	others := h.userIds(userId)
	h.sendEvent(&EventUserEnter{
		EventMeta: *NewEventMetaNow(),
		Name:      username,
	}, others...)
	h.sendEvent(&EventUserListUpdate{
		EventMeta: *NewEventMetaNow(),
		Users:     h.userList(),
	}, others...)

	return userId, nil
}

func (h *Hub) disconnectUser(userId hubId, notify bool) error {
	h.usersMu.Lock()
	defer h.usersMu.Unlock()

	user, err := h.findUser(userId)
	if err != nil {
		return err
	}
	h.users.Delete(userId)
	user.events.Close()

	if notify {
		// Notify other users.
		others := h.userIds(userId)
		h.sendEvent(&EventUserLeave{
			EventMeta: *NewEventMetaNow(),
			Name:      user.name,
		}, others...)
		h.sendEvent(&EventUserListUpdate{
			EventMeta: *NewEventMetaNow(),
			Users:     h.userList(),
		}, others...)
	}

	// Give the user some time to consume events.
	select {
	case <-time.After(time.Second * 2):
	case <-user.events.Empty():
	}
	// Truly disconnect the user.
	return user.conn.Close(nil)
}

func (h *Hub) userIds(exclude ...hubId) []hubId {
	ex := map[int]bool{}
	for _, v := range exclude {
		ex[v] = true
	}
	keys := h.users.Keys()
	ids := []hubId{}
	for _, v := range keys {
		if _, ok := ex[v]; !ok {
			ids = append(ids, v)
		}
	}
	return ids
}

func (h *Hub) userList(pendingUsernames ...string) []string {
	coll := map[string]struct{}{}

	for _, user := range h.users.Values() {
		coll[user.name] = struct{}{}
	}
	for _, name := range pendingUsernames {
		coll[name] = struct{}{}
	}

	names := []string{}
	for key := range coll {
		names = append(names, key)
	}

	sort.Strings(names)
	return names
}

func (h *Hub) findUser(userId hubId) (*hubUser, error) {
	user, _ := h.users.Get(userId)
	if user == nil {
		return nil, &ErrUserIdNotFound{id: userId}
	}
	return user, nil
}

func (h *Hub) pumpToUser(userId hubId) error {
	user, err := h.findUser(userId)
	if err != nil {
		return err
	}
	for {
		e, err := user.events.Read()
		if err != nil {
			return err
		}
		err = user.conn.SendEvent(e)
		if err != nil {
			return err
		}
	}
}

func (h *Hub) pumpFromUser(userId hubId) error {
	user, err := h.findUser(userId)
	if err != nil {
		return err
	}
	for {
		e, err := user.conn.ReadEvent()
		if err != nil {
			return err
		}
		if err := h.handleEvent(userId, e); err != nil {
			return err
		}
	}
}

func (h *Hub) handleEvent(userId hubId, e Event) error {
	logger := h.logger
	user, err := h.findUser(userId)
	if err != nil {
		return err
	}

	switch t := e.(type) {
	case *EventConnected:
	case *EventUserListUpdate:
	case *EventUserEnter:
	case *EventUserLeave:
		//
	case *EventSendMessage:
		h.sendEvent(&EventNewMessage{
			EventMeta: *NewEventMetaNow(),
			Sender:    user.name,
			Message:   t.Message,
		}, h.userIds()...)

	case *EventNewMessage:
		//
	default:
		logger.Warnw(
			"unhandled event type",
			"username", user.name,
			"userid", userId,
			"type", reflect.TypeOf(e).String())
	}
	return nil
}

func (h *Hub) sendEvent(e Event, userIds ...hubId) error {
	for _, userId := range userIds {
		user, err := h.findUser(userId)
		if err != nil {
			return err
		}
		if err := user.events.Add(e); err != nil {
			h.disconnectUser(userId, true) // unforgiving
			return err
		}
	}
	return nil
}

func NewHub(logger log.Logger) *Hub {
	return &Hub{
		logger:  logger,
		users:   kvstore.NewKVStore[int, *hubUser](),
		usersMu: sync.RWMutex{},
		idInc:   0,
	}
}

// ErrHubUserNotFound when hub did not find the user by id.
type ErrUserNotFound struct {
	username string
}

func (e *ErrUserNotFound) Error() string {
	return fmt.Sprintf(`uknown user "%s"`, e.username)
}

// ErrUserIdNotFound when hub did not find the user by id.
type ErrUserIdNotFound struct {
	id hubId
}

func (e *ErrUserIdNotFound) Error() string {
	return fmt.Sprintf(`uknown user id "%d"`, e.id)
}

// ErrUsernameExists for when the hub already has the user(name)
type ErrUsernameExists struct {
	username string
}

func (e *ErrUsernameExists) Error() string {
	return fmt.Sprintf(`user "%s" already exists`, e.username)
}
