package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"setupforme/database"
	"setupforme/handlers"
	"setupforme/middleware"
)

func main() {
	// Initialize database
	db, err := database.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Initialize handlers with database
	authHandler := handlers.NewAuthHandler(db)
	appHandler := handlers.NewAppHandler(db)

	// Setup routes
	mux := http.NewServeMux()

	// Auth routes
	mux.HandleFunc("POST /api/auth/signup", authHandler.Signup)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)

	// Winget search route (unauthenticated is fine for suggestions)
	mux.HandleFunc("GET /api/winget/search", handlers.WingetSearchHandler)

	// Protected app routes
	mux.Handle("GET /api/apps", middleware.AuthMiddleware(http.HandlerFunc(appHandler.GetApps)))
	mux.Handle("POST /api/apps", middleware.AuthMiddleware(http.HandlerFunc(appHandler.CreateApp)))
	mux.Handle("PUT /api/apps/{id}", middleware.AuthMiddleware(http.HandlerFunc(appHandler.UpdateApp)))
	mux.Handle("DELETE /api/apps/{id}", middleware.AuthMiddleware(http.HandlerFunc(appHandler.DeleteApp)))
	mux.Handle("GET /api/apps/script", middleware.AuthMiddleware(http.HandlerFunc(appHandler.GenerateScript)))

	// CORS middleware
	handler := middleware.CORSMiddleware(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
