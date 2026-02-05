package traderepublic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"valhafin/internal/service/scraper/types"
)

type loginResponse struct {
	ProcessID          string `json:"processId"`
	CountdownInSeconds int    `json:"countdownInSeconds"`
}

// Authenticate performs the authentication flow for Trade Republic (exported for CLI tools)
func (s *Scraper) Authenticate(phoneNumber, pin string) (string, error) {
	return s.authenticate(phoneNumber, pin)
}

// Complete2FA completes the 2FA authentication process (exported for CLI tools)
func (s *Scraper) Complete2FA(processID, code string) (string, error) {
	return s.Authenticate2FA(processID, code)
}

// authenticate performs the authentication flow for Trade Republic
// Note: This is a simplified version that doesn't handle 2FA interactively
// In a production environment, this would need to be handled differently
func (s *Scraper) authenticate(phoneNumber, pin string) (string, error) {
	// Step 1: Initiate login
	loginPayload := map[string]string{
		"phoneNumber": phoneNumber,
		"pin":         pin,
	}

	loginBody, _ := json.Marshal(loginPayload)
	req, err := http.NewRequest("POST", baseURL+"/api/v1/auth/web/login", bytes.NewBuffer(loginBody))
	if err != nil {
		return "", types.NewNetworkError("traderepublic", "Failed to create login request", err)
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", types.NewNetworkError("traderepublic", "Failed to send login request", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", types.NewAuthError("traderepublic", "Login failed. Check your phone number and PIN", nil)
	}

	var loginResp loginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return "", types.NewParsingError("traderepublic", "Failed to parse login response", err)
	}

	if loginResp.ProcessID == "" {
		return "", types.NewAuthError("traderepublic", "Failed to initialize connection. Check your phone number and PIN", nil)
	}

	// Note: In a real implementation, we would need to handle 2FA here
	// This could be done via:
	// 1. Storing the processID and returning it to the client
	// 2. Having a separate endpoint to complete authentication with the 2FA code
	// 3. Using a callback mechanism

	// For now, we return an error indicating that 2FA is required
	return "", types.NewAuthError("traderepublic",
		fmt.Sprintf("2FA authentication required. Process ID: %s. This needs to be completed interactively.", loginResp.ProcessID),
		nil)
}

// Authenticate2FA completes the 2FA authentication process
// This is a helper method that can be called separately when 2FA code is available
func (s *Scraper) Authenticate2FA(processID, code string) (string, error) {
	// Step 3: Verify device with 2FA code
	verifyURL := fmt.Sprintf("%s/api/v1/auth/web/login/%s/%s", baseURL, processID, code)
	verifyReq, err := http.NewRequest("POST", verifyURL, nil)
	if err != nil {
		return "", types.NewNetworkError("traderepublic", "Failed to create verification request", err)
	}

	verifyReq.Header.Set("User-Agent", userAgent)

	verifyResp, err := s.client.Do(verifyReq)
	if err != nil {
		return "", types.NewNetworkError("traderepublic", "Failed to send verification request", err)
	}
	defer verifyResp.Body.Close()

	if verifyResp.StatusCode != http.StatusOK {
		return "", types.NewAuthError("traderepublic", "Device verification failed. Check the code and try again", nil)
	}

	// Step 4: Extract session token from cookies
	for _, cookie := range verifyResp.Cookies() {
		if cookie.Name == "tr_session" {
			return cookie.Value, nil
		}
	}

	// Also check Set-Cookie header
	setCookie := verifyResp.Header.Get("Set-Cookie")
	if strings.Contains(setCookie, "tr_session=") {
		parts := strings.Split(setCookie, "tr_session=")
		if len(parts) > 1 {
			token := strings.Split(parts[1], ";")[0]
			return token, nil
		}
	}

	return "", types.NewAuthError("traderepublic", "Session token not found in response", nil)
}
