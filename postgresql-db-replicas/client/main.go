package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

type User struct {
	ID   int    `json:"id" gorm:"primaryKey"`
	Name string `json:"name"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func dsn(host, port, user, password, dbname string) string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func main() {
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "service")
	writeHost := getEnv("DB_WRITE_HOST", "localhost")
	readHost1 := getEnv("DB_READ_HOST_1", "localhost")
	readHost2 := getEnv("DB_READ_HOST_2", "localhost")
	readPort1 := getEnv("DB_READ_PORT_1", port)
	readPort2 := getEnv("DB_READ_PORT_2", port)

	db, err := gorm.Open(postgres.Open(dsn(writeHost, port, user, password, dbname)), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to write database: %v", err)
	}

	if err := db.Use(dbresolver.Register(dbresolver.Config{
		Replicas: []gorm.Dialector{
			postgres.Open(dsn(readHost1, readPort1, user, password, dbname)),
			postgres.Open(dsn(readHost2, readPort2, user, password, dbname)),
		},
		Policy: dbresolver.RandomPolicy{},
	})); err != nil {
		log.Fatalf("failed to configure read replicas: %v", err)
	}

	log.Printf("connected to write: %s, replicas: %s, %s", writeHost, readHost1, readHost2)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// List all users — routed to a read replica
	mux.HandleFunc("GET /users", func(w http.ResponseWriter, r *http.Request) {
		var users []User
		if result := db.Find(&users); result.Error != nil {
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{result.Error.Error()})
			return
		}
		writeJSON(w, http.StatusOK, users)
	})

	// Create a user — routed to the write primary
	mux.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {
		var user User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil || user.Name == "" {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{"field 'name' is required"})
			return
		}
		if result := db.Create(&user); result.Error != nil {
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{result.Error.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, user)
	})

	// Get a single user by ID — routed to a read replica
	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{"invalid id"})
			return
		}
		var user User
		if result := db.First(&user, id); result.Error != nil {
			writeJSON(w, http.StatusNotFound, ErrorResponse{"user not found"})
			return
		}
		writeJSON(w, http.StatusOK, user)
	})

	// Delete a user by ID — routed to the write primary
	mux.HandleFunc("DELETE /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{"invalid id"})
			return
		}
		if result := db.Delete(&User{}, id); result.Error != nil {
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{result.Error.Error()})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	log.Println("server listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
