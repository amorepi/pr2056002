package dashboard

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"pr2056002/pkg/render"
	"strings"
)

// Struttura dati per passare i valori a structure.html
type StructureContext struct {
	ProjectName string
	LogoUrl     string
	AvatarUrl   string
	UserName    string
	Version     string
	TreeOutput  string
}

// Struttura dati per passare i valori a markdown.html
type MarkdownContext struct {
	ProjectName string
	LogoUrl     string
	AvatarUrl   string
	UserName    string
	Version     string
	Title       string
	Subtitle    string
	HtmlContent template.HTML
}

// StructureHandler esegue il parsing del file structure.html senza HTML inline
func StructureHandler(w http.ResponseWriter, r *http.Request) {
	project := os.Getenv("PROJECT")
	currentDirectory, _ := os.Getwd()
	if project == "" {
		project = filepath.Base(currentDirectory)
	}

	treeOutput := fmt.Sprintf("%s/\n%s", project, BuildTree(currentDirectory, ""))

	data := StructureContext{
		ProjectName: project,
		LogoUrl:     "/img/" + os.Getenv("LOGO_PROJ"),
		AvatarUrl:   "/img/" + os.Getenv("AVATAR_PROJ"),
		UserName:    "Amministratore",
		Version:     render.CurrentVersion,
		TreeOutput:  treeOutput,
	}

	eseguiTemplateProtetto(w, r, "html/templates/structure.html", data)
}

// ViewMarkdownHandler esegue il parsing del file markdown.html senza HTML inline
func ViewMarkdownHandler(w http.ResponseWriter, r *http.Request, targetFile, title, subtitle string) {
	data, err := os.ReadFile(targetFile)
	var contenuto template.HTML
	if err != nil {
		contenuto = template.HTML(`<div class="text-muted italic">Impossibile accedere al file.</div>`)
	} else {
		// Sostituzione base dei ritorni a capo convertiti in tag HTML prima di inserire un parser .md
		conversione := strings.ReplaceAll(string(data), "\n", "<br>")
		contenuto = template.HTML(conversione)
	}

	context := MarkdownContext{
		ProjectName: os.Getenv("PROJECT"),
		LogoUrl:     "/img/" + os.Getenv("LOGO_PROJ"),
		AvatarUrl:   "/img/" + os.Getenv("AVATAR_PROJ"),
		UserName:    "Amministratore",
		Version:     render.CurrentVersion,
		Title:       title,
		Subtitle:    subtitle,
		HtmlContent: contenuto,
	}

	eseguiTemplateProtetto(w, r, "html/templates/markdown.html", context)
}

// Helper per proteggere l'esecuzione del template ed evitare pagine bianche
func eseguiTemplateProtetto(w http.ResponseWriter, r *http.Request, pagePath string, data interface{}) {
	layoutPath := "html/templates/layout.html"
	tmplt, err := template.ParseFiles(layoutPath, pagePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Errore caricamento template componenti: " + err.Error()))
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
