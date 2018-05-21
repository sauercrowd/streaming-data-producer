package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/sauercrowd/streaming-data-producer/pkg/data"
)

const currentlyPlayingEndpoint = "https://api.spotify.com/v1/me/player/currently-playing"
const maxErrorCount = -1

type currentlyPlayingState struct {
	IsPlaying  bool
	songURI    string
	progressMs int64
}

func (s *Session) SubscribeCurrentPlaying(ctx context.Context, ch chan data.Datapoint, sleep time.Duration, onlyNew bool, compactFormat bool) error {
	first := true
	playID := int64(0)
	state := currentlyPlayingState{}
	errorCount := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if !first {
				time.Sleep(sleep)
			}
			first = false

			song, err := s.getCurrentlyPlaying()
			if err != nil {
				log.Println("Error while getting current song:", err)
				if maxErrorCount != -1 {
					errorCount++
				}
				if maxErrorCount != -1 && errorCount > maxErrorCount {
					log.Fatal("Max Error count reached")
				}
				continue
			}
			if onlyNew && state.IsPlaying == song.IsPlaying &&
				state.songURI == song.Item.URI &&
				state.progressMs <= song.ProgressMs { //song didn't get restarted or is paused
				continue
			}
			//log
			artist := ""
			if len(song.Item.Artists) > 0 {
				artist = song.Item.Artists[0].Name
			}
			log.Printf("[%s] %s", artist, song.Item.Name)

			dataPoint := transformSong(song, playID)

			ch <- dataPoint
			if state.songURI != song.Item.URI {
				playID++
			}
			state.IsPlaying = song.IsPlaying
			state.songURI = song.Item.URI
			state.progressMs = song.ProgressMs
		}
	}
}

func (s *Session) getCurrentlyPlaying() (*CurrentlyPlaying, error) {
	req, err := http.NewRequest("GET", currentlyPlayingEndpoint, nil)
	if err != nil {
		return nil, err
	}
	res, err := s.oauth2Session.DoHTTPRequest(s.oauth2Client, req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	var cp CurrentlyPlaying
	if err := json.Unmarshal(body, &cp); err != nil {
		return nil, err
	}
	return &cp, err
}

//creates a map with 19 fields
func transformSong(cp *CurrentlyPlaying, playID int64) data.Datapoint {
	m := make(map[string]string)
	m["timestamp"] = fmt.Sprint(cp.Timestamp)

	//song specific
	m["name"] = cp.Item.Name
	m["url"] = cp.Item.ExternalUrls.Spotify
	m["uri"] = cp.Item.URI
	m["popularity"] = fmt.Sprint(cp.Item.Popularity)
	m["explicit"] = fmt.Sprint(cp.Item.Explicit)
	m["progress_ms"] = fmt.Sprint(cp.ProgressMs)
	m["duration_ms"] = fmt.Sprint(cp.Item.DurationMs)
	m["is_playing"] = fmt.Sprint(cp.IsPlaying)

	//album specific
	m["album_name"] = cp.Item.Album.Name
	m["album_url"] = cp.Item.Album.ExternalUrls.Spotify
	m["track_number"] = fmt.Sprint(cp.Item.TrackNumber)
	if len(cp.Item.Album.Images) > 0 {
		m["album_image_url"] = cp.Item.Album.Images[0].URL
	} else {
		m["album_image_url"] = ""
	}

	//artist specific
	if len(cp.Item.Artists) > 0 {
		m["artist_name"] = cp.Item.Artists[0].Name
		m["artist_uri"] = cp.Item.Artists[0].URI
		m["artist_url"] = cp.Item.Artists[0].ExternalUrls.Spotify
	} else {
		m["artist_name"] = ""
		m["artist_uri"] = ""
		m["artist_url"] = ""
	}

	//context specific
	m["context_url"] = cp.Context.ExternalUrls.Spotify
	m["context_type"] = cp.Context.Type
	m["context_uri"] = cp.Context.URI

	flatCPlaying := CurrentlyPlayingStruct{
		PlayID:      playID,
		Timestamp:   cp.Timestamp,
		Name:        cp.Item.Name,
		URL:         cp.Item.ExternalUrls.Spotify,
		URI:         cp.Item.URI,
		Popularity:  cp.Item.Popularity,
		Explicit:    cp.Item.Explicit,
		ProgressMS:  cp.ProgressMs,
		DurationMS:  cp.Item.DurationMs,
		IsPlaying:   cp.IsPlaying,
		AlbumName:   cp.Item.Album.Name,
		AlbumURL:    cp.Item.Album.ExternalUrls.Spotify,
		TrackNumber: cp.Item.TrackNumber,
		ContextURL:  cp.Context.ExternalUrls.Spotify,
		ContextType: cp.Context.Type,
		ContextURI:  cp.Context.URI,
	}
	if len(cp.Item.Album.Images) > 0 {
		flatCPlaying.AlbumImageURL = cp.Item.Album.Images[0].URL
	}
	if len(cp.Item.Artists) > 0 {
		flatCPlaying.ArtistName = cp.Item.Artists[0].Name
		flatCPlaying.ArtistURI = cp.Item.Artists[0].URI
		flatCPlaying.ArtistURL = cp.Item.Artists[0].ExternalUrls.Spotify
	}
	return data.Datapoint{Map: m, Struct: flatCPlaying}
}

type CurrentlyPlayingStruct struct {
	Timestamp     int64
	PlayID        int64
	Name          string
	URL           string
	URI           string
	Popularity    int
	Explicit      bool
	ProgressMS    int64
	DurationMS    int64
	IsPlaying     bool
	AlbumName     string
	AlbumURL      string
	TrackNumber   int
	AlbumImageURL string
	ArtistName    string
	ArtistURI     string
	ArtistURL     string
	ContextURL    string
	ContextType   string
	ContextURI    string
}

type CurrentlyPlaying struct {
	Timestamp  int64 `json:"timestamp"`
	ProgressMs int64 `json:"progress_ms"`
	IsPlaying  bool  `json:"is_playing"`
	Item       struct {
		Album struct {
			AlbumType string `json:"album_type"`
			Artists   []struct {
				ExternalUrls struct {
					Spotify string `json:"spotify"`
				} `json:"external_urls"`
				Href string `json:"href"`
				ID   string `json:"id"`
				Name string `json:"name"`
				Type string `json:"type"`
				URI  string `json:"uri"`
			} `json:"artists"`
			AvailableMarkets []string `json:"available_markets"`
			ExternalUrls     struct {
				Spotify string `json:"spotify"`
			} `json:"external_urls"`
			Href   string `json:"href"`
			ID     string `json:"id"`
			Images []struct {
				Height int    `json:"height"`
				URL    string `json:"url"`
				Width  int    `json:"width"`
			} `json:"images"`
			Name                 string `json:"name"`
			ReleaseDate          string `json:"release_date"`
			ReleaseDatePrecision string `json:"release_date_precision"`
			Type                 string `json:"type"`
			URI                  string `json:"uri"`
		} `json:"album"`
		Artists []struct {
			ExternalUrls struct {
				Spotify string `json:"spotify"`
			} `json:"external_urls"`
			Href string `json:"href"`
			ID   string `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
			URI  string `json:"uri"`
		} `json:"artists"`
		AvailableMarkets []string `json:"available_markets"`
		DiscNumber       int      `json:"disc_number"`
		DurationMs       int64    `json:"duration_ms"`
		Explicit         bool     `json:"explicit"`
		ExternalIds      struct {
			Isrc string `json:"isrc"`
		} `json:"external_ids"`
		ExternalUrls struct {
			Spotify string `json:"spotify"`
		} `json:"external_urls"`
		Href        string `json:"href"`
		ID          string `json:"id"`
		IsLocal     bool   `json:"is_local"`
		Name        string `json:"name"`
		Popularity  int    `json:"popularity"`
		PreviewURL  string `json:"preview_url"`
		TrackNumber int    `json:"track_number"`
		Type        string `json:"type"`
		URI         string `json:"uri"`
	} `json:"item"`
	Context struct {
		ExternalUrls struct {
			Spotify string `json:"spotify"`
		} `json:"external_urls"`
		Href string `json:"href"`
		Type string `json:"type"`
		URI  string `json:"uri"`
	} `json:"context"`
}
