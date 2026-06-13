package webui

import (
	"HomemadeTorrent/pkg/control"
	"HomemadeTorrent/pkg/delay"
	"HomemadeTorrent/pkg/registre"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

//go:embed static/*
var staticFiles embed.FS

type RegisterState struct {
	SiteID string          `json:"SiteID"`
	Files  []registre.File `json:"Files"`
	Delay  *delay.Delay    `json:"Delay"`
}

type WSOutRegisterChanged struct {
	Type string        `json:"type"`
	Data RegisterState `json:"data"`
}

type WSInEnvelope struct {
	Type string `json:"type"`
}

type WSInRefresh struct {
	Type string `json:"type"`
}

type WSInAction struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type WSInUpdateDelay struct {
	Type  string      `json:"type"`
	Delay delay.Delay `json:"delay"`
}

type WebUI struct {
	SiteID    string
	Port      string
	OnMessage func(string)
	controler *control.Controller
	clientMu  sync.Mutex // overkill
	client    *websocket.Conn
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func StartWebUI(
	c *control.Controller,
	onMessage func(string),
) *WebUI {
	// Offset port by index so each site has a unique endpoint
	port := fmt.Sprintf("808%d", c.SiteIndex)

	ui := &WebUI{
		SiteID:    c.SiteID,
		Port:      port,
		OnMessage: onMessage,
		controler: c,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", ui.handleHome)
	mux.HandleFunc("/api/info", ui.handleInfo)
	mux.HandleFunc("/api/send", ui.handleSendMessage)
	mux.HandleFunc("/ws", ui.handleWS)

	log.Printf("[WEBUI] Starting for %s on http://localhost:%s\n", ui.SiteID, port)
	go func() {
		err := http.ListenAndServe(":"+port, mux)
		if err != nil {
			log.Printf("[WEBUI] Error starting HTTP server on %s : %v\n", port, err)
		}
	}()

	return ui
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
	state, err := ui.stateJSON()
	if err != nil {
		log.Printf("[WEBUI] Error building register state: %v\n", err)
		http.Error(w, "Erreur serveur : impossible de charger l'interface", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(state)
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

	if ui.OnMessage != nil {
		ui.OnMessage(msg)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (ui *WebUI) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WEBUI] WebSocket upgrade failed: %v\n", err)
		return
	}
	defer conn.Close()

	ui.clientMu.Lock()
	ui.client = conn
	ui.clientMu.Unlock()
	log.Printf("[WEBUI] Client connected for %s\n", ui.SiteID)

	ui.SendRegisterState()

	for {
		_, msgData, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var envelope WSInEnvelope
		err = json.Unmarshal(msgData, &envelope)
		if err != nil {
			log.Printf("[WEBUI] Failed to parse message type: %v\n", err)
			continue
		}

		switch envelope.Type {
		case "refresh":
			var msg WSInRefresh
			err = json.Unmarshal(msgData, &msg)
			if err != nil {
				log.Printf("[WEBUI] Failed to parse refresh message: %v\n", err)
				continue
			}
			ui.handleRefresh(&msg)

		case "action":
			var msg WSInAction
			err = json.Unmarshal(msgData, &msg)
			if err != nil {
				log.Printf("[WEBUI] Failed to parse action message: %v\n", err)
				continue
			}
			ui.handleAction(&msg)

		case "update-delay":
			var msg WSInUpdateDelay
			err = json.Unmarshal(msgData, &msg)
			if err != nil {
				log.Printf("[WEBUI] Failed to parse update delay message: %v\n", err)
				continue
			}
			ui.handleUpdateDelay(&msg)

		default:
			log.Printf("[WEBUI] Unknown message type: %s\n", envelope.Type)
		}
	}

	ui.clientMu.Lock()
	ui.client = nil
	ui.clientMu.Unlock()
	log.Printf("[WEBUI] Client disconnected for %s\n", ui.SiteID)
}

// envoie l'état actuelle du registre au client connecté
// méthode publique car doit pouvoir être appelée depuis l'eventloop
func (ui *WebUI) SendRegisterState() {
	ui.clientMu.Lock()
	client := ui.client
	ui.clientMu.Unlock()
	if client == nil {
		return
	}

	state := ui.registerState()

	msg := WSOutRegisterChanged{
		Type: "register-changed",
		Data: state,
	}
	encoded, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WEBUI] Error encoding websocket message: %v\n", err)
		return
	}

	if err := client.WriteMessage(websocket.TextMessage, encoded); err != nil {
		log.Printf("[WEBUI] Error sending state: %v\n", err)
		ui.clientMu.Lock()
		ui.client = nil
		ui.clientMu.Unlock()
	}
}

func (ui *WebUI) handleRefresh(msg *WSInRefresh) {
	log.Printf("[WEBUI] Client requested refresh\n")
	ui.SendRegisterState()
}

func (ui *WebUI) handleAction(msg *WSInAction) {
	if msg.Message == "" {
		log.Printf("[WEBUI] Empty action message\n")
		return
	}
	if len(msg.Message) > 50 {
		log.Printf("[WEBUI] Action received: %s\n", msg.Message[:50])
	} else {
		log.Printf("[WEBUI] Action received: %s\n", msg.Message)
	}
	if ui.OnMessage != nil {
		ui.OnMessage(msg.Message)
	}
}

func (ui *WebUI) handleUpdateDelay(msg *WSInUpdateDelay) {
	log.Printf("[WEBUI] Update delay received: %v\n", msg.Delay)
	ui.controler.UpdateDelay(msg.Delay)
}

func (ui *WebUI) registerState() RegisterState {
	var fileList []registre.File
	if ui.controler.Reg != nil {
		fileList = ui.controler.Reg.GetFileList()
	}

	return RegisterState{
		SiteID: ui.SiteID,
		Files:  fileList,
		Delay:  ui.controler.Delay,
	}
}

func (ui *WebUI) stateJSON() ([]byte, error) {
	return json.Marshal(ui.registerState())
}
