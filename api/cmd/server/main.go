// Package main is the entry point for X-Ray API server.
//
// @title X-Ray API
// @version 1.0
// @description Reasoning-based observability API for multi-step decision pipelines
//
// @contact.name X-Ray Team
// @license.name MIT
//
// @host localhost:8080
// @BasePath /api/v1
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
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/xray-sdk/xray-api/docs" // Swagger docs
	"github.com/xray-sdk/xray-api/internal/handlers"
	"github.com/xray-sdk/xray-api/internal/store"
	"github.com/xray-sdk/xray-api/internal/store/dynamodb"
)

func main() {
	// Configuration from environment
	port := getEnv("PORT", "8080")
	dynamoEndpoint := getEnv("DYNAMODB_ENDPOINT", "")
	dynamoTable := getEnv("DYNAMODB_TABLE", "xray_data")
	awsRegion := getEnv("AWS_REGION", "us-east-1")

	ctx := context.Background()

	// Initialize store
	var dataStore store.Store
	var err error

	dataStore, err = dynamodb.New(ctx, dynamodb.Config{
		TableName: dynamoTable,
		Endpoint:  dynamoEndpoint,
		Region:    awsRegion,
	})
	if err != nil {
		log.Fatalf("Failed to initialize DynamoDB store: %v", err)
	}
	defer dataStore.Close()

	// Initialize handlers
	ingestHandler := handlers.NewIngestHandler(dataStore)
	queryHandler := handlers.NewQueryHandler(dataStore)

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Health check
	r.Get("/health", queryHandler.Health)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Trace endpoints
		r.Route("/traces", func(r chi.Router) {
			r.Post("/", ingestHandler.CreateTrace)
			r.Post("/batch", ingestHandler.BatchCreateTraces)
			r.Get("/", queryHandler.QueryTraces)

			r.Route("/{traceId}", func(r chi.Router) {
				r.Get("/", queryHandler.GetTrace)
				r.Patch("/", ingestHandler.UpdateTrace)
				r.Get("/events", queryHandler.GetEventsByTrace)

				r.Route("/events/{eventId}", func(r chi.Router) {
					r.Get("/", queryHandler.GetEvent)
					r.Get("/decisions", queryHandler.GetDecisionsByEvent)
				})
			})
		})

		// Event endpoints
		r.Route("/events", func(r chi.Router) {
			r.Post("/", ingestHandler.CreateEvent)
			r.Post("/batch", ingestHandler.BatchCreateEvents)
		})

		// Decision endpoints
		r.Route("/decisions", func(r chi.Router) {
			r.Post("/", ingestHandler.CreateDecision)
			r.Post("/batch", ingestHandler.BatchCreateDecisions)
		})

		// Item history
		r.Get("/items/{itemId}/history", queryHandler.GetItemHistory)

		// Query endpoint
		r.Route("/query", func(r chi.Router) {
			r.Post("/", queryHandler.Query)
			r.Get("/events", queryHandler.QueryEvents)
		})
	})

	// Start server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Printf("X-Ray API server starting on port %s", port)
		log.Printf("Swagger UI available at http://localhost:%s/swagger/index.html", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
