package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cybozu-go/well"
)

func getJSON(ctx context.Context, p string, params map[string]string, data interface{}) error {
	client := &well.HTTPClient{
		Client: &http.Client{},
	}

	req, _ := http.NewRequest("GET", globalParams.endpoint+p, nil)
	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(data)
	if err != nil {
		return err
	}

	return nil
}

func postAction(ctx context.Context, p string, params map[string]string) error {
	client := &well.HTTPClient{
		Client: &http.Client{},
	}

	req, _ := http.NewRequest("POST", globalParams.endpoint+p, nil)
	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}
	return err
}
