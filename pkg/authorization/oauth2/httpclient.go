package oauth2

import (
	"fmt"
	"net/http"
)

func (session *Session) DoHTTPRequest(client *Client, req *http.Request) (*http.Response, error) {
	httpclient := http.Client{}
	err := session.Ensure(client)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", session.AccessToken))
	return httpclient.Do(req)
}
