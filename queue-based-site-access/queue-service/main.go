package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/messeb/docker-playground/queue-based-site-access/internal/config"
	"github.com/messeb/docker-playground/queue-based-site-access/internal/handler"
	"github.com/messeb/docker-playground/queue-based-site-access/internal/queue"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	rdb := connectRedis(cfg.RedisAddr, ctx)
	store := queue.New(rdb)
	store.InitCapacity(ctx, cfg.Capacity)

	go store.RunWorker(ctx)

	h := handler.New(store, cfg, newProxy(cfg.TargetURL))

	log.Printf("queue-service listening on :%s (target=%s)", cfg.Port, cfg.TargetURL)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, h.Routes()))
}

// connectRedis creates a Redis client and waits until the server responds.
func connectRedis(addr string, ctx context.Context) *redis.Client {
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	for i := 0; i < 30; i++ {
		if rdb.Ping(ctx).Err() == nil {
			return rdb
		}
		log.Printf("waiting for Redis at %s ...", addr)
		time.Sleep(time.Second)
	}
	return rdb
}

// newProxy builds a reverse proxy to the target service.
func newProxy(targetURL string) *httputil.ReverseProxy {
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("invalid TARGET_URL %q: %v", targetURL, err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	orig := proxy.Director
	proxy.Director = func(req *http.Request) {
		orig(req)
		req.Host = target.Host
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error: %v", err)
		http.Error(w, "Service temporarily unavailable", http.StatusBadGateway)
	}
	return proxy
}
