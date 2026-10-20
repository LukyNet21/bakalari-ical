package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

func login(baseURL, username, password string) (string, error) {
	loginURL := fmt.Sprintf("%s/api/login", strings.TrimRight(baseURL, "/"))
	data := url.Values{}
	data.Set("client_id", "ANDR")
	data.Set("grant_type", "password")
	data.Set("username", username)
	data.Set("password", password)

	r, err := http.NewRequest("POST", loginURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := httpClient.Do(r)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	var decodedLoginRes loginResponse
	if err := json.NewDecoder(res.Body).Decode(&decodedLoginRes); err != nil {
		return "", err
	}

	return decodedLoginRes.AccessToken, nil
}
