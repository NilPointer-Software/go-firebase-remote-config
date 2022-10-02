package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/api/option"
	"testing"
)

func TestRemoteConfig(t *testing.T) {
	client, err := NewClient(context.Background(), option.WithCredentialsFile("service-account.json"))
	if err != nil {
		panic(err)
	}

	raw, err := client.Get()
	if err != nil {
		panic(err)
	}

	out, err := json.MarshalIndent(raw, "", "    ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))
}

func TestListVersions(t *testing.T) {
	client, err := NewClient(context.Background(), option.WithCredentialsFile("service-account.json"))
	if err != nil {
		panic(err)
	}

	versions, err := client.ListAllVersions()
	if err != nil {
		panic(err)
	}

	fmt.Println("Available versions:", len(versions))
	data, err := json.MarshalIndent(versions, "", "    ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
}
