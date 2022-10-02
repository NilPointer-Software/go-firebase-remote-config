package remote

import (
	"bytes"
	"context"
	"encoding/json"
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

func (rcc *RemoteConfigClient) Get() (*RemoteConfig, error) {
	var config RemoteConfig
	etag, err := rcc.doRequest("GET", "", &config, nil, nil)
	if err != nil {
		return nil, err
	}
	config.ETag = etag
	config.client = rcc
	return &config, nil
}

func (rcc *RemoteConfigClient) Update(config *RemoteConfig) (*RemoteConfig, error) {
	_, err := rcc.doRequest("PUT", "", config, config, &config.ETag)
	if err != nil {
		return nil, err
	}
	config.client = rcc
	return config, nil
}

func (rcc *RemoteConfigClient) doRequest(method, call string, output any, input any, etag *string, try ...int) (string, error) {
	if call != "" && call[0] != ':' {
		call = ":" + call
	}
	var err error
	var body []byte
	if input != nil {
		body, err = json.Marshal(input)
		if err != nil {
			return "", nil
		}
	}
	request, err := http.NewRequest(method, fmt.Sprintf("%s/%s/projects/%s/remoteConfig%s?alt=json&prettyPrint=false", rcc.endpoint, apiVersion, rcc.projectId, call), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "go-firebase-remote-config/v0")
	if etag != nil {
		request.Header.Set("If-Match", *etag)
	}

	response, err := rcc.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusConflict && len(try) == 0 {
			newTag, err := rcc.doRequest("GET", "", nil, nil, nil)
			if err != nil {
				return "", err
			}
			return rcc.doRequest(method, call, output, input, &newTag, 1)
		}
		return "", fmt.Errorf("unexpected status code %s (%d)\n%s", response.Status, response.StatusCode, body)
	}

	tag := response.Header.Get("ETag")
	if output != nil {
		err = json.Unmarshal(body, output)
		if err != nil {
			return "", err
		}
	}
	return tag, nil
}
