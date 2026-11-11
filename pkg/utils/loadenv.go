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

// accessibile da tutto il progetto come "utils.CurrentVersion"
var CurrentVersion string

// Logger pubblici dedicati (inizializzati sul default per sicurezza anti-panic)
var (
	LogFile    *log.Logger = log.Default()
	LogConsole *log.Logger = log.Default()
)

// Gestisce il caricamento del .env e la configurazione hardware dei Log (Mac vs VPS)
func InizializzaAmbiente() {
	// 1. Forza il tracciamento di data, ora e riga esatta nei log di sistema
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 2. Tenta il caricamento del file .env locale (Mac)
	if err := godotenv.Load(); err != nil {
		// Sulla VPS non ci blocchiamo perché i parametri sono già gestiti in memoria da Systemd
		log.Printf("[INFO] File .env non rilevato o già presente in memoria di sistema")
	}

	// =======================================================
	// VALORIZZAZIONE DELLA VERSIONE DOPO IL CARICAMENTO .ENV
	// =======================================================
	CurrentVersion = os.Getenv("APP_VERSION")

	// 3. DETERMINAZIONE HARDWARE DELL'AMBIENTE (Mac vs VPS Linux)
	_, statErr := os.Stat("/var/www")
	isProduction := statErr == nil

	var logPathGo string
	var logPathNginx string

	if isProduction {
		logPathGo = "/var/log/nginx/pr2056002_go.log"
		logPathNginx = "/var/log/nginx/pr2056002_ssl_error.log"
	} else {
		logPathGo = "development.log"
		logPathNginx = "/opt/homebrew/var/log/nginx/error.log"
	}

	// Forza Go a mettere il percorso in memoria
	os.Setenv("LOG_PATH_GO", logPathGo)
	os.Setenv("LOG_PATH_NGINX", logPathNginx)

	// 4. CONFIGURAZIONE CANALI DI LOG (Unico punto di apertura file)
	logFile, err := os.OpenFile(logPathGo, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("[AVVISO] Impossibile aprire il file di log %s. Output solo su terminale: %v", logPathGo, err)
		return
	}

	// Logger dedicato SOLO al file (development.log su Mac, pr2056002_go.log su VPS)
	LogFile = log.New(logFile, "", log.LstdFlags)

	// Logger dedicato SOLO alla console
	LogConsole = log.New(os.Stdout, "", log.LstdFlags)

	// MultiWriter standard: log.Println / log.Printf scrivono su ENTRAMBI
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
