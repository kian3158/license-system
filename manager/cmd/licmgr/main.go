package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
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
	store       = map[string]*ClientInfo{}
	storeMu     sync.Mutex
	dbFile      = "clients_store.json"
	hwEmuURL    = func() string { // override via HW_EMULATOR_URL env var if needed
		if v := os.Getenv("HW_EMULATOR_URL"); v != "" {
			return v
		}
		return "http://localhost:8000"
	}()
	summariesDir = "summaries"
)

func main() {
	loadStore()

	http.HandleFunc("/ping", pingHandler)
	http.HandleFunc("/heartbeat", heartbeatHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/report", reportHandler)
	http.HandleFunc("/revoke", revokeHandler)

	// summary endpoints
	http.HandleFunc("/generate_summary", generateSummaryHandler) // POST or GET for dev
	http.HandleFunc("/summary/latest", latestSummaryHandler)     // GET

	// start dev scheduler goroutine that periodically generates summaries
	go startSummaryScheduler()

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

/*****  S U M M A R Y   G E N E R A T I O N   &   HW  E M U  I N T E R A C T I O N  *****/

// signWithHW posts the canonical payload to the HW-emulator and returns the base64 signature string
func signWithHW(payload []byte) (string, error) {
	url := fmt.Sprintf("%s/sign", hwEmuURL)
	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("hw sign request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("hw sign returned %d: %s", resp.StatusCode, string(b))
	}
	var out struct {
		Signature string `json:"signature"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("invalid hw sign response: %w", err)
	}
	return out.Signature, nil
}

// generateSummary builds a summary object from current store, requests HW signature, and writes to disk.
func generateSummary() (string, error) {
	summary := map[string]interface{}{}
	clients := map[string]interface{}{}

	storeMu.Lock()
	for id, ci := range store {
		clients[id] = map[string]interface{}{
			"client_id":         ci.ClientID,
			"license_id":        ci.LicenseID,
			"total_usage_bytes": ci.TotalUsage,
			"quota_bytes":       ci.QuotaBytes,
			"revoked":           ci.Revoked,
			"issued_at":         ci.IssuedAt,
			"expires_at":        ci.ExpiresAt,
		}
	}
	storeMu.Unlock()

	today := time.Now().UTC().Format("2006-01-02")
	summary["date"] = today
	summary["generated_at"] = time.Now().UTC().Format(time.RFC3339)
	summary["clients"] = clients

	canonical, err := json.Marshal(summary)
	if err != nil {
		return "", err
	}

	// get signature from HW emulator
	sig, err := signWithHW(canonical)
	if err != nil {
		return "", err
	}

	outObj := map[string]interface{}{
		"summary":   summary,
		"signature": sig,
	}
	outBytes, err := json.MarshalIndent(outObj, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(summariesDir, 0o755); err != nil {
		return "", err
	}

	filename := filepath.Join(summariesDir, fmt.Sprintf("%s.json", today))
	if err := os.WriteFile(filename, outBytes, 0o644); err != nil {
		return "", err
	}

	return filename, nil
}

func generateSummaryHandler(w http.ResponseWriter, r *http.Request) {
	filename, err := generateSummary()
	if err != nil {
		http.Error(w, "generate summary failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]interface{}{"status": "ok", "file": filename})
}

func latestSummaryHandler(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(summariesDir)
	if err != nil || len(files) == 0 {
		writeJSON(w, map[string]interface{}{"status": "ok", "found": false})
		return
	}
	latest := ""
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		if name > latest {
			latest = name
		}
	}
	if latest == "" {
		writeJSON(w, map[string]interface{}{"status": "ok", "found": false})
		return
	}
	content, _ := os.ReadFile(filepath.Join(summariesDir, latest))
	w.Header().Set("Content-Type", "application/json")
	w.Write(content)
}

func startSummaryScheduler() {
	intervalMin := 1440 // daily by default
	if v := os.Getenv("SUMMARY_INTERVAL_MIN"); v != "" {
		if iv, err := strconv.Atoi(v); err == nil {
			intervalMin = iv
		}
	}
	ticker := time.NewTicker(time.Duration(intervalMin) * time.Minute)
	for range ticker.C {
		if fn, err := generateSummary(); err != nil {
			log.Printf("generateSummary error: %v", err)
		} else {
			log.Printf("summary written: %s", fn)
		}
	}
}
