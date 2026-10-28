package middleware

import (
	"fmt"
	"net/http"
	"pr2056002/pkg/render" // Importa il motore grafico per disegnare la pagina 500
	"runtime/debug"
)

/*
	Al posto di [Ref.11] - main.go
	"pr2056002/pkg/middleware"

	[Ref.12]
	// 4. Pacchetto middleware: avvolge tutto lo scheletro nello scudo protettivo
	handlerProtetto := middleware.RecoveryMiddleware(mux)

	log.Fatal(http.ListenAndServe(":"+port, handlerProtetto))   // [Ref.13] in main.go
*/

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

				// Altrimenti, rendering dinamico del file 500.html tramite il pacchetto render
				render.RenderTemplate(w, r, "500.html", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
