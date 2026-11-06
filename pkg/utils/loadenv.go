package utils

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"

	// Auto-caricamento del file delle variabili d'ambiente .env
	"github.com/joho/godotenv"
)

// InizializzaAmbiente gestisce il caricamento del .env e la configurazione hardware dei Log (Mac vs VPS)
func InizializzaAmbiente() {
	// 1. Forza il tracciamento di data, ora e riga esatta nei log di sistema
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 2. Tenta il caricamento del file .env locale (Mac)
	if err := godotenv.Load(); err != nil {
		// Sulla VPS non ci blocchiamo perché i parametri sono già gestiti in memoria da Systemd
		log.Printf("[INFO] File .env non rilevato o già presente in memoria di sistema")
	}

	// 3. DETERMINAZIONE HARDWARE DELL'AMBIENTE (Mac vs VPS Linux)
	// Controlliamo l'esistenza inconfutabile della cartella /var/www
	_, statErr := os.Stat("/var/www")
	isProduction := statErr == nil

	var logPathGo string
	var logPathNginx string

	if isProduction {
		logPathGo = "/var/log/nginx/pr2056002_go.log"
		logPathNginx = "/var/log/nginx/pr2056002_ssl_error.log"
	} else {
		// Assicuriamoci che si chiami esattamente development.log
		logPathGo = "development.log"
		logPathNginx = "/opt/homebrew/var/log/nginx/error.log"
	}

	// Questa istruzione è fondamentale! Forza Go a mettere il percorso in memoria
	os.Setenv("LOG_PATH_GO", logPathGo)
	os.Setenv("LOG_PATH_NGINX", logPathNginx)

	// 4. DIROTTARE I LOG STANDARD SU MULTI-WRITER (Terminale + File)
	logFile, err := os.OpenFile(logPathGo, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("[AVVISO] Impossibile aprire il file di log %s. Output solo su terminale: %v", logPathGo, err)
		return
	}

	// Il MultiWriter permette a log.Printf di stampare sia a schermo che nel file .log
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}

// ConnettiDatabase apre la connessione e verifica se il catalogo PostgreSQL risponde
func ConnettiDatabase(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("inizializzazione del driver fallita: %w", err)
	}

	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("il database non risponde o non esiste nel catalogo PostgreSQL: %w", err)
	}

	return db, nil
}
