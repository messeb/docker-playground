package main

import (
	"encoding/json"
	"net/http"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type UserResponse struct {
	AffectedRows int	`json:"affected_rows"`
	Users        []User `json:"users"`
}

type UserCreatedResponse struct {
	User User `json:"user"`
}


func main() {
	db, _ := gorm.Open(postgres.Open("host=localhost port=5432 user=postgres password=postgres dbname=service"), &gorm.Config{})

	db.Use(dbresolver.Register(dbresolver.Config{
		Replicas: []gorm.Dialector{
			postgres.Open("host=localhost port=5433 user=postgres password=postgres dbname=service"),
			postgres.Open("host=localhost port=5434 user=postgres password=postgres dbname=service"),
		},
		Policy: dbresolver.RandomPolicy{},
	}))


	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			var user User

			json.NewDecoder(r.Body).Decode(&user)
			db.Create(&user)

			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(user)

		case "GET":
			var users []User
			result := db.Find(&users)

			response := UserResponse{
				AffectedRows: int(result.RowsAffected),
				Users:        users,
			}

			w.Header().Add("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}
	})

	http.ListenAndServe(":8080", nil)
}
