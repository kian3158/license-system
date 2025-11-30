package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
	"crypto/ed25519"
)

// simple client info store
type ClientInfo struct {
	ClientID    string                 `json:"client_id"`
	PubKey      string                 `json:"pub_key"`
	Fingerprint map[string]interface{} `json:"fingerprint"`
	AppID       string                 `json:"app_id"`
	Version     string                 `json:"version"`
	LicenseID   string                 `json:"license_id"`
	QuotaBytes  int64                  `json:"quota_bytes"`
	TotalUsage  int64                  `json:"total_usage_bytes"`
	IssuedAt    string                 `json:"issued_at"`
	ExpiresAt   string                 `json:"expires_at"`
	Revoked     bool                   `json:"revoked"`
}

var (
	store   = map[string]*ClientInfo{}
	storeMu sync.Mutex
	dbFile  = "clients_store.json"
)

func main() {
	loadStore()

	http.HandleFunc("/ping", pingHandler)
	http.HandleFunc("/heartbeat", heartbeatHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/report", reportHandler)
	http.HandleFunc("/revoke", revokeHandler)

	log.Println("License Manager running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

func heartbeatHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "ok", "server_time": time.Now().UTC().Format(time.RFC3339)})
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	body, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	clientID, _ := req["client_id"].(string)
	pubKey, _ := req["pub_key"].(string)
	appID, _ := req["app_id"].(string)
	version, _ := req["version"].(string)
	fingerprint, _ := req["fingerprint"].(map[string]interface{})

	if clientID == "" || pubKey == "" || appID == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	licenseID := "LIC-" + time.Now().UTC().Format("20060102150405")
	now := time.Now().UTC().Format(time.RFC3339)
	expires := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	quota := int64(100 * 1024 * 1024)

	ci := &ClientInfo{
		ClientID:    clientID,
		PubKey:      pubKey,
		Fingerprint: fingerprint,
		AppID:       appID,
		Version:     version,
		LicenseID:   licenseID,
		QuotaBytes:  quota,
		TotalUsage:  0,
		IssuedAt:    now,
		ExpiresAt:   expires,
		Revoked:     false,
	}

	storeMu.Lock()
	store[clientID] = ci
	saveStore()
	storeMu.Unlock()

	license := map[string]interface{}{
		"license_id":  licenseID,
		"client_id":   clientID,
		"app_id":      appID,
		"quota_bytes": quota,
		"issued_at":   now,
		"expires_at":  expires,
		"fingerprint": fingerprint,
	}

	writeJSON(w, map[string]interface{}{
		"license":        license,
		"signature":      "dev-signed-placeholder",
		"server_pub_key": "dev-server-pubkey-placeholder",
	})
}

func reportHandler(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	body, _ := io.ReadAll(r.Body)
	json.Unmarshal(body, &req)

	clientID, _ := req["client_id"].(string)
	totalUsage, _ := req["total_usage_bytes"].(float64)
	sigB64, _ := req["signature"].(string)

	storeMu.Lock()
	ci, ok := store[clientID]
	storeMu.Unlock()
	if !ok {
		http.Error(w, "unknown client", http.StatusBadRequest)
		return
	}

	if ci.Revoked {
		writeJSON(w, map[string]interface{}{
			"status":  "ok",
			"allowed": false,
			"action":  "disable",
			"reason":  "revoked",
		})
		return
	}

	sig, _ := base64.StdEncoding.DecodeString(sigB64)
	delete(req, "signature")
	canonical, _ := json.Marshal(req)
	pubKeyBytes, _ := base64.StdEncoding.DecodeString(ci.PubKey)

	if !ed25519.Verify(pubKeyBytes, canonical, sig) {
		http.Error(w, "signature verification failed", http.StatusBadRequest)
		return
	}

	storeMu.Lock()
	ci.TotalUsage = int64(totalUsage)
	allowed := ci.TotalUsage <= ci.QuotaBytes
	remaining := ci.QuotaBytes - ci.TotalUsage
	saveStore()
	storeMu.Unlock()

	resp := map[string]interface{}{
		"status":          "ok",
		"allowed":         allowed,
		"remaining_bytes": remaining,
	}
	if !allowed {
		resp["action"] = "disable"
		resp["reason"] = "quota exceeded"
	} else {
		resp["action"] = "continue"
	}
	writeJSON(w, resp)
}

func revokeHandler(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	body, _ := io.ReadAll(r.Body)
	json.Unmarshal(body, &req)
	clientID := req["client_id"]

	storeMu.Lock()
	ci, ok := store[clientID]
	if ok {
		ci.Revoked = true
		saveStore()
	}
	storeMu.Unlock()

	writeJSON(w, map[string]interface{}{"status": "ok", "revoked": ok})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func saveStore() {
	f, _ := os.Create(dbFile)
	json.NewEncoder(f).Encode(store)
	f.Close()
}

func loadStore() {
	if _, err := os.Stat(dbFile); err == nil {
		f, _ := os.Open(dbFile)
		json.NewDecoder(f).Decode(&store)
		f.Close()
	}
}
