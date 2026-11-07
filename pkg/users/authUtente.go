package users

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

// CookieSessionName definisce il nome del cookie di sessione usato dall'applicazione.
const CookieSessionName = "session_token"

// TokenPayload contiene tutte le informazioni utili dell'utente necessarie al progetto senza rieseguire query.
type TokenPayload struct {
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Cellulare string    `json:"cellulare"`
	Ruolo     string    `json:"ruolo"`
	Attivo    bool      `json:"attivo"`
	Scadenza  time.Time `json:"scadenza"`
}

// ATTENZIONE: In produzione questa chiave deve essere caricata da variabili d'ambiente
var segretoAES = []byte("chiave-segreta-da-32-caratteri!!")

// GeneraTokenCriptato prende l'intero set di dati utente, lo serializza in JSON e lo cifra con AES-GCM.
func GeneraTokenCriptato(userID int64, username, email, cellulare, ruolo string, attivo bool, durata time.Duration) (string, error) {
	payload := TokenPayload{
		UserID:    userID,
		Username:  username,
		Email:     email,
		Cellulare: cellulare,
		Ruolo:     ruolo,
		Attivo:    attivo,
		Scadenza:  time.Now().Add(durata),
	}

	plaintext, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(segretoAES)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// ValidaTokenCriptato decodifica, decifra e verifica se il token è ancora valido.
func ValidaTokenCriptato(tokenString string) (*TokenPayload, error) {
	ciphertext, err := base64.URLEncoding.DecodeString(tokenString)
	if err != nil {
		return nil, errors.New("token non valido: errore decodifica base64")
	}

	block, err := aes.NewCipher(segretoAES)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("token non valido: dimensione insufficiente")
	}

	nonce, pureCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := aesGCM.Open(nil, nonce, pureCiphertext, nil)
	if err != nil {
		return nil, errors.New("token non valido: fallita decifratura o firma non autentica")
	}

	var payload TokenPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return nil, errors.New("token non valido: errore parsing dati")
	}

	if time.Now().After(payload.Scadenza) {
		return nil, errors.New("token non valido: credenziali scadute")
	}

	return &payload, nil
}

// SetSessionCookie impacchetta tutte le specifiche dell'utente nel cookie crittografato AES-GCM.
func SetSessionCookie(w http.ResponseWriter, userID int64, username, email, cellulare, ruolo string, attivo bool) error {
	durataSessione := 24 * time.Hour

	tokenCriptato, err := GeneraTokenCriptato(userID, username, email, cellulare, ruolo, attivo, durataSessione)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     CookieSessionName,
		Value:    tokenCriptato,
		Path:     "/",
		Expires:  time.Now().Add(durataSessione),
		MaxAge:   int(durataSessione.Seconds()),
		HttpOnly: true,
		Secure:   false, // Imposta a true se usi HTTPS in produzione
		SameSite: http.SameSiteStrictMode,
	})

	return nil
}

// GetSessionPayload estrae l'intera anagrafica decifrata dal cookie. Comoda per inviare Email/SMS/WhatsApp nelle altre rotte.
func GetSessionPayload(r *http.Request) (*TokenPayload, error) {
	cookie, err := r.Cookie(CookieSessionName)
	if err != nil {
		return nil, err
	}
	return ValidaTokenCriptato(cookie.Value)
}

// RichiedeRuoli è un middleware che blocca l'accesso se l'utente non è autenticato
// o se non possiede uno dei ruoli autorizzati passati come argomento.
func RichiedeRuoli(ruoliAmmessi ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Estraiamo il payload dal cookie
			payload, err := GetSessionPayload(r)
			if err != nil {
				// Utente non autenticato: blocco immediato con 401 Unauthorized
				http.Error(w, "Accesso non autorizzato: autenticazione richiesta.", http.StatusUnauthorized)
				return
			}

			// 2. Verifichiamo se l'account è ancora attivo nel token
			if !payload.Attivo {
				http.Error(w, "Accesso non autorizzato: account disabilitato.", http.StatusForbidden)
				return
			}

			// 3. Controlliamo se il ruolo dell'utente è tra quelli ammessi
			ruoloAutorizzato := false
			for _, ruolo := range ruoliAmmessi {
				if payload.Ruolo == ruolo {
					ruoloAutorizzato = true
					break
				}
			}

			if !ruoloAutorizzato {
				// L'utente è loggato ma non ha il ruolo corretto (es. utente normale su /dashboard)
				http.Error(w, "Accesso non autorizzato: permessi insufficienti.", http.StatusForbidden)
				return
			}

			// 4. Se tutto è valido, proseguiamo con l'handler della pagina
			next.ServeHTTP(w, r)
		})
	}
}
