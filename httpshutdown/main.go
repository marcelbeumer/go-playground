package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s -addr HOST:PORT", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	var addr string
	var help bool

	flag.BoolVar(&help, "h", false, "show help")
	flag.StringVar(&addr, "addr", "", "host:port")
	flag.Parse()

	if help {
		usage()
		os.Exit(0)
	}
	if addr == "" {
		usage()
		os.Exit(1)
	}

	acceptCtx, cancelAcceptCtx := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelAcceptCtx()

	serverCtx, cancelServerCtx := context.WithCancel(context.Background())
	defer cancelServerCtx()

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-acceptCtx.Done():
			http.Error(w, "Shutting down", http.StatusServiceUnavailable)
		default:
			fmt.Fprintln(w, "OK")
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
			fmt.Fprintln(w, "Hello, world!")
		case <-r.Context().Done():
			http.Error(w, "Request cancelled.", http.StatusRequestTimeout)
		}
	})

	server := &http.Server{
		Addr: addr,
		BaseContext: func(_ net.Listener) context.Context {
			return serverCtx
		},
	}

	go func() {
		log.Printf("Server starting on %s.\n", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	<-acceptCtx.Done() // no longer accepting new requests
	log.Println("Received shutdown signal, shutting down.")

	time.Sleep(2 * time.Second) // grace period for running requests
	cancelServerCtx()           // cancel context now

	shutdownCtx, cancelShutdownCtx := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelShutdownCtx()

	err := server.Shutdown(shutdownCtx) // shutdown with timeout
	if err != nil {
		log.Println("Failed to wait for ongoing requests to finish, waiting for forced cancellation.")
		time.Sleep(2 * time.Second) // grace period for running requests
	}

	log.Println("Server shut down gracefully.")
}
