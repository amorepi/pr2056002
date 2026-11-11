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
	"os"
	"time"
)

// CookieSessionName definisce il nome del cookie di sessione usato dall'applicazione.
const CookieSessionName = "session_token"

// TokenPayload contiene tutte le informazioni utili dell'utente necessarie al progetto senza rieseguire query.
type TokenPayload struct {
	UserID    int64     `json:"user_id"`
	Email     string    `json:"email"`
	Cellulare string    `json:"cellulare"`
	Ruolo     string    `json:"ruolo"`
	Attivo    bool      `json:"attivo"`
	Scadenza  time.Time `json:"scadenza"`
}

// Apri il tuo terminale del Mac e lancia questo comando per generare una stringa casuale esadecimale perfetta di 32 caratteri:
// openssl rand -hex 16
// Assegna il codice risultante a SEGRETO_AES nel file .env (locale) e in Systemd (produzione).
func ottieniChiaveAES() []byte {
	chiave := os.Getenv("SEGRETO_AES")
	if len(chiave) != 32 {
		// Fallback di sicurezza da 32 caratteri esatti per evitare errori a runtime
		return []byte("chiave-segreta-da-32-caratteri!!")
	}
	return []byte(chiave)
}

// GeneraTokenCriptato prende l'intero set di dati utente, lo serializza in JSON e lo cifra con AES-GCM.
func GeneraTokenCriptato(userID int64, email, cellulare, ruolo string, attivo bool, durata time.Duration) (string, error) {
	payload := TokenPayload{
		UserID:    userID,
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

	block, err := aes.NewCipher(ottieniChiaveAES())
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

	block, err := aes.NewCipher(ottieniChiaveAES())
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
// Cookie configurato con queste regole:
// Durata: 24 ore in produzione, 30 giorni in locale.
// Doppia verifica: La scadenza è scritta sia nel browser (Expires / MaxAge del cookie) sia dentro il payload crittografato
// (payload.Scadenza). Anche se un utente modificasse la scadenza del cookie sul proprio browser, ValidaTokenCriptato bloccherebbe
// comunque l'accesso allo scadere del tempo.
// Validità ai riavvii: Essendo un token stateless (i dati sono cifrati direttamente nel cookie senza salvataggio in RAM),
// la sessione sopravvive ai riavvii del server Go. Per invalidarla manualmente prima della scadenza devi cliccare
// su Esci (/logout).
func SetSessionCookie(w http.ResponseWriter, userID int64, email, cellulare, ruolo string, attivo bool) error {
	// Determinazione hardware dell'ambiente per adeguare le policy di sicurezza
	_, statErr := os.Stat("/var/www")
	isProduction := statErr == nil

	var durataSessione time.Duration
	if isProduction {
		durataSessione = 24 * time.Hour // In produzione scade dopo 24 ore
	} else {
		durataSessione = 30 * 24 * time.Hour // In sviluppo locale dura 30 giorni per comodità
	}

	tokenCriptato, err := GeneraTokenCriptato(userID, email, cellulare, ruolo, attivo, durataSessione)
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
		Secure:   isProduction, // false=nessun vincolo; true=invia il cookie solo ed esclusivamente sotto cifratura
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
