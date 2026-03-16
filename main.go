package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wolfword/internal/ws"
)

func main() {
	port := envOrDefault("PORT", "3001")
	dayTimeout := envSeconds("DAY_TIMEOUT_SEC", 300)
	voteTimeout := envSeconds("VOTE_TIMEOUT_SEC", 60)

	hub := ws.NewHub(dayTimeout, voteTimeout)
	r := gin.Default()

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/ws", func(c *gin.Context) {
		hub.HandleWS(c.Writer, c.Request)
	})

	registerSPARoutes(r)

	addr := ":" + port
	log.Printf("listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func registerSPARoutes(r *gin.Engine) {
	distDir := filepath.Join("frontend", "dist")
	indexPath := filepath.Join(distDir, "index.html")

	if _, err := os.Stat(indexPath); err != nil {
		r.GET("/", func(c *gin.Context) {
			c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<!doctype html><html><body><h1>wolfword backend running</h1><p>Frontend build not found.</p></body></html>`))
		})
		return
	}

	r.Static("/assets", filepath.Join(distDir, "assets"))
	r.StaticFile("/favicon.ico", filepath.Join(distDir, "favicon.ico"))
	r.NoRoute(func(c *gin.Context) {
		c.File(indexPath)
	})
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envSeconds(key string, fallback int) time.Duration {
	v := envOrDefault(key, strconv.Itoa(fallback))
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		n = fallback
	}
	return time.Duration(n) * time.Second
}
