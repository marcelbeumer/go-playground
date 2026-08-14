package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var echoData = "hello world! this is some data for you" // default

func main() {
	var addr string
	var filePath string
	var targetAddr string

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: tcpecho [-addr HOST:PORT] [-target HOST:PORT] [-f {-|PATH}]")
		fmt.Fprintln(os.Stderr)
		flag.PrintDefaults()
	}
	flag.StringVar(&filePath, "f", "", "echo data from file (- for stdin)")
	flag.StringVar(&addr, "addr", "", "host:port")
	flag.StringVar(&targetAddr, "target", "", "target host:port")
	flag.Parse()

	if filePath != "" {
		var r io.Reader
		if filePath == "-" {
			r = os.Stdin
		} else {
			f, err := os.Open(filePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Read echo data file: %s", err)
				os.Exit(1)
			}
			defer f.Close()
			r = f
		}

		b, err := io.ReadAll(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Read data: %s", err)
			os.Exit(1)
		}
		echoData = string(b)
	}

	ctx, cancelCtx := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelCtx()

	var wg sync.WaitGroup

	if addr != "" {
		wg.Go(func() {
			if err := listener(ctx, addr); err != nil {
				fmt.Fprintf(os.Stderr, "Listener: %s", err)
			}
		})
	}

	if targetAddr != "" {
		wg.Go(func() {
			if err := sender(ctx, targetAddr); err != nil {
				fmt.Fprintf(os.Stderr, "Sender: %s", err)
			}
		})
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		cancelCtx()
	case <-ctx.Done():
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			fmt.Fprintln(os.Stderr, "Timed out waiting for goroutines to exit")
		}
	}
}

func listener(ctx context.Context, addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Listening on %s", addr)

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return err
			}
		}

		go func() {
			defer conn.Close()
			l := slog.With(
				slog.String("role", "listener"),
				slog.String("addr", conn.RemoteAddr().String()),
			)
			l.Info("Handling listener connection")
			handleListenerConn(ctx, conn, l)
		}()
	}
}

func handleListenerConn(ctx context.Context, conn net.Conn, l *slog.Logger) {
	for {
		select {
		case <-ctx.Done():
			l.Info("Context canceled")
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(time.Second))

		b := make([]byte, 32) // for demo
		n, err := conn.Read(b)
		if err != nil {
			if errors.Is(err, io.EOF) {
				l.Info("Done (EOF)")
				return
			}
			l.Info("Read error, stopping", slog.Any("err", err))
			return
		}

		l.Info("Read", slog.Int("n", n))
		conn.SetReadDeadline(time.Now().Add(time.Second))

		n, err = conn.Write(b[:n])
		if err != nil {
			l.Info("Write error, stopping", slog.Any("err", err))
			return
		}

		l.Info("Wrote", slog.Int("n", n))
	}
}

func sender(ctx context.Context, targetAddr string) error {
	ticker := time.NewTicker(2 * time.Second)
	l := slog.Default()

	tcpAddr, err := net.ResolveTCPAddr("tcp", targetAddr)
	if err != nil {
		return fmt.Errorf("resolve addr: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			l.Info("Context canceled")
			return nil
		case <-ticker.C:
		}

		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			l.Info("Dial error", slog.Any("err", err))
			continue
		}

		go func() {
			defer conn.Close()
			l := slog.With(
				slog.String("role", "sender"),
				slog.String("addr", conn.RemoteAddr().String()),
			)
			l.Info("Handling listener connection")
			l.Info("Handling sender connection")
			handleSenderConn(ctx, conn, l)
		}()
	}
}

func handleSenderConn(ctx context.Context, conn *net.TCPConn, l *slog.Logger) {
	var wg sync.WaitGroup

	wg.Go(func() {
		b := []byte(echoData)
		chunkSize := 8
		for len(b) > 0 {
			select {
			case <-ctx.Done():
				l.Info("Write: context canceled")
				return
			default:
			}

			end := min(chunkSize, len(b))
			conn.SetWriteDeadline(time.Now().Add(time.Second))
			n, err := conn.Write(b[:end])
			if err != nil {
				l.Info("Write error, stopping", slog.Any("err", err))
				return
			}
			l.Info("Wrote", slog.Int("n", n))
			b = b[n:]
		}
		conn.CloseWrite()
	})

	var echo []byte

	wg.Go(func() {
		b := make([]byte, 8)
		for {
			select {
			case <-ctx.Done():
				l.Info("Read: context canceled")
				return
			default:
			}

			conn.SetReadDeadline(time.Now().Add(time.Second))
			n, err := conn.Read(b)
			if err != nil {
				if errors.Is(err, io.EOF) {
					l.Info("Read: EOF")
					return
				}
				l.Info("Read error, stopping", slog.Any("err", err))
				return
			}

			echo = append(echo, b[:n]...)
			l.Info("Read", slog.Int("n", n))
		}
	})

	wg.Wait()

	got, want := string(echo), echoData
	l.Info("Result", slog.Bool("same", got == want), slog.Int("out", len(echoData)), slog.Int("in", len(got)))
}
