package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// WingetPackage represents a subset of winget.run API response
type WingetPackage struct {
	Id     string `json:"Id"`
	Latest struct {
		Name      string `json:"Name"`
		Publisher string `json:"Publisher"`
	} `json:"Latest"`
}

type wingetV2Response struct {
	Packages []WingetPackage `json:"Packages"`
	Total    int             `json:"Total"`
}

var httpClient = &http.Client{Timeout: 8 * time.Second}

// ResolveWingetID tries to find a winget package Id by a human-friendly app name.
// It returns the first match's Id if found.
func ResolveWingetID(appName string) (string, error) {
	if appName == "" {
		return "", errors.New("app name is empty")
	}
	endpoint := "https://api.winget.run/v2/packages"
	q := url.Values{}
	q.Set("query", appName)
	reqURL := fmt.Sprintf("%s?%s", endpoint, q.Encode())

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("winget.run returned status %d", resp.StatusCode)
	}

	var data wingetV2Response
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	if len(data.Packages) == 0 {
		return "", errors.New("no package found")
	}

	// Return the first match id
	return data.Packages[0].Id, nil
}
