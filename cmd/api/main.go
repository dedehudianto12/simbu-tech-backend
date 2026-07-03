package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/dedehudianto12/simbu-tech-backend/internal/handler"
	"github.com/dedehudianto12/simbu-tech-backend/internal/middleware"
	"github.com/dedehudianto12/simbu-tech-backend/internal/repository"
	"github.com/dedehudianto12/simbu-tech-backend/internal/service"
)

func main() {
	_ = godotenv.Load()

	db := connectDB()
	defer db.Close()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-production"
	}

	ticketRepo := repository.NewTicketRepo(db)
	userRepo := repository.NewUserRepo(db)

	notifSvc := service.NewNotificationService(service.NotificationConfig{})
	ticketSvc := service.NewTicketService(ticketRepo, notifSvc)
	authSvc := service.NewAuthService(userRepo, jwtSecret, 15*time.Minute, 7*24*time.Hour)

	publicHandler := handler.NewPublicTicketHandler(ticketSvc)
	adminHandler := handler.NewAdminTicketHandler(ticketSvc)
	authHandler := handler.NewAuthHandler(authSvc)

	r := chi.NewRouter()
	r.Use(middleware.CORS)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	// Auth routes (no JWT middleware)
	r.Post("/api/admin/auth/login", authHandler.Login)
	r.Post("/api/admin/auth/refresh", authHandler.RefreshToken)

	// Public routes
	r.Route("/api/public", func(r chi.Router) {
		r.Get("/health", healthHandler)
		r.Post("/tickets", publicHandler.CreateTicket)
		r.Get("/tickets/{ticketNumber}", publicHandler.GetTicketByNumber)
	})

	// Admin routes (JWT protected)
	r.Route("/api/admin", func(r chi.Router) {
		r.Use(middleware.JWTAuth(jwtSecret))
		r.Get("/health", healthHandler)
		r.Get("/tickets", adminHandler.ListTickets)
		r.Get("/tickets/{id}", adminHandler.GetTicket)
		r.Patch("/tickets/{id}", adminHandler.UpdateTicket)
		r.Post("/tickets/{id}/comments", adminHandler.AddComment)
		r.Get("/dashboard/stats", adminHandler.GetStats)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{Addr: ":" + port, Handler: r}

	go func() {
		log.Printf("server running on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	srv.Shutdown(context.Background())
}

func connectDB() *pgxpool.Pool {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("unable to create connection pool: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("unable to ping database: %v", err)
	}
	log.Println("connected to database")
	return pool
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"ok"}`))
}
