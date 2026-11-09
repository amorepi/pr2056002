package initialize

import (
	"database/sql"
	"fmt"
	"os"

	"pr2056002/pkg/utils"

	"golang.org/x/crypto/bcrypt"
)

// AutoBootstrapDB controlla l'esistenza della tabella 'users' ed esegue il setup strutturale
func AutoBootstrapDB(db *sql.DB) error {
	// 1. BLOCCO LOGICO AUTOMATICO: Controlliamo l'effettiva presenza fisica nel DB
	var esiste bool
	queryCheck := `SELECT EXISTS (
		SELECT FROM information_schema.tables 
		WHERE table_schema = 'public' AND table_name = 'users'
	);`

	err := db.QueryRow(queryCheck).Scan(&esiste)
	if err != nil {
		return fmt.Errorf("errore durante il controllo della tabella users: %w", err)
	}

	// Se la tabella esiste già, usciamo silenziosamente con successo
	if esiste {
		return nil
	}

	utils.LogFile.Println("[BOOTSTRAP] ATTENZIONE: Tabella 'users' mancante. Avvio procedura di inizializzazione d'emergenza...")

	// Recuperiamo la password iniziale da inserire dal file .env per evitare credenziali cablate fisse
	passwordInChiaro := os.Getenv("ADMIN_INITIAL_PASSWORD")
	if len(passwordInChiaro) < 8 {
		// Fallback di emergenza se ti sei dimenticato di configurare il .env
		passwordInChiaro = "041913150201110351608588"
		utils.LogFile.Printf("[BOOTSTRAP WARNING] ADMIN_INITIAL_PASSWORD non trovata o troppo corta nel .env. Utilizzo credenziale di fallback temporanea.")
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(passwordInChiaro), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("errore generazione hash della password amministratore: %w", err)
	}
	stringaHash := string(hashBytes)

	// APERTURA TRANSAZIONE ATOMICA SU POSTGRESQL 17
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("impossibile avviare la transazione PostgreSQL: %w", err)
	}
	defer tx.Rollback()

	// A. CREAZIONE TABELLA
	ddl := `
	CREATE TABLE users (
		id SERIAL PRIMARY KEY,
		cognome VARCHAR(100) NOT NULL,
		nome VARCHAR(100) NOT NULL,
		email VARCHAR(100) UNIQUE NOT NULL,
		cellulare VARCHAR(20) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		ruolo VARCHAR(20) NOT NULL DEFAULT 'user',
		attivo BOOLEAN NOT NULL DEFAULT TRUE,
		creato_il TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX idx_users_email ON users(email);
	CREATE INDEX idx_users_cellulare ON users(cellulare);
	`
	_, err = tx.Exec(ddl)
	if err != nil {
		return fmt.Errorf("creazione tabelle e indici fallita: %w", err)
	}

	cellulareAdmin := os.Getenv("CELLULARE_ADMIN")
	if cellulareAdmin == "" {
		cellulareAdmin = "+393381334545"
	}

	// B. INSERIMENTO OPERATORE ADMIN
	queryInsertAdmin := `
	INSERT INTO users (cognome, nome, email, cellulare, password_hash, ruolo, attivo) 
	VALUES ($1, $2, $3, $4, $5, $6, $7);`

	_, err = tx.Exec(queryInsertAdmin, "amore", "pietro", "piero@ingamore.it", cellulareAdmin, stringaHash, "admin", true)
	if err != nil {
		return fmt.Errorf("popolamento utente amministratore fallito: %w", err)
	}

	// Salvataggio definitivo su DB
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transazione di bootstrap fallito: %w", err)
	}

	// C. AGGIORNAMENTO FILE .ENV FISICO (Manteniamo la tua traccia storica dell'hash generato)
	fileEnv, err := os.OpenFile(".env", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err == nil {
		_, _ = fileEnv.WriteString(fmt.Sprintf("\n# Configurazione generata automaticamente al bootstrap strutturale\nBOOTSTRAP_HASH=\"%s\"\n", stringaHash))
		fileEnv.Close()
	}

	utils.LogFile.Println("[BOOTSTRAP] Inizializzazione completata con successo! Tabella users generata e amministratore 'piero@ingamore.it' censito correttamente.")
	return nil
}
