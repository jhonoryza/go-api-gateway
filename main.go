package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"encoding/json"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

type Backend struct {
	ID    int
	URL   string
	Alive atomic.Bool
}

type Route struct {
	Path     string
	Backends []*Backend
	Counter  atomic.Uint64
}

var (
	db     *sql.DB
	routes = map[string]*Route{}
	mu     sync.RWMutex
)

type BackendView struct {
	ID    int    `json:"id"`
	URL   string `json:"url"`
	Alive bool   `json:"alive"`
}

type RouteView struct {
	Path     string        `json:"path"`
	Backends []BackendView `json:"backends"`
}

/* ================= DB LOAD ================= */

func loadRoutes() error {
	rows, err := db.Query(`
		SELECT r.path, b.id, b.target_url
		FROM gateway_routes r
		JOIN gateway_backends b ON b.route_id=r.id
		WHERE b.is_active=true
	`)
	if err != nil {
		return fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	temp := map[string]*Route{}

	for rows.Next() {
		var path string
		var id int
		var url string

		if err := rows.Scan(&path, &id, &url); err != nil {
			return err
		}

		if temp[path] == nil {
			temp[path] = &Route{Path: path}
		}

		b := &Backend{ID: id}
		b.Alive.Store(true)
		b.URL = url

		temp[path].Backends = append(temp[path].Backends, b)
	}

	mu.Lock()
	routes = temp
	mu.Unlock()

	log.Println("Routes loaded:", len(routes))
	return nil
}

/* ================= HEALTH CHECK ================= */

func healthLoop() {
	for {
		mu.RLock()
		for _, route := range routes {
			for _, b := range route.Backends {
				go func(backend *Backend) {
					client := http.Client{
						Timeout: 2 * time.Second,
					}
					resp, err := client.Get(backend.URL + "/health")
					if err == nil {
						resp.Body.Close()
					}
					if err == nil && resp.StatusCode == 200 {
						backend.Alive.Store(true)
					} else {
						backend.Alive.Store(false)
					}
				}(b)
			}
		}
		mu.RUnlock()

		time.Sleep(10 * time.Second)
	}
}

/* ================= PICK BACKEND ================= */

func (r *Route) Pick() *Backend {
	n := len(r.Backends)
	if n == 0 {
		return nil
	}

	for i := 0; i < n; i++ {
		idx := r.Counter.Add(1) % uint64(n)
		b := r.Backends[idx]
		if b.Alive.Load() {
			return b
		}
	}
	return nil
}

/* ================= PROXY ================= */

func serveProxy(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path == "/" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		return
	}
		
	mu.RLock()
	defer mu.RUnlock()

	var reqBody []byte
	if r.Body != nil {
		var buf bytes.Buffer
		tee := io.TeeReader(r.Body, &buf)
		reqBody, _ = io.ReadAll(io.LimitReader(tee, 4096))
		r.Body = io.NopCloser(&buf)
	}

	for path, route := range routes {
		if strings.HasPrefix(r.URL.Path, path) {

			backend := route.Pick()
			if backend == nil {
				http.Error(w, "No healthy backend", 503)
				return
			}

			target, _ := url.Parse(backend.URL)
			proxy := httputil.NewSingleHostReverseProxy(target)

			proxy.Director = func(req *http.Request) {
				newPath := strings.TrimPrefix(req.URL.Path, path)
				if newPath == "" {
					newPath = "/"
				}

				req.URL.Scheme = target.Scheme
				req.URL.Host = target.Host
				req.URL.Path = newPath
				req.URL.RawQuery = r.URL.RawQuery

				req.Host = target.Host
				req.Header = r.Header.Clone()
				req.Header.Set("User-Agent", "Go-Gateway/1.0")

				log.Println("Forwarding to:", target.String()+newPath)
			}

			// ===== Capture response =====
			rec := &responseRecorder{
				ResponseWriter: w,
				status:         200,
			}

			proxy.ServeHTTP(rec, r)

			// ===== Push log async =====
			logItem := GatewayLog{
				Method:     r.Method,
				Path:       r.URL.Path,
				Query:      r.URL.RawQuery,
				URL:        backend.URL,
				ReqBody:    normalizeBody(reqBody, r.Header.Get("Content-Type")),
				RespBody:   normalizeBody(rec.body, rec.Header().Get("Content-Type")),
				StatusCode: rec.status,
				CreatedAt:  time.Now().UTC(),
			}

			select {
			case logQueue <- logItem:
			default:
				log.Println("logQueue full, drop log")
			}

			return
		}
	}

	http.NotFound(w, r)
}

func reloadHandler(w http.ResponseWriter, r *http.Request) {

	if r.Header.Get("X-ADMIN-TOKEN") != os.Getenv("ADMIN_TOKEN") {
		http.Error(w, "Forbidden", 403)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", 405)
		return
	}

	if err := loadRoutes(); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write([]byte("Routes reloaded"))
}

func listRoutesHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", 405)
		return
	}

	if r.Header.Get("X-ADMIN-TOKEN") != os.Getenv("ADMIN_TOKEN") {
		http.Error(w, "Forbidden", 403)
		return
	}

	mu.RLock()
	defer mu.RUnlock()

	var result []RouteView

	for _, route := range routes {

		rv := RouteView{
			Path: route.Path,
		}

		for _, b := range route.Backends {
			rv.Backends = append(rv.Backends, BackendView{
				ID:    b.ID,
				URL:   b.URL,
				Alive: b.Alive.Load(),
			})
		}

		result = append(result, rv)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

/* ================= MAIN ================= */

func main() {
	var err error
	if err := godotenv.Load(); err != nil {
	    log.Println("No .env file found, using system environment variables")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("Error loading DATABASE_URL from .env file")
	}

	db, err = sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(100)
	db.SetConnMaxIdleTime(5 * time.Second)
	db.SetConnMaxLifetime(60 * time.Second)
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("DB ping failed:", err)
	}

	if err = loadRoutes(); err != nil {
		log.Fatal(err)
	}

	go healthLoop()

	startLogWorker()
	startRetentionWorker()

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/logs", logsHandler)
	http.HandleFunc("/routes", listRoutesHandler)
	http.HandleFunc("/reload", reloadHandler)
	http.HandleFunc("/", serveProxy)

	log.Println("Gateway running on :8080")
	http.ListenAndServe(":8080", nil)
}
