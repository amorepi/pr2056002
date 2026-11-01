package utils

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"os"
)

type MailerPayload struct {
	Subject string
	Html    string
}

func InviaEmail(payload MailerPayload) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	developerEmail := os.Getenv("DEVELOPER_EMAIL")

	if smtpHost == "" || smtpUser == "" || developerEmail == "" {
		log.Println("[MAILER ERRORE]: Parametri SMTP mancanti nel file .env")
		return fmt.Errorf("configurazione SMTP incompleta")
	}

	// 1. Confezionamento degli Header e del Messaggio
	header := make(map[string]string)
	header["From"] = smtpUser
	header["To"] = developerEmail
	header["Subject"] = payload.Subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = `text/html; charset="UTF-8"`

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + payload.Html

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	tlsconfig := &tls.Config{
		InsecureSkipVerify: true, // Sicurezza flessibile per test locali
		ServerName:         smtpHost,
	}

	// 2. GESTIONE PROTOCOLLO STRATEGICO IN BASE ALLA PORTA
	// Se usi la porta 465, avviamo una connessione TLS nativa diretta (Risolve l'errore EOF)
	if smtpPort == "465" {
		conn, err := tls.Dial("tcp", smtpHost+":"+smtpPort, tlsconfig)
		if err != nil {
			return fmt.Errorf("fallita connessione TLS nativa su porta 465: %w", err)
		}
		defer conn.Close()

		c, err := smtp.NewClient(conn, smtpHost)
		if err != nil {
			return fmt.Errorf("creazione client SMTP fallita: %w", err)
		}
		defer c.Quit()

		if err = c.Auth(auth); err != nil {
			return fmt.Errorf("autenticazione SMTP fallita: %w", err)
		}
		if err = c.Mail(header["From"]); err != nil {
			return err
		}
		if err = c.Rcpt(header["To"]); err != nil {
			return err
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		if _, err = w.Write([]byte(message)); err != nil {
			return err
		}
		return w.Close()
	}

	// 3. FALLBACK STANDARD PER PORTA 587 (Connessione Standard + StartTLS)
	conn, err := smtp.Dial(smtpHost + ":" + smtpPort)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err = conn.StartTLS(tlsconfig); err != nil {
		return err
	}
	if err = conn.Auth(auth); err != nil {
		return err
	}
	if err = conn.Mail(header["From"]); err != nil {
		return err
	}
	if err = conn.Rcpt(header["To"]); err != nil {
		return err
	}
	w, err := conn.Data()
	if err != nil {
		return err
	}
	if _, err = w.Write([]byte(message)); err != nil {
		return err
	}
	return w.Close()
}
