// tools/verify_summary.go
package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func fatal(msg string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", a...)
	os.Exit(1)
}

func main() {
	// find hw.json (search current dir or hw-emulator/)
	hwPaths := []string{"hw.json", "hw-emulator/hw.json"}
	var hwPath string
	for _, p := range hwPaths {
		if _, err := os.Stat(p); err == nil {
			hwPath = p
			break
		}
	}
	if hwPath == "" {
		fatal("hw.json not found in current dir or hw-emulator/")
	}

	hwB, err := os.ReadFile(hwPath)
	if err != nil {
		fatal("read hw.json: %v", err)
	}
	var hw struct {
		PrivateBase64 string `json:"private_key_base64"`
		DeviceID      string `json:"device_id"`
	}
	if err := json.Unmarshal(hwB, &hw); err != nil {
		fatal("parse hw.json: %v", err)
	}

	seed, err := base64.StdEncoding.DecodeString(hw.PrivateBase64)
	if err != nil {
		fatal("decode private_key_base64: %v", err)
	}
	if len(seed) != 32 {
		// PyNaCl SigningKey.encode() returns 32 bytes seed; if it's 64, take first 32
		if len(seed) >= 32 {
			seed = seed[:32]
		} else {
			fatal("unexpected seed length: %d", len(seed))
		}
	}

	// derive ed25519 private key from seed (Go expects 32-byte seed)
	priv := ed25519.NewKeyFromSeed(seed) // 64 bytes
	pub := priv.Public().(ed25519.PublicKey)

	// find latest summary file in summaries/
	summDir := "summaries"
	files, err := os.ReadDir(summDir)
	if err != nil {
		fatal("cannot read summaries directory: %v", err)
	}
	if len(files) == 0 {
		fatal("no files in summaries/")
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
		fatal("no summary files")
	}

	path := filepath.Join(summDir, latest)
	b, err := os.ReadFile(path)
	if err != nil {
		fatal("read summary file: %v", err)
	}

	// parse wrapper { "summary": {...}, "signature": "..." }
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal(b, &wrapper); err != nil {
		fatal("parse summary wrapper: %v", err)
	}
	summRaw, ok := wrapper["summary"]
	if !ok {
		fatal("summary key missing in file")
	}
	sigRaw, ok := wrapper["signature"]
	if !ok {
		fatal("signature key missing in file")
	}

	// canonicalize summary using Go's json.Marshal (same as manager did)
	var summary interface{}
	if err := json.Unmarshal(summRaw, &summary); err != nil {
		fatal("unmarshal inner summary: %v", err)
	}
	canonical, err := json.Marshal(summary)
	if err != nil {
		fatal("marshal canonical: %v", err)
	}

	// get signature string
	var sigStr string
	if err := json.Unmarshal(sigRaw, &sigStr); err != nil {
		// signature might be raw (without quotes)
		var tmp interface{}
		if err2 := json.Unmarshal(sigRaw, &tmp); err2 == nil {
			if s, ok2 := tmp.(string); ok2 {
				sigStr = s
			}
		}
		if sigStr == "" {
			fatal("cannot parse signature value")
		}
	}

	sigBytes, err := base64.StdEncoding.DecodeString(sigStr)
	if err != nil {
		fatal("decode signature base64: %v", err)
	}

	// verify
	okVerify := ed25519.Verify(pub, canonical, sigBytes)
	if okVerify {
		fmt.Printf("OK: signature verified for %s\n", path)
		os.Exit(0)
	} else {
		fmt.Printf("FAIL: signature verification failed for %s\n", path)
		// save canonical bytes to tmp file for debugging
		_ = os.WriteFile("last_canonical.json", canonical, 0644)
		fmt.Println("wrote last_canonical.json for debugging")
		os.Exit(2)
	}
}
