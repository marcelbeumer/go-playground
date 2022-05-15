package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/marcelbeumer/crispy-octo-goggles/chat"
	"github.com/marcelbeumer/crispy-octo-goggles/chat/log"
)

type ClientServerOpts struct {
	Host string `help:"Server host." short:"h" default:"127.0.0.1"`
	Port int    `help:"Server port." short:"p" default:"9998"`
}

type ClientOpts struct {
	Username       string `help:"Username."                   required:"" short:"u"`
	StdoutFrontend bool   `help:"Use simple stdout frontend."             short:"s"`
}

type Commands struct {
	Verbose     bool `help:"Verbose (logging info)"       short:"v"`
	VeryVerbose bool `help:"Very verbose (logging debug)" short:"V"`
	Client      struct {
		ClientServerOpts
		ClientOpts
	} `help:"Start client"                           cmd:"client"`
	Server struct {
		ClientServerOpts
	} `help:"Start server"                           cmd:"client"`
	Test struct {
	} `help:"Run non-http test scenario"             cmd:"test"`
}

func main() {
	cli := Commands{}
	ctx := kong.Parse(
		&cli,
		kong.Name("chat"),
		kong.UsageOnError(),
	)

	switch ctx.Command() {

	case "client":
		stdErrBuf := bufio.NewWriter(os.Stderr)
		defer stdErrBuf.Flush()
		zl := log.NewZapLogger(stdErrBuf, cli.Verbose, cli.VeryVerbose)
		logger := log.NewZapLoggerAdapter(zl)
		log.RedirectStdLog(zl)

		addr := fmt.Sprintf("%s:%d", cli.Server.Host, cli.Server.Port)

		conn, err := chat.NewWebsocketClientConnection(addr, cli.Client.Username, logger)
		if err != nil {
			logger.Errorw(
				"could not connect websocket",
				"error", err.Error(),
				"addr", addr,
			)
			os.Exit(1)
		}

		defer conn.Close()

		frontendErr := func(err error) {
			logger.Error("frontend error", "error", err.Error())
			os.Exit(1)
		}

		if cli.Client.StdoutFrontend {
			fe := chat.NewStdoutFrontend(conn, logger)
			if err := fe.Start(); err != nil {
				frontendErr(err)
			}
		} else {
			fe, err := chat.NewGUIFrontend(conn, logger)
			if err != nil {
				frontendErr(err)
			}
			if err := fe.Start(); err != nil {
				frontendErr(err)
			}
		}

	case "server":
		zl := log.NewZapLogger(os.Stderr, cli.Verbose, cli.VeryVerbose)
		logger := log.NewZapLoggerAdapter(zl)
		log.RedirectStdLog(zl)

		addr := fmt.Sprintf("%s:%d", cli.Server.Host, cli.Server.Port)
		s := chat.NewWebsocketServer(logger)

		if err := s.Start(addr); err != nil {
			logger.Error("server error", "error", err.Error())
			os.Exit(1)
		}

	case "test":
	}
}
