package traderepublic

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// deviceInfo represents the device identification payload sent to Trade Republic
type deviceInfo struct {
	StableDeviceID string `json:"stableDeviceId"`
}

// generateDeviceInfo creates a base64-encoded device info payload
// matching the format expected by Trade Republic's API
func generateDeviceInfo() string {
	// Generate a stable device ID using SHA-512 of a UUID
	raw := fmt.Sprintf("%d", time.Now().UnixNano())
	hash := sha512.Sum512([]byte(raw))
	deviceID := fmt.Sprintf("%x", hash)

	info := deviceInfo{StableDeviceID: deviceID}
	jsonBytes, _ := json.Marshal(info)
	return base64.StdEncoding.EncodeToString(jsonBytes)
}

// fetchWAFToken uses a headless browser (rod) to load the Trade Republic web app
// and extract the AWS WAF token that is required for API authentication.
// The WAF token is generated client-side by AWS WAF JavaScript challenge.
func fetchWAFToken() (string, error) {
	log.Println("Fetching AWS WAF token via headless browser...")

	// Launch browser — rod auto-downloads Chromium if needed
	path, _ := launcher.LookPath()
	u := launcher.New().Bin(path).
		Headless(true).
		Set("disable-gpu").
		Set("no-sandbox").
		MustLaunch()

	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	// Hide webdriver detection
	page := browser.MustPage("")
	// Use CDP to mask navigator.webdriver
	_, err := page.EvalOnNewDocument(`Object.defineProperty(navigator, 'webdriver', {get: () => undefined})`)
	if err != nil {
		return "", fmt.Errorf("failed to set webdriver override: %w", err)
	}

	// Navigate to Trade Republic app
	err = page.Navigate("https://app.traderepublic.com/")
	if err != nil {
		return "", fmt.Errorf("failed to navigate to Trade Republic: %w", err)
	}

	// Wait for the WAF challenge to complete (typically takes a few seconds)
	time.Sleep(6 * time.Second)

	// Method 1: Try to extract from cookies
	cookies, err := page.Cookies([]string{"https://app.traderepublic.com"})
	if err == nil {
		for _, cookie := range cookies {
			if cookie.Name == "aws-waf-token" {
				log.Println("WAF token retrieved from cookie")
				return cookie.Value, nil
			}
		}
	}

	// Method 2: Try via JavaScript AWSWafIntegration API
	result, err := page.Eval(`() => {
		if (window.AWSWafIntegration && typeof window.AWSWafIntegration.getToken === 'function') {
			return window.AWSWafIntegration.getToken();
		}
		return null;
	}`)
	if err == nil && result.Value.Str() != "" {
		log.Println("WAF token retrieved via AWSWafIntegration.getToken()")
		return result.Value.Str(), nil
	}

	// Method 3: Check all cookies for any waf-related token
	for _, cookie := range cookies {
		if containsWAF(cookie.Name) {
			log.Printf("WAF token retrieved from cookie: %s", cookie.Name)
			return cookie.Value, nil
		}
	}

	return "", fmt.Errorf("could not retrieve AWS WAF token")
}

// containsWAF checks if a cookie name is WAF-related
func containsWAF(name string) bool {
	return len(name) > 0 && (name == "aws-waf-token" ||
		name == "awswaf-token" ||
		containsSubstring(name, "waf"))
}

// containsSubstring is a simple case-insensitive substring check
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sc := s[i+j]
			uc := substr[j]
			// lowercase comparison
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if uc >= 'A' && uc <= 'Z' {
				uc += 32
			}
			if sc != uc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// Ensure proto import is used (cookies return type)
var _ = proto.NetworkCookie{}
