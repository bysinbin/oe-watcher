package web_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"medya-filter/pkg/app"
	"medya-filter/pkg/config"
	"medya-filter/pkg/web"
)

func TestServerAuthentication(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.WebPassword = "testsecretpassword"
	cfg.WebPort = 8089

	application := app.NewAppWithConfig(cfg, "", tempDir)
	_ = application.Init(false)

	server := web.NewServer(application, cfg.WebPort)

	// Spin up test HTTP server with handler from server.Start mux
	// Since Start calls ListenAndServe, we can test endpoints via HTTP calls once started or test handler
	// Let's start the server in a goroutine on an ephemeral port
	go func() {
		_ = server.Start()
	}()
	time.Sleep(100 * time.Millisecond)

	baseURL := "http://127.0.0.1:8089"

	// 1. Check auth status: should require auth and be unauthenticated
	resp, err := http.Get(baseURL + "/api/auth-status")
	if err != nil {
		t.Fatalf("Failed to get auth status: %v", err)
	}
	var statusData map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&statusData)
	resp.Body.Close()

	if statusData["required"] != true {
		t.Errorf("Expected required=true, got %v", statusData["required"])
	}
	if statusData["authenticated"] != false {
		t.Errorf("Expected authenticated=false, got %v", statusData["authenticated"])
	}

	// 2. Unauthenticated access to /api/status should be rejected with 401
	resp401, err := http.Get(baseURL + "/api/status")
	if err != nil {
		t.Fatalf("Failed to request /api/status: %v", err)
	}
	resp401.Body.Close()
	if resp401.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized, got %d", resp401.StatusCode)
	}

	// 3. Login with wrong password should fail
	wrongBody, _ := json.Marshal(map[string]string{"password": "wrong"})
	respWrong, err := http.Post(baseURL+"/api/login", "application/json", bytes.NewBuffer(wrongBody))
	if err != nil {
		t.Fatalf("Login request failed: %v", err)
	}
	respWrong.Body.Close()
	if respWrong.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for wrong password, got %d", respWrong.StatusCode)
	}

	// 4. Login with correct password should succeed and give token
	correctBody, _ := json.Marshal(map[string]string{"password": "testsecretpassword"})
	respLogin, err := http.Post(baseURL+"/api/login", "application/json", bytes.NewBuffer(correctBody))
	if err != nil {
		t.Fatalf("Login request failed: %v", err)
	}
	var loginData map[string]interface{}
	_ = json.NewDecoder(respLogin.Body).Decode(&loginData)
	respLogin.Body.Close()

	if respLogin.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK for correct password, got %d", respLogin.StatusCode)
	}
	token, ok := loginData["token"].(string)
	if !ok || token == "" {
		t.Fatalf("Expected valid session token, got %v", loginData["token"])
	}

	// 5. Authenticated access using Bearer token to /api/status should succeed with 200
	req, _ := http.NewRequest("GET", baseURL+"/api/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	respAuth, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Authenticated request failed: %v", err)
	}
	respAuth.Body.Close()
	if respAuth.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for authenticated request, got %d", respAuth.StatusCode)
	}

	// 6. Test Settings Auto-Skip toggle and verify config persistence
	configFile := filepath.Join(tempDir, "config.json")
	_ = config.Save(configFile, cfg)
	appWithFile := app.NewAppWithConfig(cfg, configFile, tempDir)
	appWithFile.SetAutoSkip(true)

	savedCfg, _, err := config.LoadOrCreate(configFile)
	if err != nil {
		t.Fatalf("Failed to reload saved config: %v", err)
	}
	if !savedCfg.AutoSkip {
		t.Errorf("Expected AutoSkip=true to be saved in config file, got false")
	}

	// 7. Test Snooze functionality
	appWithFile.SnoozePerson("Test Artist", 1*time.Hour)
	if !appWithFile.IsSnoozed("Test Artist") {
		t.Errorf("Expected 'Test Artist' to be snoozed")
	}
	if appWithFile.IsSnoozed("Another Artist") {
		t.Errorf("Expected 'Another Artist' not to be snoozed")
	}

	// 8. Test Export and Import Rules
	exportBytes, err := appWithFile.ExportRules()
	if err != nil {
		t.Fatalf("ExportRules failed: %v", err)
	}
	if len(exportBytes) == 0 {
		t.Fatalf("Expected non-empty export bytes")
	}
	err = appWithFile.ImportRules(exportBytes)
	if err != nil {
		t.Fatalf("ImportRules failed: %v", err)
	}

	// 9. Test SSE Endpoint (/api/events)
	reqSSE, _ := http.NewRequest("GET", baseURL+"/api/events", nil)
	reqSSE.Header.Set("Authorization", "Bearer "+token)
	respSSE, err := http.DefaultClient.Do(reqSSE)
	if err != nil {
		t.Fatalf("SSE request failed: %v", err)
	}
	defer respSSE.Body.Close()

	if respSSE.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for SSE, got %d", respSSE.StatusCode)
	}
	ct := respSSE.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Errorf("Expected text/event-stream Content-Type, got %s", ct)
	}

	// 10. Test History API endpoints
	reqHistory, _ := http.NewRequest("GET", baseURL+"/api/history", nil)
	reqHistory.Header.Set("Authorization", "Bearer "+token)
	respHistory, err := http.DefaultClient.Do(reqHistory)
	if err != nil {
		t.Fatalf("History request failed: %v", err)
	}
	defer respHistory.Body.Close()
	if respHistory.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 for /api/history, got %d", respHistory.StatusCode)
	}

	// Test CSV export
	reqCSV, _ := http.NewRequest("GET", baseURL+"/api/history/export?format=csv", nil)
	reqCSV.Header.Set("Authorization", "Bearer "+token)
	respCSV, err := http.DefaultClient.Do(reqCSV)
	if err != nil {
		t.Fatalf("History CSV export failed: %v", err)
	}
	defer respCSV.Body.Close()
	if respCSV.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 for CSV export, got %d", respCSV.StatusCode)
	}
	if !strings.Contains(respCSV.Header.Get("Content-Type"), "text/csv") {
		t.Errorf("Expected text/csv Content-Type, got %s", respCSV.Header.Get("Content-Type"))
	}

	// Test JSON export
	reqJSON, _ := http.NewRequest("GET", baseURL+"/api/history/export?format=json", nil)
	reqJSON.Header.Set("Authorization", "Bearer "+token)
	respJSON, err := http.DefaultClient.Do(reqJSON)
	if err != nil {
		t.Fatalf("History JSON export failed: %v", err)
	}
	defer respJSON.Body.Close()
	if respJSON.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 for JSON export, got %d", respJSON.StatusCode)
	}

	// Test Clear History
	reqClear, _ := http.NewRequest("POST", baseURL+"/api/history/clear", nil)
	reqClear.Header.Set("Authorization", "Bearer "+token)
	respClear, err := http.DefaultClient.Do(reqClear)
	if err != nil {
		t.Fatalf("History clear failed: %v", err)
	}
	defer respClear.Body.Close()
	if respClear.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 for clear history, got %d", respClear.StatusCode)
	}
}

