# 🚀 Go Mini API Gateway

Mini API Gateway / Reverse Proxy berbasis Go dengan fitur:

* Path-based routing
* Multiple backend per route
* Load balancing (round-robin)
* Active health check
* Database-driven configuration (PostgreSQL)
* In-memory cache
* Manual reload config via endpoint
* Status endpoint untuk monitoring

Cocok untuk:

* Gateway internal
* Failover API
* Lightweight load balancer
* Proyek riset / skripsi / prototype

---

# 📐 Arsitektur Singkat

```
Client
  |
  v
Go Gateway
  |
  +--> Lookup routes in memory
  |
  +--> Pick healthy backend
  |
  +--> ReverseProxy
  |
  +--> API A / API B / ...
```

Konfigurasi disimpan di PostgreSQL dan dicache di memory.

---

# 📁 Struktur Project

```
gateway/
 ├─ go.mod
 ├─ main.go
 └─ README.md
```

---

# 🧰 Prasyarat

* Go >= 1.22
* PostgreSQL >= 12
* Git

Cek versi:

```bash
go version
psql --version
```

---

# 🗄️ Database Schema

Jalankan SQL berikut di PostgreSQL:

```sql
CREATE TABLE gateway_routes (
    id SERIAL PRIMARY KEY,
    path TEXT UNIQUE NOT NULL
);

CREATE TABLE gateway_backends (
    id SERIAL PRIMARY KEY,
    route_id INT REFERENCES routes(id) ON DELETE CASCADE,
    target_url TEXT NOT NULL,
    is_active BOOLEAN DEFAULT true
);
```

## Contoh Data

```sql
INSERT INTO gateway_routes(path) VALUES('/blog');

INSERT INTO gateway_backends(route_id, target_url)
VALUES
(1, 'https://api-a.example.com'),
(1, 'https://api-b.example.com');
```

---

# 🔐 Environment Variables

Gateway membaca koneksi database dari:

```env
DATABASE_URL=postgres://user:password@host:5432/dbname?sslmode=disable
ADMIN_TOKEN=supersecret
```

Contoh Linux/macOS:

```env
export DATABASE_URL="postgres://postgres:pass@localhost:5432/gateway?sslmode=disable"
export ADMIN_TOKEN="secret123"
```

---

# 📦 Install Dependency

Masuk folder project:

```bash
cd gateway
```

Download dependency:

```bash
go mod tidy
```

---

# ▶️ Menjalankan Aplikasi (Development)

```bash
go run main.go
```

Output:

```
Gateway running on :8080
```

---

# 🏗️ Build Binary

```bash
go build -o gateway
```

Jalankan:

```bash
./gateway
```

---

# 🌍 Contoh Request

Client request:

```
GET http://localhost:8080/users/profile?id=5
```

Gateway akan forward ke salah satu backend:

```
https://api-a.example.com/users/profile?id=5
atau
https://api-b.example.com/users/profile?id=5
```

---

# 🔄 Reload Config (Manual)

Memuat ulang routes dari database tanpa restart server.

```
GET /reload
```

Dengan header:

```
X-ADMIN-TOKEN: secret123
```

Contoh curl:

```
curl -H "X-ADMIN-TOKEN: secret123" http://localhost:8080/reload
```

Response:

```
Routes reloaded
```

---

# 📊 Lihat Semua Routes & Backend

```
GET /routes
```

Response contoh:

```json
[
  {
    "path": "/users",
    "backends": [
      {
        "id": 1,
        "url": "https://api-a.example.com",
        "alive": true
      },
      {
        "id": 2,
        "url": "https://api-b.example.com",
        "alive": false
      }
    ]
  }
]
```

Disarankan juga diberi header admin token.

---

# ❤️ Health Check

Gateway akan memanggil:

```
GET {backend}/health
```

Backend dianggap sehat jika:

* Response 200 OK

Interval:

```
10 detik
```

---

# ⚖️ Load Balancing

Menggunakan:

* Round Robin
* Hanya backend yang `Alive=true`

Jika semua backend mati →

```
503 No healthy backend
```

---

# 🔒 Keamanan Minimum

Disarankan:

* Gunakan ADMIN_TOKEN
* Letakkan gateway di private network
* Tambahkan firewall

---

# 🚀 Deployment ke Render

1. Push project ke GitHub
2. Render → New Web Service
3. Environment: Go
4. Build Command:

```bash
go build -o app
```

5. Start Command:

```bash
./app
```

6. Tambahkan Environment Variables di dashboard Render

---

# ⚠️ Limitasi

* Konfigurasi hanya di memory
* Jika service restart → reload dari DB
* Belum ada auth per-route
* Belum ada rate limiting

---

# 🛣️ Roadmap (Optional Enhancement)

* Weighted round robin
* Admin CRUD API
* JWT auth per route
* Rate limit
* Metrics Prometheus
* Circuit breaker

---

# 🧪 Tips Development

Reload config setelah edit database:

```bash
curl -H "X-ADMIN-TOKEN: secret123" http://localhost:8080/reload
```

Lihat status backend:

```bash
curl http://localhost:8080/routes
```

---

# 📜 License

MIT / Bebas digunakan untuk keperluan pribadi dan edukasi.

---
