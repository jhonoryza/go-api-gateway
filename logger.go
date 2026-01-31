package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"encoding/base64"
	"encoding/json"
	"fmt"
)

type responseRecorder struct {
	http.ResponseWriter
	status int
	body   []byte
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	return r.ResponseWriter.Write(b)
}

type GatewayLog struct {
	Method     string
	Path       string
	Query      string
	URL        string
	ReqBody    string
	RespBody   string
	StatusCode int
	CreatedAt  time.Time
}

var logQueue = make(chan GatewayLog, 1000)

const maxBodySize = 4096

func limitString(s string) string {
	if len(s) > maxBodySize {
		return s[:maxBodySize]
	}
	return s
}

func extractBoundary(ct string) string {
	_, params, err := mime.ParseMediaType(ct)
	if err != nil {
		return ""
	}
	return params["boundary"]
}

func bodyToString(b []byte, contentType string) string {

	if len(b) == 0 {
		return ""
	}

	// ===== JSON =====
	if strings.Contains(contentType, "application/json") {
		if utf8.Valid(b) {
			return string(b)
		}
	}

	// ===== FORM URLENCODED =====
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		return string(b)
	}

	// ===== MULTIPART =====
	if strings.Contains(contentType, "multipart/form-data") {

		reader := multipart.NewReader(
			bytes.NewReader(b),
			extractBoundary(contentType),
		)

		result := make(map[string]string)

		for {
			part, err := reader.NextPart()
			if err != nil {
				break
			}

			name := part.FormName()
			filename := part.FileName()

			data, _ := io.ReadAll(io.LimitReader(part, 1024))

			if filename != "" {
				result[name] = "[FILE:" + filename + "]"
			} else {
				result[name] = string(data)
			}
		}

		out, _ := json.Marshal(result)
		return string(out)
	}

	// ===== FALLBACK =====
	if utf8.Valid(b) {
		return string(b)
	}

	return base64.StdEncoding.EncodeToString(b)
}

func gunzipIfNeeded(body []byte) []byte {
    r, err := gzip.NewReader(bytes.NewReader(body))
    if err != nil {
        return body
    }
    defer r.Close()

    out, err := io.ReadAll(r)
    if err != nil {
        return body
    }
    return out
}

func normalizeBody(body []byte, contentType string) string {
	return limitString(bodyToString(gunzipIfNeeded(body), contentType))
}


func startLogWorker() {
	go func() {
		for logItem := range logQueue {
			_, err := db.Exec(`
				INSERT INTO gateway_logs
				(method, path, query, target_url, request_body, response_body, status_code, created_at)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			`,
				logItem.Method,
				logItem.Path,
				logItem.Query,
				logItem.URL,
				logItem.ReqBody,
				logItem.RespBody,
				logItem.StatusCode,
				logItem.CreatedAt,
			)

			if err != nil {
				log.Println("log insert error:", err)
			}
		}
	}()
}

func startRetentionWorker() {
	go func() {
		for {
			_, err := db.Exec(`
				DELETE FROM gateway_logs
				WHERE created_at < NOW() - INTERVAL '7 day'
			`)
			if err != nil {
				log.Println("retention error:", err)
			}
			time.Sleep(1 * time.Hour)
		}
	}()
}

func logsHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", 405)
		return
	}

	if r.Header.Get("X-ADMIN-TOKEN") != os.Getenv("ADMIN_TOKEN") {
		http.Error(w, "Forbidden", 403)
		return
	}

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		fmt.Sscan(v, &limit)
	}

	cursor := r.URL.Query().Get("cursor")
	q := r.URL.Query().Get("q")
	var args []any
	query := `
		SELECT
			id, method, path, query, target_url,
			request_body, response_body,
			status_code, created_at
		FROM gateway_logs
		WHERE 1=1
	`

	if q != "" {
		args = append(args, "%"+q+"%")
		query += fmt.Sprintf(`
			AND (path ILIKE $%d OR request_body ILIKE $%d)
		`, len(args), len(args))
	}

	if cursor != "" {
		args = append(args, cursor)
		query += fmt.Sprintf(`
			AND created_at < $%d
		`, len(args))
	}

	args = append(args, limit)
	query += fmt.Sprintf(`
		ORDER BY created_at DESC
		LIMIT $%d
	`, len(args))

	rows, err := db.Query(query, args...)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	type LogRow struct {
		ID        int64     `json:"id"`
		Method    string    `json:"method"`
		Path      string    `json:"path"`
		Query     string    `json:"query"`
		URL       string    `json:"target_url"`
		Status    int       `json:"status_code"`
		Request   string    `json:"request_body"`
		Response  string    `json:"response_body"`
		CreatedAt time.Time `json:"created_at"`
	}

	var list []LogRow
	var nextCursor string

	for rows.Next() {
		var l LogRow
		rows.Scan(
			&l.ID,
			&l.Method,
			&l.Path,
			&l.Query,
			&l.URL,
			&l.Request,
			&l.Response,
			&l.Status,
			&l.CreatedAt,
		)
		list = append(list, l)

		nextCursor = l.CreatedAt.Format(time.RFC3339Nano)
	}

	resp := map[string]any{
		"data":        list,
		"next_cursor": nextCursor,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
