# Simbu Ticketing API

Backend monolith untuk sistem ticketing/helpdesk PT Simbu Teknologi.
Stack: Go + PostgreSQL.

## Struktur Folder

- `cmd/api` — entrypoint aplikasi (`main.go`)
- `internal/handler` — HTTP handler (controller layer), parsing request & response
- `internal/service` — business logic (generate ticket number, hitung SLA, dll)
- `internal/repository` — akses database, query langsung ke PostgreSQL
- `internal/middleware` — JWT auth (admin only), CORS
- `internal/model` — struct domain (Ticket, User, dll)
- `pkg/jwt` — helper generate/verifikasi JWT, reusable
- `pkg/validator` — helper validasi input
- `migrations` — file SQL migration

## Keputusan Arsitektur (v1)

- Monolith, bukan microservice.
- Public endpoint TIDAK perlu login — cukup `ticket_number` untuk cek status.
  `ticket_number` di-generate RANDOM (bukan sequential) supaya tidak bisa ditebak.
- Notifikasi: email only di v1 (WhatsApp ditunda untuk versi berikutnya).
- Admin endpoint (dashboard internal) pakai JWT auth.

## Setup

1. `cp .env.example .env` lalu isi nilai aslinya
2. `go mod tidy`
3. Jalankan migration (lihat folder `migrations/`)
4. `go run cmd/api/main.go`

## TODO

- [ ] Implementasi koneksi DB di main.go
- [ ] Implementasi router + middleware
- [ ] Implementasi handler & service untuk create/get ticket
- [ ] Implementasi auth (login admin)
- [ ] Implementasi notification service (email)
