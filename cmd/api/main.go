// Entrypoint aplikasi. Tugasnya: load config/env, buka koneksi DB,
// inisialisasi router + middleware, lalu start HTTP server.
// Logic bisnis TIDAK ditaruh di sini — itu tugas internal/service.
package main

func main() {
	// TODO: load .env (godotenv)
	// TODO: connect ke PostgreSQL (pgxpool)
	// TODO: setup router (chi/fiber) + middleware (CORS, logger)
	// TODO: register routes dari internal/handler
	// TODO: start server, listen di PORT dari env
}
