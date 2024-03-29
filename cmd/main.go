package main

import (
	"encoding/json"
	"fmt"
	"icfs-peer/adapters/ipfs"
	"icfs-peer/domain"
	"io"
	"log"
	"net/http"

	"github.com/pkg/browser"
	"github.com/pkg/errors"
)

const (
	bootstrap = "bootstrap"
	bootPort  = 8000
)

func getConnInfo() (*domain.UserConfig, error) {
	cl := &http.Client{}

	req, err := http.NewRequest("GET", getInfoURL(bootstrap, bootPort), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	resp, err := cl.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request")
	}

	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	var connInfo domain.UserConfig

	err = json.Unmarshal(body, &connInfo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to uwrap body into map")
	}

	log.Println(connInfo.Bootstrap)

	return &connInfo, nil
}

func getInfoURL(host string, port int) string {
	return fmt.Sprintf("http://%s:%d/ipfs", host, port)
}

func run() error {
	connInfo, err := getConnInfo()
	if err != nil {
		return errors.Wrap(err, "failed to get connInfo")
	}

	cancel, service, err := ipfs.NewService(connInfo)
	defer cancel()
	if err != nil {
		return errors.Wrap(err, "failed to create ipfs service")
	}

	ec := make(chan error)
	go service.Start(ec)

	err = browser.OpenURL(fmt.Sprintf("http://%s:%d", bootstrap, bootPort))
	if err != nil {
		return errors.Wrap(err, "failed to open browser")
	}

	return <-ec
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("%+v", err)
	}
}
