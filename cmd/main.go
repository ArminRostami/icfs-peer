package main

import (
	"encoding/json"
	httpa "icfs-client/adapters/http"
	"icfs-client/adapters/ipfs"
	"icfs-client/domain"
	"io"
	"log"
	"net/http"

	"github.com/pkg/errors"
)

const base = "http://127.0.0.1:8000"
const baseUI = "http://127.0.0.1:4200"

func run() error {
	cl := &http.Client{}

	req, err := http.NewRequest("GET", base+"/ipfs", nil)
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	resp, err := cl.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send request")
	}

	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return errors.Wrap(err, "failed to read response body")
	}

	var connInfo domain.UserConfig

	err = json.Unmarshal(body, &connInfo)
	if err != nil {
		return errors.Wrap(err, "failed to uwrap body into map")
	}

	log.Println(connInfo.Bootstrap, connInfo.SwarmKey)

	p, err := httpa.NewProxy(baseUI)
	if err != nil {
		return errors.Wrap(err, "failed to create proxy")
	}

	cancel, service, err := ipfs.NewService(&connInfo)
	defer cancel()
	if err != nil {
		return errors.Wrap(err, "failed to create ipfs service")
	}

	h := httpa.Handler{IS: service, RProxy: p}

	return h.Serve()
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("%+v", err)
	}
}
