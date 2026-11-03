package middleware

import (
	"context"
	"log"
	"net/http"
)

type contextKey string

const UserContextKey contextKey = "user_session"

// Ruoli previsti dall'articolazione del sito
const (
	RuoloAnonimo = "anonimo"
	RuoloUtente  = "user"
	RuoloAdmin   = "admin"
)

// UserSession racchiude lo stato di autorizzazione corrente
type UserSession struct {
	ID          int
	Username    string
	Ruolo       string
	Autenticato bool
}

// SentinellaInerte è il regista di Fase 1. Intercetta ogni richiesta,
// assegna lo stato e arricchisce il contesto per gli handler successivi.
func SentinellaInerte(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// 1. STATO DI DEFAULT: Utente Autonomo (Parte statica pubblica, navigazione anonima)
		sessione := UserSession{
			ID:          0,
			Username:    "Ospite",
			Ruolo:       RuoloAnonimo,
			Autenticato: false,
		}

		// 2. ISPEZIONE DELLA SESSIONE (Cookie)
		// Cerchiamo il cookie per verificare se l'utente si è autenticato
		cookie, err := r.Cookie("pr2056002_session")
		if err == nil {
			// In questa fase simuliamo la decodifica del token o il match dei test:
			switch cookie.Value {
			case "session_token_amorepi":
				sessione = UserSession{ID: 1, Username: "amorepi", Ruolo: RuoloAdmin, Autenticato: true}
			case "session_token_piero":
				sessione = UserSession{ID: 2, Username: "piero", Ruolo: RuoloUtente, Autenticato: true}
			}
		}

		// 3. DIARIO DI LOG SINTETICO (Solo una riga in console/terminale come concordato)
		log.Printf("[SENTINELLA] %s %s -> Ruolo: %s | Auth: %t", r.Method, r.URL.Path, sessione.Ruolo, sessione.Autenticato)

		// 4. ARRICCHIMENTO DEL CONTESTO
		ctx := context.WithValue(r.Context(), UserContextKey, sessione)

		// 5. INOLTRO DELLA RICHIESTA
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
