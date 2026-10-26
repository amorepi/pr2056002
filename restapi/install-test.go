package restapi

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// User definisce la struttura minimale degli utenti
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var (
	db             *sql.DB
	CurrentVersion string
)

func SetDatabase(activeDB *sql.DB) {
	db = activeDB
}

// UsersHandler orchestra le operazioni di GET e POST sulla tabella PostgreSQL
func UsersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if db == nil {
		http.Error(w, "Connessione al database non inizializzata", http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query("SELECT id, name, email FROM users ORDER BY id ASC")
		if err != nil {
			http.Error(w, "Errore durante l'interrogazione del database", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var currentUsers []User
		for rows.Next() {
			var u User
			if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
				continue
			}
			currentUsers = append(currentUsers, u)
		}

		jsonOutput, _ := json.Marshal(currentUsers)
		fmt.Printf("[GET DB SUCCESS] Richiesta completata. Record trasmessi: %s\n", string(jsonOutput))
		w.Write(jsonOutput)

	case http.MethodPost:
		var newUser User
		if err := json.NewDecoder(r.Body).Decode(&newUser); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Inserimento del record con ritorno dell'ID incrementale gestito da Postgres
		query := "INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id"
		err := db.QueryRow(query, newUser.Name, newUser.Email).Scan(&newUser.ID)
		if err != nil {
			http.Error(w, "Impossibile inserire l'utente (Email duplicata o non valida)", http.StatusConflict)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(newUser)

		fmt.Printf("[POST DB SUCCESS] Scrittura eseguita: ID=%d, Name=%s, Email=%s\n", newUser.ID, newUser.Name, newUser.Email)

	default:
		http.Error(w, "Metodo non consentito", http.StatusMethodNotAllowed)
	}
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if db == nil || db.Ping() != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf(`{"status":"DOWN","engine":"PostgreSQL Relational Disconnected","version":"%s"}`, CurrentVersion)))
		return
	}

	// Recupero dinamico: se CurrentVersion è impostata dal linker usiamo quella,
	// altrimenti cerchiamo la variabile d'ambiente APP_VERSION valorizzata nel file .env
	displayVersion := CurrentVersion
	if displayVersion == "" || displayVersion == "working-progress" {
		displayVersion = os.Getenv("APP_VERSION")
		if displayVersion == "" {
			displayVersion = "v1.0.0" // Fallback di sicurezza per la prima installazione
		}
	}

	w.Write([]byte(fmt.Sprintf(`{"status":"UP","engine":"PostgreSQL Real Database","version":"%s"}`, displayVersion)))
}
