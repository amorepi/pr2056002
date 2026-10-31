## v1.1.17 - 31/10/2026_08:46: Continuazione lavori sulla dashboard

## v1.1.16 - 30/10/2026_17:46: Fix

## v1.1.15 - 30/10/2026_10:08: Fix

## v1.1.14 - 30/10/2026_09:48: Fix

## v1.1.13 - 30/10/2026_09:13: Fix

## v1.1.12 - 30/10/2026_09:02: Fix

## v1.1.11 - 30/10/2026_08:34: Fix

## v1.1.10 - 30/10/2026_08:13: Fix

## v1.1.9 - 30/10/2026_08:11: Fix

## v1.1.8 - 30/10/2026_06:46: Fix

## v1.1.7 - 28/10/2026_20:48: Aggiornamento funzionalità e manutenzione

## v1.1.6 - 28/10/2026_19:10: Aggiornamento funzionalità e manutenzione

## v1.1.5 - 28/10/2026_18:28: Aggiornamento funzionalità e manutenzione

## v1.1.4 - 26/10/2026_22:53: Fix

## v1.1.3 - 26/10/2026_22:44: Fix

## v1.1.2 - 26/10/2026_22:29: Fix

## v1.1.1 - 26/10/2026_22:15: Fix
  - Fix

## v1.1.0 - 26/10/2026_16:38: Adozione Bootstrap 3.8, Gestione errori e alcune funzioni di base
  - Ingegnerizzazione **Architettura Modulare a Pacchetti**: Disaccoppiamento radicale della logica aziendale (JSON API) dai servizi visivi globali e di sistema tramite una strutturazione con `/pkg`.
  - Sviluppo **Motore di Rendering Nativo**: Implementazione del pacchetto isolato `pkg/render` basato su `html/template` per la compilazione a componenti annidati (`layout.html` e template figli).
  - Integrazione **Interfaccia Bootstrap 5.8**: Layout Top-Fixed responsivo a tre colonne geometriche con asset statici (Logo e Avatar Amministratore) gestiti dinamicamente da variabili d'ambiente `.env`.
  - Scudo di Sicurezza **Recovery Middleware**: Rilascio del middleware globale in `pkg/middleware` per l'intercettazione dei crash critici (`panic`) con estrazione dello stack trace dei log e deviazione su errore grafico `500.html`.
  - Standardizzazione **Gestione Errori di Rete**: Instradamento automatico intelligente dei fallimenti (`404` e `500`) in modalità bivalente: risposta in JSON per le chiamate `/api` e interfaccia grafica Bootstrap per la navigazione web.

## v1.0.0 - 25/10/2026_13:26 - INITIAL START
