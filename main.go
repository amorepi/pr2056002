package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"pr2056002/pkg/dashboard"
	"pr2056002/pkg/middleware"
	"pr2056002/pkg/render"
	"pr2056002/pkg/router"
	"pr2056002/pkg/utils"

	// Il "_" obbliga l'inclusione di questo pacchetto anche in assenza di chiamate dirette
	// Driver nativo PostgreSQL
	_ "github.com/lib/pq"
)

func main() {
	utils.InizializzaAmbiente()

	// 1. Recupero della stringa DSN e connessione a PostgreSQL 17
	dbConnStr := os.Getenv("DATABASE_URL")
	if dbConnStr == "" {
		log.Fatal("ERRORE CRITICO: La variabile d'ambiente DATABASE_URL non è impostata")
	}

	db, err := sql.Open("postgres", dbConnStr)
	if err != nil {
		log.Fatalf("ERRORE CRITICO: Inizializzazione del driver fallita: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalf("CRITICAL BLINK: Il database non risponde o non esiste nel catalogo PostgreSQL: %v", err)
	}

	// 2. Allineamento della versione globale sul motore grafico isolato
	render.CurrentVersion = os.Getenv("APP_VERSION")

	// Passa la connessione al package dashboard
	dashboard.SetDatabase(db)

	// 3. ARCHITETTURA DEL ROUTING RIGENERATA
	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	// 4. Avvio dell'ascolto di rete legato al router attivo
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 5. Pacchetto middleware: avvolge tutto lo scheletro nello scudo protettivo
	handlerProtetto := middleware.RecoveryMiddleware(mux)

	fmt.Printf("[START] Operativo sulla porta %s...\n", port)

	log.Fatal(http.ListenAndServe(":"+port, handlerProtetto))
}
