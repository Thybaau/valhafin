package traderepublic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type loginResponse struct {
	ProcessID          string `json:"processId"`
	CountdownInSeconds int    `json:"countdownInSeconds"`
}

func (s *Scraper) Authenticate() (string, error) {
	// Step 1: Initiate login
	loginPayload := map[string]string{
		"phoneNumber": s.config.Secret.PhoneNumber,
		"pin":         s.config.Secret.Pin,
	}

	loginBody, _ := json.Marshal(loginPayload)
	req, err := http.NewRequest("POST", baseURL+"/api/v1/auth/web/login", bytes.NewBuffer(loginBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var loginResp loginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return "", fmt.Errorf("failed to initialize connection. Check your phone number and PIN")
	}

	if loginResp.ProcessID == "" {
		return "", fmt.Errorf("failed to initialize connection. Check your phone number and PIN")
	}

	// Step 2: Request 2FA code
	fmt.Printf("❓ Enter the 2FA code received (%d seconds remaining) or type 'SMS': ", loginResp.CountdownInSeconds)
	var code string
	fmt.Scanln(&code)

	// If user wants SMS
	if strings.ToUpper(code) == "SMS" {
		resendURL := fmt.Sprintf("%s/api/v1/auth/web/login/%s/resend", baseURL, loginResp.ProcessID)
		resendReq, _ := http.NewRequest("POST", resendURL, nil)
		resendReq.Header.Set("User-Agent", userAgent)
		s.client.Do(resendReq)

		fmt.Print("❓ Enter the 2FA code received by SMS: ")
		fmt.Scanln(&code)
	}

	// Step 3: Verify device with 2FA code
	verifyURL := fmt.Sprintf("%s/api/v1/auth/web/login/%s/%s", baseURL, loginResp.ProcessID, code)
	verifyReq, _ := http.NewRequest("POST", verifyURL, nil)
	verifyReq.Header.Set("User-Agent", userAgent)

	verifyResp, err := s.client.Do(verifyReq)
	if err != nil {
		return "", err
	}
	defer verifyResp.Body.Close()

	if verifyResp.StatusCode != 200 {
		return "", fmt.Errorf("device verification failed. Check the code and try again")
	}

	fmt.Println("✅ Device verified successfully!")

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

	return "", fmt.Errorf("session token not found")
}
