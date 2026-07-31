package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/y08lin4/proxy-cat/internal/service"
)

func main() {
	svc := service.New(service.Config{})

	// Detect frontend assets directory
	fd := ""
	for _, dir := range []string{"./frontend/dist", "./dist/frontend", "frontend/dist", "./frontend"} {
		if _, err := os.Stat(dir + "/index.html"); err == nil {
			fd = dir
			break
		}
	}
	// Also check relative to executable
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		for _, rel := range []string{"frontend/dist", "frontend"} {
			dir := filepath.Join(exeDir, rel)
			if _, err := os.Stat(dir + "/index.html"); err == nil {
				fd = dir
				break
			}
		}
	}
	if fd != "" {
		service.SetFrontendDir(fd)
		log.Printf("Serving frontend from %s", fd)
	}

	router := svc.Router()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("Failed to find free port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		url := "http://" + addr
		log.Printf("Opening %s", url)
		openNativeWindow(url)
	}()

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

	log.Printf("Proxy-Cat desktop server on %s", addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

func openNativeWindow(url string) {
	switch runtime.GOOS {
	case "windows":
		// Edge WebView2 --app mode = frameless native window
		edgePaths := []string{
			os.Getenv("ProgramFiles(x86)") + `\Microsoft\Edge\Application\msedge.exe`,
			os.Getenv("ProgramFiles") + `\Microsoft\Edge\Application\msedge.exe`,
			`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
			`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		}
		for _, ep := range edgePaths {
			if _, err := os.Stat(ep); err == nil {
				cmd := exec.Command(ep,
					"--app="+url,
					"--window-size=1160,760",
					"--disable-extensions",
					"--disable-sync",
					"--no-first-run",
					"--no-default-browser-check",
				)
				cmd.Start()
				return
			}
		}
		// Fallback: open default browser
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		exec.Command("open", "-a", "Safari", url).Start()
	default:
		exec.Command("xdg-open", url).Start()
	}
}
