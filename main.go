package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/y08lin4/proxy-cat/internal/service"
)

func main() {
	port := flag.Int("port", 8080, "HTTP API listen port")
	headless := flag.Bool("headless", false, "Run without desktop shell")
	noSystemProxy := flag.Bool("no-system-proxy", false, "Disable system proxy features")
	mihomoBinary := flag.String("mihomo-binary", "mihomo.exe", "Path to mihomo binary")
	dataDir := flag.String("data-dir", "", "Data directory (default: %APPDATA%/Proxy-Cat)")
	frontendDir := flag.String("frontend-dir", "", "Path to frontend assets (default: frontend/dist if it exists)")
	flag.Parse()

	cfg := service.DefaultConfig()
	if *dataDir != "" {
		cfg.DataDir = *dataDir
		cfg.MihomoHome = filepath.Join(*dataDir, "mihomo")
	}
	cfg.Headless = *headless
	cfg.NoSystemProxy = *noSystemProxy
	cfg.MihomoBinary = *mihomoBinary

	svc := service.New(cfg)

	// Determine frontend directory: flag > env var > auto-detect
	fd := *frontendDir
	if fd == "" {
		fd = os.Getenv("FRONTEND_DIR")
	}
	if fd == "" {
		if _, err := os.Stat("frontend/dist/index.html"); err == nil {
			fd = "frontend/dist"
		}
	}
	if fd != "" {
		service.SetFrontendDir(fd)
	}

	router := svc.Router()
	addr := fmt.Sprintf("127.0.0.1:%d", *port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		svc.Shutdown(ctx)
		srv.Shutdown(ctx)
	}()

	log.Printf("Proxy-Cat API server listening on %s", addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
