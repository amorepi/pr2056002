package middleware

import (
	"fmt"
	"net/http"
	"pr2056002/pkg/render"
	"pr2056002/pkg/users" // <-- Aggiunto l'import del pacchetto users
	"runtime/debug"
)

// RecoveryMiddleware intercetta i panic e inietta l'errore 500 grafico o JSON
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				stackTrace := string(debug.Stack())
				fmt.Printf("CRITICAL EXCEPTION: %v\n%s\n", err, stackTrace)

				w.WriteHeader(http.StatusInternalServerError)

				// Se la richiesta è API, rispondi con un JSON pulito
				if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(`{"success":false,"error":"Errore interno del server"}`))
					return
				}

				// Genera la mappa dati base per la navbar della pagina 500
				dataEmergenza := users.GeneraMappaDatiBase(r)

				// Rendering dinamico del file 500.html passando l'interfaccia data richiesta
				render.RenderTemplate(w, r, "500.html", http.StatusInternalServerError, dataEmergenza)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
