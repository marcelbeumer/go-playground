package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/alecthomas/kong"
	"github.com/marcelbeumer/go-playground/streamproc/services/event-producer/internal/log"
)

type CLI struct {
	APIHost   string `help:"API host."                                         short:"h" default:"127.0.0.1" env:"API_HOST"`
	APIPort   int    `help:"API port."                                         short:"p" default:"9998"      env:"API_PORT"`
	PlainText bool   `help:"Send events to the API plain text instead of JSON"                               env:"PLAIN_TEXT"`
	Disable   bool   `help:"Disable all processing."                           short:"x"                     env:"DISABLE"`
}

type EventBuffer struct {
	events []Event
	l      sync.RWMutex
}

func (b *EventBuffer) Append(e Event) {
	b.l.Lock()
	defer b.l.Unlock()
	b.events = append(b.events, e)
}

func (b *EventBuffer) Recover(e []Event) {
	b.l.Lock()
	defer b.l.Unlock()
	b.events = append(e, b.events...)
}

func (b *EventBuffer) Flush() []Event {
	b.l.Lock()
	defer b.l.Unlock()
	slice := b.events[:]
	b.events = make([]Event, 0)
	return slice
}

// Event is send to server for processing
type Event struct {
	// Time is timestamp in UnixMilli
	Time int64 `json:"time"`
	// Amount if number between 0-10
	Amount *big.Float `json:"amount"`
}

func main() {
	zl := log.NewZapLogger(os.Stderr)
	log.RedirectStdLog(zl)
	logger := log.NewZapLoggerAdapter(zl)

	cli := CLI{}
	_ = kong.Parse(
		&cli,
		kong.Name("event-producer"),
		kong.UsageOnError(),
	)

	buffer := EventBuffer{}

	go func() {
		for {
			time.Sleep(time.Second * 2)
			events := buffer.Flush()
			if len(events) == 0 {
				continue
			}

			url := fmt.Sprintf("http://%s:%d", cli.APIHost, cli.APIPort)
			var contentType string
			var reqBody []byte

			if cli.PlainText {
				contentType = "application/text"
				for _, event := range events {
					line, err := json.Marshal(&event)
					if err != nil {
						logger.Error(log.Error(err))
						os.Exit(1)
					}
					if len(reqBody) > 0 {
						reqBody = append(reqBody, []byte("\n")...)
					}
					reqBody = append(reqBody, line...)
				}
			} else {
				contentType = "application/json"
				j, err := json.Marshal(&events)
				if err != nil {
					logger.Error(log.Error(err))
					os.Exit(1)
				}
				reqBody = j
			}

			if cli.Disable {
				logger.Info("processing disabled, not sending events")
				continue
			}

			resp, err := http.Post(url, contentType, bytes.NewBuffer(reqBody))
			if err != nil {
				logger.Errorw("request error", log.Error(err))
				buffer.Recover(events)
				continue
			}

			defer resp.Body.Close()
			respBody, err := ioutil.ReadAll(resp.Body)
			if resp.StatusCode >= 400 {
				logger.Errorw("request http error",
					"statusCode", resp.StatusCode,
					"responseBody", respBody)
				buffer.Recover(events)
				continue
			}

			logger.Infow("sent events",
				"amount", len(events))
		}
	}()

	for {
		event := Event{
			Time:   time.Now().UnixMilli(),
			Amount: big.NewFloat(rand.Float64() * 10),
		}
		buffer.Append(event)
		time.Sleep(time.Second)
	}
}
