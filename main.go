package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	// Logica degli endpoint [Ref.01]

	"pr2056002/pkg/middleware"
	"pr2056002/pkg/render"
	"pr2056002/pkg/router"

	// Il "_" obbliga l'inclusione di questo pacchetto anche in assenza di chiamate dirette
	// Driver nativo PostgreSQL
	_ "github.com/lib/pq"

	// Auto-caricamento del file delle variabili d'ambiente .env
	"github.com/joho/godotenv"
)

func main() {
	// 1. Configurazione del tracciamento difensivo dei log (File e Riga esatta)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 2. Caricamento controllato delle variabili d'ambiente (.env)
	if err := godotenv.Load(); err != nil {
		// Non blocchiamo se manca il file (es. sulla VPS sono già in memoria grazie a Systemd)
		log.Printf("INFO: File .env non rilevato sul disco o già caricato in memoria")
	}

	// 3. Recupero della stringa DSN e connessione a PostgreSQL 17
	dbConnStr := os.Getenv("DATABASE_URL")
	if dbConnStr == "" {
		log.Fatal("ERRORE CRITICO: La variabile d'ambiente DATABASE_URL non è impostata nel .env")
	}

	// Apertura della connessione al database PostgreSQL
	db, err := sql.Open("postgres", dbConnStr)
	if err != nil {
		log.Fatalf("ERRORE CRITICO: Inizializzazione del driver fallita: %v", err)
	}
	defer db.Close()

	// Verifica ispettiva della reale raggiungibilità del Database
	if err = db.Ping(); err != nil {
		log.Fatalf("CRITICAL BLINK: Il database non risponde o non esiste nel catalogo PostgreSQL: %v", err)
	}

	// Allineamento della versione globale sul motore grafico isolato
	render.CurrentVersion = os.Getenv("APP_VERSION")

	// ARCHITETTURA DEL ROUTING ESSENZIALE PER LA PRODUZIONE
	mux := http.NewServeMux()

	// Affidiamo interamente la scansione e l'attivazione delle rotte al pacchetto isolato
	router.RegisterRoutes(mux)

	// 3. Avvio dell'ascolto di rete legato al router attivo
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 4. Pacchetto middleware: avvolge tutto lo scheletro nello scudo protettivo
	handlerProtetto := middleware.RecoveryMiddleware(mux)

	fmt.Printf("[START] Fondazione Server Go attiva - Versione: %s\n", os.Getenv("APP_VERSION"))
	fmt.Printf("[START] Connessione a PostgreSQL 17 verificata con successo.\n")
	fmt.Printf("[START] Interfaccia Bootstrap 5 e API operative sulla porta %s...\n", port)

	log.Fatal(http.ListenAndServe(":"+port, handlerProtetto))
}
