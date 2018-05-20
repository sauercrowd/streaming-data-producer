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

// Session is a Oauth2 Session
type Session struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	clientID     string
	clientSecret string
	NextRefresh  time.Time
}

// Client is a Oauth2 Client
type Client struct {
	targetState, targetScope            string
	tokenURL, redirectURI, authorizeURL *url.URL
}

// NewClient creates a new Oauth2 Client and makes sure all URLs are valid
func NewClient(redirectURI, targetScope, targetState, authorizeURL, tokenURL string) (*Client, error) {
	parsedAuthorizeURL, err := url.Parse(authorizeURL)
	if err != nil {
		return nil, err
	}
	parsedTokenURL, err := url.Parse(tokenURL)
	if err != nil {
		return nil, err
	}
	parsedRedirectURI, err := url.Parse(redirectURI)
	if err != nil {
		return nil, err
	}
	return &Client{
		redirectURI:  parsedRedirectURI,
		targetState:  targetState,
		targetScope:  targetScope,
		authorizeURL: parsedAuthorizeURL,
		tokenURL:     parsedTokenURL,
	}, nil
}

// Login to a Oauth2 Provider
func (c *Client) Login(clientID, clientSecret string) (*Session, error) {
	if err := c.authorize(clientID); err != nil {
		return nil, err
	}

	code, err := c.waitForLogin()
	if err != nil {
		return nil, err
	}

	session, err := c.getSession(code, clientID, clientSecret)
	if err != nil {
		return nil, err
	}
	session.clientID = clientID
	session.clientSecret = clientSecret
	return session, nil
}

func (c *Client) authorize(clientID string) error {
	responseType := "code"

	url := c.authorizeURL

	q := url.Query()
	q.Set("client_id", clientID)
	q.Set("response_type", responseType)
	q.Set("redirect_uri", c.redirectURI.String())
	q.Set("state", c.targetState)
	q.Set("scope", c.targetScope)

	url.RawQuery = q.Encode()
	s := url.String()
	fmt.Printf("Please visit the following url, but make sure you're browser is able to reach %s:%s on this server\n", c.redirectURI.Hostname(), c.redirectURI.Port())
	fmt.Println(s)
	return nil
}

func (c *Client) waitForLogin() (string, error) {
	s := &http.Server{
		Addr: fmt.Sprintf("%s:%s", c.redirectURI.Hostname(), c.redirectURI.Port()),
	}
	ret := ""

	http.HandleFunc(c.redirectURI.RequestURI(), func(w http.ResponseWriter, req *http.Request) {
		q := req.URL.Query()
		code := q.Get("code")
		state := q.Get("state")
		if state == c.targetState {
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

func (c *Client) getSession(code, clientID, clientSecret string) (*Session, error) {
	client := &http.Client{}
	basicToken := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))

	formData := url.Values{}
	formData.Add("grant_type", "authorization_code")
	formData.Add("code", code)
	formData.Add("redirect_uri", c.redirectURI.String())

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
	var session Session
	if err := json.Unmarshal(bytes, &session); err != nil {
		return nil, fmt.Errorf("Could not parse JSON for token")
	}
	t := time.Now()
	nextRefresh := t.Add(time.Second * time.Duration(session.ExpiresIn))
	session.NextRefresh = nextRefresh
	return &session, nil
}
