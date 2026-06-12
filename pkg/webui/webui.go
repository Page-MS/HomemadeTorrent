package webui

import (
	"HomemadeTorrent/pkg/registre"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
)

//go:embed static/*
var staticFiles embed.FS

type WebUI struct {
	SiteID    string
	Port      string
	OnMessage func(string)
	Register  *registre.Registre
}

func StartWebUI(siteID string, index int, onMessage func(string), register *registre.Registre, isBootstrap int) {
	var port string

	if isBootstrap == 1 {
		// Laisser le système choisir un port libre
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			log.Printf("[WEBUI] Impossible de trouver un port libre : %v", err)
			return
		}
		port = fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)
		listener.Close() // On libère pour que ListenAndServe puisse l'utiliser
	} else {
		// Offset port by index so each site has a unique endpoint
		port = fmt.Sprintf("808%d", index)
	}

	ui := WebUI{
		SiteID:    siteID,
		Port:      port,
		OnMessage: onMessage,
		Register:  register,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", ui.handleHome)
	mux.HandleFunc("/api/info", ui.handleInfo)
	mux.HandleFunc("/api/send", ui.handleSendMessage)

	log.Printf("[WEBUI] Starting for %s on http://localhost:%s\n", siteID, port)
	err := http.ListenAndServe(":"+port, mux)
	if err != nil {
		log.Printf("[WEBUI] Error starting HTTP server on %s : %v\n", port, err)
	}
}

func (ui *WebUI) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" || r.URL.Path == "/index.html" {
		htmlData, err := staticFiles.ReadFile("static/index.html")
		if err != nil {
			log.Printf("[WEBUI] Error reading index.html: %v\n", err)
			http.Error(w, "Erreur serveur : impossible de charger l'interface", http.StatusInternalServerError)
			return
		}

		data := struct {
			SiteID string
			Port   string
		}{
			SiteID: ui.SiteID,
			Port:   ui.Port,
		}

		tmpl, err := template.New("index").Parse(string(htmlData))
		if err != nil {
			log.Printf("[WEBUI] Error parsing template: %v\n", err)
			http.Error(w, "Erreur serveur : impossible de charger l'interface", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, data)
		return
	}

	// Serve static files from embedded files
	fileServer := http.FileServer(http.FS(staticFiles))
	fileServer.ServeHTTP(w, r)
}

func (ui *WebUI) handleInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var fileList []registre.File
	if ui.Register != nil {
		fileList = ui.Register.GetFileList()
	}

	data := struct {
		SiteID string
		Files  []registre.File
	}{
		SiteID: ui.SiteID,
		Files:  fileList,
	}

	json.NewEncoder(w).Encode(data)
}

func (ui *WebUI) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	msg := r.FormValue("message")

	if msg == "" {
		http.Error(w, "Message vide", http.StatusBadRequest)
		return
	}

	log.Printf("[WEBUI] Message reçu depuis l'interface pour %s : %s\n", ui.SiteID, msg)

	// Envoyer à l'event_loop
	if ui.OnMessage != nil {
		ui.OnMessage(msg)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
