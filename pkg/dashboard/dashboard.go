package dashboard

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"pr2056002/pkg/middleware"
	"pr2056002/pkg/render"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// MetricCard definisce la struttura dati per le card hardware/software
type MetricCard struct {
	Title       string
	Value       template.HTML
	BadgeText   string
	BadgeClass  string
	IsFullWidth bool
	IsHalfWidth bool
}

// ToolButton definisce lo scheletro per i pulsanti di utility e database
type ToolButton struct {
	Href  string
	Label string
}

// raccoglie l'intero payload dati richiesto da dashboard.html
type DashboardContext struct {
	ProjectName string
	LogoUrl     string
	AvatarUrl   string
	UserRole    string
	Version     string
	Metrics     []MetricCard
	Tools       []ToolButton
	Databases   []ToolButton
}

// Questa variabile mantiene in memoria la connessione aperta al boot
var db *sql.DB // Conserva l'istanza della connessione a PostgreSQL

// SetDatabase riceve la connessione attiva dal main.go
func SetDatabase(database *sql.DB) {
	db = database
}

func GetDatabase() *sql.DB {
	return db
}

// Helper nativo per leggere la RAM su Linux (VPS) senza comandi esterni
func leggiRamLinux() float64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0.0
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.ParseFloat(fields[1], 64)
				if err == nil {
					return kb / 1024 / 1024
				}
			}
		}
	}
	return 0.0
}

// Helper nativo per leggere la RAM su Mac tramite sysctl
func leggiRamMac() float64 {
	cmd := exec.Command("sysctl", "-n", "hw.memsize")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0.0
	}
	bytesVal, err := strconv.ParseFloat(strings.TrimSpace(out.String()), 64)
	if err == nil {
		return bytesVal / 1024 / 1024 / 1024
	}
	return 0.0
}

// Calcolo dello spazio su disco tramite syscall
func getSpazioDisco() string {
	var stat syscall.Statfs_t
	err := syscall.Statfs("/", &stat)
	if err != nil {
		return "Errore lettura disco"
	}
	all := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)

	allGb := float64(all) / 1024 / 1024 / 1024
	freeGb := float64(free) / 1024 / 1024 / 1024

	return fmt.Sprintf("%.1f GB liberi su %.1f GB totali", freeGb, allGb)
}

// BuildTree scansiona ricorsivamente le directory per esporre la struttura ad albero
func BuildTree(dirPath string, prefix string) string {
	var output strings.Builder
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return prefix + " [Errore di lettura directory]\n"
	}

	var filteredFiles []os.DirEntry
	for _, f := range files {
		nome := f.Name()
		if nome == "node_modules" || nome == ".git" || nome == ".gitignore" || nome == ".DS_Store" || nome == ".env" || nome == "bin" {
			continue
		}
		filteredFiles = append(filteredFiles, f)
	}

	for index, file := range filteredFiles {
		fullPath := filepath.Join(dirPath, file.Name())
		isLast := index == len(filteredFiles)-1
		marker := "├── "
		if isLast {
			marker = "└── "
		}

		output.WriteString(fmt.Sprintf("%s%s%s\n", prefix, marker, file.Name()))

		if file.IsDir() {
			newPrefix := prefix + "│   "
			if isLast {
				newPrefix = prefix + "    "
			}
			output.WriteString(BuildTree(fullPath, newPrefix))
		}
	}
	return output.String()
}

// RenderDashboard compila la Dashboard estraendo i dati reali della macchina
func RenderDashboard(w http.ResponseWriter, r *http.Request) {
	// 1. Recupero della sessione dal contesto iniettato dalla SentinellaInerte
	sessione, _ := r.Context().Value(middleware.UserContextKey).(middleware.UserSession)

	// Sicurezza: Se l'utente non è autenticato, viene rispedito in Homepage
	if !sessione.Autenticato {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	project := os.Getenv("PROJECT")
	logoUrl := "/img/" + os.Getenv("LOGO_PROJ")
	avatarUrl := "/img/" + os.Getenv("AVATAR_PROJ")

	sistemaOperativo := fmt.Sprintf("%s (%s)", runtime.GOOS, runtime.GOARCH)
	versioneGo := runtime.Version()
	spazioDisco := getSpazioDisco()

	var ramTotale float64
	if runtime.GOOS == "darwin" {
		ramTotale = leggiRamMac()
	} else {
		ramTotale = leggiRamLinux()
	}

	var ramValore string
	if ramTotale > 0 {
		ramValore = fmt.Sprintf("%.1f GB totali rilevati", ramTotale)
	} else {
		ramValore = "Rilevazione hardware fallita"
	}

	// 2. Correzione dei tipi espliciti template.HTML per evitare il blocco grafico
	metrics := []MetricCard{
		{Title: "Sistema operativo", Value: template.HTML(sistemaOperativo), IsFullWidth: false, IsHalfWidth: false},
		{Title: "Motore Backend Attivo", Value: template.HTML(versioneGo + " / Native Server"), IsFullWidth: false, IsHalfWidth: false},
		{Title: "Server Web Nginx", Value: template.HTML("Proxy Inverso Operativo"), IsFullWidth: false, IsHalfWidth: false},
		{Title: "Memoria RAM Fisico", Value: template.HTML(ramValore), BadgeText: "Ottimale", BadgeClass: "text-bg-success", IsFullWidth: false, IsHalfWidth: false},
		{Title: "Archiviazione Disco (Root)", Value: template.HTML(spazioDisco), IsFullWidth: false, IsHalfWidth: false},
		{Title: "Motore Database", Value: template.HTML("PostgreSQL 17 (Connesso)"), BadgeText: "Operativo", BadgeClass: "text-bg-success", IsFullWidth: false, IsHalfWidth: false},
	}

	tools := []ToolButton{
		{Href: "/Structure", Label: "view-Structure"},
		{Href: "/view-todo", Label: "view-todo"},
		{Href: "/view-tags", Label: "view-tags"},
		{Href: "/send-test-email", Label: "Invio email"},
		{Href: "/test-error-404", Label: "Test Error 404"},
		{Href: "/test-error-500", Label: "Test Error 500"},
		{Href: "/development-log", Label: "Development log"},
	}

	databases := []ToolButton{
		{Href: "/allUsers", Label: "Tabella Utenti"},
	}

	// Determina la dizione in italiano in base al ruolo della sessione
	var ruoloVisualizzato string
	if !sessione.Autenticato {
		ruoloVisualizzato = "Anonimo"
	} else {
		switch sessione.Ruolo { // Presumendo che l'oggetto sessione contenga il ruolo
		case "admin":
			ruoloVisualizzato = "Amministratore"
		case "user":
			ruoloVisualizzato = "Utente"
		default:
			ruoloVisualizzato = "Utente"
		}
	}

	// 3. Il nome utente viene iniettato dinamicamente (mostrerà "amorepi" o chi effettua il login)
	data := DashboardContext{
		ProjectName: project,
		LogoUrl:     logoUrl,
		AvatarUrl:   avatarUrl,
		UserRole:    ruoloVisualizzato,
		Version:     render.CurrentVersion,
		Metrics:     metrics,
		Tools:       tools,
		Databases:   databases,
	}

	layoutPath := "html/templates/layout.html"
	pagePath := "html/templates/dashboard.html"

	tmplt, err := template.ParseFiles(layoutPath, pagePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Errore parsing template dashboard: " + err.Error()))
		return
	}

	var buf bytes.Buffer
	err = tmplt.ExecuteTemplate(&buf, "layout", data)
	if err != nil {
		render.HandleTemplateError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = buf.WriteTo(w)
}
