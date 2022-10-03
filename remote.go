package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/api/option"
	"google.golang.org/api/transport"
	"io"
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
	etag, err := rcc.doRequest(request{method: "GET"}, &config)
	if err != nil {
		return nil, err
	}
	config.ETag = etag
	config.client = rcc
	return &config, nil
}

func (rcc *RemoteConfigClient) Update(config *RemoteConfig) (*RemoteConfig, error) {
	var newConfig RemoteConfig
	_, err := rcc.doRequest(request{
		method: "PUT",
		input:  config,
		etag:   config.ETag,
	}, &newConfig)
	if err != nil {
		return nil, err
	}
	newConfig.client = rcc
	return &newConfig, nil
}

func (rcc *RemoteConfigClient) Rollback(version int64) (*RemoteConfig, error) {
	var config RemoteConfig
	rollback := Rollback{VersionNumber: version}
	_, err := rcc.doRequest(request{
		method: "POST",
		call:   "rollback",
		input:  rollback,
	}, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (rcc *RemoteConfigClient) ListAllVersions() ([]Version, error) {
	var versions []Version
	page, err := rcc.ListVersionsByPageToken("")
	if err != nil {
		return nil, err
	}
	versions = append(versions, page.Versions...)
	for page.NextPageToken != "" {
		page, err = rcc.ListVersionsByPageToken(page.NextPageToken)
		if err != nil {
			return nil, err
		}
		versions = append(versions, page.Versions...)
	}
	return versions, nil
}

func (rcc *RemoteConfigClient) ListVersionsByPageToken(pageToken string) (*VersionPage, error) {
	var page VersionPage
	_, err := rcc.doRequest(request{
		method: "GET",
		call:   "listVersions",
		query: map[string]string{
			"pageToken": pageToken,
		},
	}, &page)
	if err != nil {
		return nil, err
	}
	return &page, nil
}

// TODO: ListVersions all query

// TODO: GetDefaults

type request struct {
	method string
	call   string
	query  map[string]string
	input  any
	etag   string
}

func (r *request) data() (reader io.Reader) {
	if r.input != nil {
		data, err := json.Marshal(r.input)
		if err != nil {
			panic(err)
		}
		reader = bytes.NewReader(data)
	}
	return
}

func (r *request) queryString() (res string) {
	for key, value := range r.query {
		res += "&" + key + "=" + value
	}
	return
}

func (r *request) callString() string {
	if r.call != "" {
		return ":" + r.call
	}
	return ""
}

func (rcc *RemoteConfigClient) doRequest(r request, output any, try ...int) (string, error) { // TODO: Rewrite
	url := fmt.Sprintf("%s/%s/projects/%s/remoteConfig%s?alt=json&prettyPrint=false%s",
		rcc.endpoint,
		apiVersion,
		rcc.projectId,
		r.callString(),
		r.queryString(),
	)
	req, err := http.NewRequest(r.method, url, r.data())
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "go-firebase-remote-config/v0")
	if r.input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if r.etag != "" {
		req.Header.Set("If-Match", r.etag)
	}

	response, err := rcc.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusConflict && len(try) == 0 {
			r.etag, err = rcc.doRequest(request{method: "GET"}, nil)
			if err != nil {
				return "", err
			}
			return rcc.doRequest(r, output, 1)
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
