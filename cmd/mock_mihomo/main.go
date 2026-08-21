package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"mihomoTui/internal/testapi"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9090", "Listen address for mock Mihomo API")
	flag.Parse()

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", *addr, err)
	}

	srv := testapi.NewMockMihomoServer()
	handler := srv.Config.Handler

	httpServer := &http.Server{
		Handler: handler,
	}

	go func() {
		fmt.Printf("Mock Mihomo API server listening on http://%s\n", *addr)
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Mock server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down mock server...")
	_ = httpServer.Close()
	srv.Close()
}
