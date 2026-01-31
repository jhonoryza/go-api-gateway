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
 ├─ logger.go
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

- init.sql
- logger.sql

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
go run .
```

Output:

```
Gateway running on :8080
```

---

# 🏗️ Build Binary

```bash
go build -o build/gateway
```

Jalankan:

```bash
./build/gateway
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

# 📊 API DOC

cek file `openapi.json`

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

# ⚠️ Limitasi

* Konfigurasi hanya di memory
* Jika service restart → reload dari DB
* Belum ada rate limiting

---

# 🧪 Tips Development

Health check:

```bash
curl http://localhost:8080/health
curl http://localhost:8080
```

Reload config setelah edit database:

```bash
curl -H "X-ADMIN-TOKEN: secret123" http://localhost:8080/reload
```

Lihat status setiap routes:

```bash
curl -H "X-ADMIN-TOKEN: secret123" http://localhost:8080/routes
```

Lihat request log:

```bash
curl -H "X-ADMIN-TOKEN: secret123" http://localhost:8080/logs
```

---

# 📜 License

MIT / Bebas digunakan untuk keperluan pribadi dan edukasi.

---
