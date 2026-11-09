package utils

import (
	"log"
	"os"
	"runtime"
	"strings"
)

// Flag globale che stabilisce se il tracciamento è attivo (impostato al boot)
var FlussoAttivo bool = false

// controlla se è stato passato l'argomento "true" da terminale
func InizializzaFlusso() {
	// os.Args[0] è sempre il percorso dell'eseguibile. os.Args[1] è il primo argomento vero.
	if len(os.Args) > 1 && os.Args[1] == "true" {
		FlussoAttivo = true
		log.Println("[INFO] 🚀 MODALITÀ TRACCIAMENTO FLUSSO ATTIVATA")
	}
}

// Traccia inserita all'inizio di una funzione per registrare chi l'ha chiamata e da quale riga
func Traccia() {
	// Se non siamo in modalità debug, esce istantaneamente a costo zero
	if !FlussoAttivo {
		return
	}

	// Recuperiamo le informazioni sul punto in cui è stata invocata la funzione utils.Traccia()
	pcCorrente, _, _, okCorrente := runtime.Caller(1)

	// Recuperiamo le informazioni su CHI ha chiamato la funzione precedente (riga e file di origine)
	_, fileOrigine, rigaOrigine, okOrigine := runtime.Caller(2)

	if okCorrente && okOrigine {
		funzioneCorrente := runtime.FuncForPC(pcCorrente).Name()

		// Puliamo il percorso del file e il pacchetto per renderlo leggibile in console
		filePulito := fileOrigine[strings.LastIndex(fileOrigine, "/")+1:]
		if idx := strings.LastIndex(funzioneCorrente, "."); idx != -1 {
			funzioneCorrente = funzioneCorrente[idx+1:]
		}

		// Scrive su entrambi (console + file) come nella penultima proposta
		log.Printf("[FLUSSO] ➡️ Entrato in [%s] -- Richiamata da file: %s (Linea: %d)\n",
			funzioneCorrente, filePulito, rigaOrigine)
	}
}
