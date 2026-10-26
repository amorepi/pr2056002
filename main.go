package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	// Logica degli endpoint e dei middleware
	"pr2056002/pkg/middleware"
	"pr2056002/pkg/render"
	"pr2056002/restapi"

	// Il "_" obbliga l'inclusione di questo pacchetto anche in assenza di chiamate dirette
	// Driver nativo PostgreSQL
	_ "github.com/lib/pq"

	// Auto-caricamento del file delle variabili d'ambiente .env
	"github.com/joho/godotenv"
)

// Version riceve il valore in fase di compilazione remota via ldflags
var Version = "working-progress"

func main() {
	// 1. Configurazione del tracciamento difensivo dei log (File e Riga esatta)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 2. Caricamento controllato delle variabili d'ambiente (.env)
	if err := godotenv.Load(); err != nil {
		// Non blocchiamo se manca il file (es. sulla VPS sono già in memoria grazie a Systemd)
		log.Printf("INFO: File .env non rilevato sul disco o già caricato in memoria")
	}

	// Allineamento dinamico della stringa della versione visiva
	if Version == "working-progress" {
		envVersion := os.Getenv("APP_VERSION")
		if envVersion != "" {
			Version = envVersion
		}
	}

	// Passiamo in sicurezza il valore della versione al pacchetto restapi prima di avviare il server
	restapi.CurrentVersion = Version

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

	// Iniezione della connessione DB attiva all'interno del modulo restapi
	restapi.SetDatabase(db)

	// Allineamento della versione globale sul motore grafico isolato
	render.CurrentVersion = Version

	// ARCHITETTURA DEL ROUTING CON COMPONENTI GLOBALMENTE DISTINTI
	mux := http.NewServeMux()

	// 1. Pacchetto restapi: si occupa SOLO degli scambi dati JSON delle API
	mux.HandleFunc("/api/users", restapi.UsersHandler)
	mux.HandleFunc("/api/health", restapi.HealthHandler)

	// 2. Pacchetto statico dei file asset
	mux.Handle("/html/", http.StripPrefix("/html/", http.FileServer(http.Dir("./html"))))
	mux.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("./html/img"))))

	// 3. Pacchetto render: intercetta la radice o smista i 404 grafici
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			render.RenderTemplate(w, r, "home.html", http.StatusOK)
			return
		}
		render.Error404Handler(w, r)
	})

	// 4. Pacchetto middleware: avvolge tutto lo scheletro nello scudo protettivo
	handlerProtetto := middleware.RecoveryMiddleware(mux)

	// 6. Avvio dell'ascolto di rete
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("[START] Fondazione Server Go attiva - Versione: %s\n", Version)
	fmt.Printf("[START] Connessione a PostgreSQL 17 verificata con successo.\n")
	fmt.Printf("[START] Interfaccia Bootstrap 5 e API operative sulla porta %s...\n", port)

	log.Fatal(http.ListenAndServe(":"+port, handlerProtetto))
}
