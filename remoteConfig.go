package firebaseRemoteConfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/api/option"
	"google.golang.org/api/transport"
	"io/ioutil"
	"net/http"
)

const (
	baseEndpoint = "https://firebaseremoteconfig.googleapis.com"
	apiVersion   = "v1"
)

type RemoteConfigClient struct {
	httpClient *http.Client
	endpoint   string
	projectId  string
}

func NewClient(ctx context.Context, opts ...option.ClientOption) (*RemoteConfigClient, error) {
	client := RemoteConfigClient{endpoint: baseEndpoint}
	options := append(opts, option.WithScopes("https://www.googleapis.com/auth/firebase.remoteconfig"))

	credentials, err := transport.Creds(ctx, options...)
	if err != nil {
		return nil, err
	}
	client.projectId = credentials.ProjectID

	endpoint := ""
	client.httpClient, endpoint, err = transport.NewHTTPClient(ctx, options...)
	if err != nil {
		return nil, err
	}
	if endpoint != "" {
		client.endpoint = endpoint
	}

	return &client, nil
}

func (rcc *RemoteConfigClient) GetRaw() (*RemoteConfig, error) {
	var config RemoteConfig
	err := rcc.doRequest("GET", "", &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (rcc *RemoteConfigClient) doRequest(method, call string, output any) error {
	if call != "" && call[0] != ':' {
		call = ":" + call
	}
	request, err := http.NewRequest(method, fmt.Sprintf("%s/%s/projects/%s/remoteConfig%s?alt=json&prettyPrint=false", rcc.endpoint, apiVersion, rcc.projectId, call), nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "go-firebase-remote-config-client/v0")

	response, err := rcc.httpClient.Do(request)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("unexpected status code %s (%d)\n%s", response.Status, response.StatusCode, body))
	}

	err = json.Unmarshal(body, output)
	if err != nil {
		return err
	}
	return nil
}
