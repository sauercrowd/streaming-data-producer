package spotify

import (
	"github.com/sauercrowd/streaming-data-producer/pkg/authorization/oauth2"
)

// Session provides (Authorization) context for methods
type Session struct {
	oauth2Session *oauth2.Session
	oauth2Client  *oauth2.Client
}

const redirectURI = "http://localhost:8085"
const targetState = "505"
const targetScope = "user-read-playback-state"
const authorizeURL = "https://accounts.spotify.com/authorize"
const tokenURL = "https://accounts.spotify.com/api/token"

// NewSession creates a new Spotify Session
func NewSession(clientID, clientSecret string) (*Session, error) {
	client, err := oauth2.NewClient(redirectURI, targetScope, targetState, authorizeURL, tokenURL)
	if err != nil {
		return nil, err
	}
	session, err := client.Login(clientID, clientSecret)
	if err != nil {
		return nil, err
	}
	return &Session{
		oauth2Client:  client,
		oauth2Session: session,
	}, nil
}
