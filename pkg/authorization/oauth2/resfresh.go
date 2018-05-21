package oauth2

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Ensure that the session is still usable, if not use the refresh token to get a new one
func (s *Session) Ensure(client *Client) error {
	t := time.Now().Add(time.Second * 5) //5 second buffer, just to be sure

	if s.NextRefresh.After(t) {
		//no need to refresh, return now
		return nil
	}
	log.Println("Refresh Token")
	newSession, err := client.refresh(s)
	if err != nil {
		return err
	}
	*s = *newSession
	return nil
}

func (c *Client) refresh(session *Session) (*Session, error) {
	client := &http.Client{}
	basicToken := base64.StdEncoding.EncodeToString([]byte(session.clientID + ":" + session.clientSecret))

	//save refresh token
	refreshToken := session.RefreshToken

	formData := url.Values{}
	formData.Add("grant_type", "refresh_token")
	formData.Add("refresh_token", session.RefreshToken)

	req, err := http.NewRequest("POST", c.tokenURL.String(), strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("Could not create request to get spotify token :%v", err)
	}

	req.Header.Add("Authorization", "Basic "+basicToken)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Could not send request for token: %v", err)
	}
	bytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("Could not read Body for token: %v", err)
	}
	var newSession Session
	if err := json.Unmarshal(bytes, &newSession); err != nil {
		return nil, fmt.Errorf("Could not parse JSON for token")
	}
	log.Println("New Session:")
	log.Println(newSession)
	t := time.Now()
	nextRefresh := t.Add(time.Second * time.Duration(session.ExpiresIn))
	newSession.NextRefresh = nextRefresh
	newSession.RefreshToken = refreshToken
	return &newSession, nil
}
