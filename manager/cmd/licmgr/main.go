package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
	"crypto/ed25519"
)

// simple in-memory store for dev
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
}

var (
	store = map[string]*ClientInfo{}
	mu    sync.Mutex
)

func main() {
	http.HandleFunc("/ping", pingHandler)
	http.HandleFunc("/heartbeat", heartbeatHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/report", reportHandler)

	log.Println("License Manager (dev) running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

func heartbeatHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{"status": "ok", "server_time": time.Now().UTC().Format(time.RFC3339)}
	writeJSON(w, resp)
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	var req map[string]interface{}
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
	quota := int64(100 * 1024 * 1024) // 100 MB default quota for dev

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
	}

	mu.Lock()
	store[clientID] = ci
	mu.Unlock()

	license := map[string]interface{}{
		"license_id":  licenseID,
		"client_id":   clientID,
		"app_id":      appID,
		"quota_bytes": quota,
		"issued_at":   now,
		"expires_at":  expires,
		"fingerprint": fingerprint,
	}

	resp := map[string]interface{}{
		"license":        license,
		"signature":      "dev-signed-placeholder",
		"server_pub_key": "dev-server-pubkey-placeholder",
	}

	writeJSON(w, resp)
}

func reportHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	clientID, _ := req["client_id"].(string)
	licenseID, _ := req["license_id"].(string)
	totalUsage, _ := req["total_usage_bytes"].(float64)
	sigB64, _ := req["signature"].(string)

	if clientID == "" || licenseID == "" || sigB64 == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	mu.Lock()
	ci, ok := store[clientID]
	mu.Unlock()
	if !ok {
		http.Error(w, "unknown client", http.StatusBadRequest)
		return
	}

	sig, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		http.Error(w, "invalid signature encoding", http.StatusBadRequest)
		return
	}

	delete(req, "signature")
	canonical, _ := json.Marshal(req)

	pubKeyBytes, err := base64.StdEncoding.DecodeString(ci.PubKey)
	if err != nil {
		http.Error(w, "invalid stored public key", http.StatusInternalServerError)
		return
	}

	if !ed25519.Verify(pubKeyBytes, canonical, sig) {
		log.Printf("Signature verification failed for client %s", clientID)
		http.Error(w, "signature verification failed", http.StatusBadRequest)
		return
	}

	mu.Lock()
	ci.TotalUsage = int64(totalUsage)
	allowed := ci.TotalUsage <= ci.QuotaBytes
	remaining := ci.QuotaBytes - ci.TotalUsage
	mu.Unlock()

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

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.Encode(v)
}
