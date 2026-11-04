package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"pr2056002/pkg/dashboard"
	"pr2056002/pkg/render"

	"golang.org/x/crypto/bcrypt"
)

// BootstrapDBHandler gestisce la creazione della tabella utenti e l'auto-configurazione del file .env
func BootstrapDBHandler(w http.ResponseWriter, r *http.Request) {
	db := dashboard.GetDatabase()

	// 1. BLOCCO LOGICO AUTOMATICO: Controlliamo l'effettiva presenza fisica nel DB
	var esiste bool
	queryCheck := `SELECT EXISTS (
		SELECT FROM information_schema.tables 
		WHERE table_schema = 'public' AND table_name = 'users'
	);`

	_ = db.QueryRow(queryCheck).Scan(&esiste)

	if esiste {
		log.Println("[BOOTSTRAP] Accesso negato: la tabella 'users' esiste già nel database.")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// 2. GESTIONE RICHIESTA GET
	if r.Method == http.MethodGet {
		data := map[string]interface{}{
			"ProjectName": "PR2056002",
			"Version":     render.CurrentVersion,
			"LogoUrl":     "/img/" + os.Getenv("LOGO_PROJ"),
			"AvatarUrl":   "/img/" + os.Getenv("AVATAR_PROJ"),
			"UserName":    "Ospite",
			"Autenticato": false, // Cambia in true se il cookie di sessione è valido
		}

		tmpl, err := template.ParseFiles("html/templates/layout.html", "html/templates/bootstrap.html")
		if err != nil {
			log.Printf("[BOOTSTRAP ERRORE] Caricamento template fallito: %v", err)
			http.Error(w, "Errore caricamento template bootstrap", http.StatusInternalServerError)
			return
		}

		_ = tmpl.ExecuteTemplate(w, "layout", data)
		return
	}

	// 3. GESTIONE RICHIESTA POST
	if r.Method == http.MethodPost {
		passwordInChiaro := r.FormValue("password")
		if len(passwordInChiaro) < 8 {
			http.Error(w, "La password deve essere di almeno 8 caratteri", http.StatusBadRequest)
			return
		}

		hashBytes, err := bcrypt.GenerateFromPassword([]byte(passwordInChiaro), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Errore generazione hash", http.StatusInternalServerError)
			return
		}
		stringaHash := string(hashBytes)

		// APERTURA TRANSAZIONE ATOMICA SU POSTGRESQL 17
		tx, err := db.Begin()
		if err != nil {
			log.Printf("[BOOTSTRAP ERRORE] Impossibile avviare la transazione: %v", err)
			http.Error(w, "Errore interno database", http.StatusInternalServerError)
			return
		}
		// In caso di panico o fallimento, esegue il rollback sicuro delle modifiche
		defer tx.Rollback()

		// A. CREAZIONE TABELLA
		ddl := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(50) UNIQUE NOT NULL,
			email VARCHAR(100) UNIQUE NOT NULL,
			cellulare VARCHAR(20) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			ruolo VARCHAR(20) NOT NULL DEFAULT 'user',
			attivo BOOLEAN NOT NULL DEFAULT TRUE,
			creato_il TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX idx_users_username ON users(username);
		CREATE INDEX idx_users_cellulare ON users(cellulare);
		`
		_, err = tx.Exec(ddl)
		if err != nil {
			log.Printf("[BOOTSTRAP ERRORE] Creazione tabella fallita: %v", err)
			http.Error(w, "Errore creazione tabella utenti: "+err.Error(), http.StatusInternalServerError)
			return
		}

		cellulareAdmin := os.Getenv("CELLULARE_ADMIN")
		if cellulareAdmin == "" {
			cellulareAdmin = "+393381334545"
		}

		// B. INSERIMENTO OPERATORE ADMIN
		queryInsertAdmin := `
		INSERT INTO users (username, email, cellulare, password_hash, ruolo, attivo) 
		VALUES ($1, $2, $3, $4, $5, $6);`

		_, err = tx.Exec(queryInsertAdmin, "amorepi", "piero@ingamore.it", cellulareAdmin, stringaHash, "admin", true)
		if err != nil {
			log.Printf("[BOOTSTRAP ERRORE] Inserimento admin supremo fallito: %v", err)
			http.Error(w, "Errore popolamento amministratore: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// COMMIT DELLA TRANSAZIONE: Scrittura definitiva su disco del DB
		if err := tx.Commit(); err != nil {
			log.Printf("[BOOTSTRAP ERRORE] Commit fallito: %v", err)
			http.Error(w, "Errore salvataggio dati", http.StatusInternalServerError)
			return
		}

		// C. AGGIORNAMENTO FILE .ENV FISICO (Solo a valle del successo sul DB)
		fileEnv, err := os.OpenFile(".env", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err == nil {
			_, _ = fileEnv.WriteString(fmt.Sprintf("\n# Configurazione generata automaticamente al bootstrap\nBOOTSTRAP_HASH=\"%s\"\n", stringaHash))
			fileEnv.Close()
		}

		log.Println("[BOOTSTRAP] Inizializzazione completata con successo. Tabella creata ed utente 'amorepi' inserito.")
		http.Redirect(w, r, "/login?inizializzazione=successo", http.StatusSeeOther)
	}
}
