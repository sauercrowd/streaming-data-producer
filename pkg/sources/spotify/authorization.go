package spotify

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// Login to Spotify
func Login(clientID, clientSecret string) (*SpotifySession, error) {
	ret := SpotifySession{}
	ret.clientID = clientID
	ret.clientSecret = clientSecret

	if err := authorize(clientID); err != nil {
		log.Fatal(err)
	}
	code, err := waitForLogin()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Code: ", code)
	sToken, err := getSession(code, clientID, clientSecret)
	if err != nil {
		log.Fatal(err)
	}
	ret.token = sToken
	return &ret, nil
}

const redirectURI = "http://localhost:8085"
const targetState = "505"
const targetScope = "user-read-playback-state"

func authorize(clientID string) error {
	responseType := "code"

	url, err := url.Parse("https://accounts.spotify.com/authorize")
	if err != nil {
		return fmt.Errorf("Could not parse login URL: %v", err)
	}
	q := url.Query()
	q.Set("client_id", clientID)
	q.Set("response_type", responseType)
	q.Set("redirect_uri", redirectURI)
	q.Set("state", targetState)
	q.Set("scope", targetScope)

	url.RawQuery = q.Encode()
	s := url.String()
	fmt.Println("Please visit the following url, but make sure you're browser is able to reach port 8085 on this server")
	fmt.Println(s)
	return nil
}

type spotifyToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

func waitForLogin() (string, error) {
	s := &http.Server{
		Addr: "localhost:8085",
	}
	ret := ""
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		q := req.URL.Query()
		code := q.Get("code")
		state := q.Get("state")
		if state == targetState {
			ret = code
			if err := s.Close(); err != nil {
				log.Print("Could not close server")
			}
		}

	})

	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return "", err
	}
	return ret, nil
}

func getSession(code, clientID, clientSecret string) (spotifyToken, error) {
	client := &http.Client{}
	basicToken := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	formData := url.Values{}
	formData.Add("grant_type", "authorization_code")
	formData.Add("code", code)
	formData.Add("redirect_uri", redirectURI)

	var ret spotifyToken
	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(formData.Encode()))
	if err != nil {
		return ret, fmt.Errorf("Could not create request to get spotify token :%v", err)
	}

	req.Header.Add("Authorization", "Basic "+basicToken)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		return ret, fmt.Errorf("Could not send request for token: %v", err)
	}
	bytes, err := ioutil.ReadAll(res.Body)
	log.Println(string(bytes))
	defer res.Body.Close()
	if err != nil {
		return ret, fmt.Errorf("Could not read Body for token: %v", err)
	}
	if err := json.Unmarshal(bytes, &ret); err != nil {
		return ret, fmt.Errorf("Could not parse JSON for token")
	}
	return ret, nil
}
