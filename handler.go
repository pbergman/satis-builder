package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"net/http"
	"os/user"

	"github.com/docker/docker/client"
	"github.com/pbergman/logger"
	"github.com/rs/xid"
)

type Handler struct {
	cnf    *Config
	logger *logger.Logger

	ctx context.Context
	usr *user.User
	cli *client.Client
}

func (h *Handler) isValidRequest(req *http.Request) bool {

	if h.cnf.Secret != "" && req.Header.Get("x-hub-signature") == "" {
		return false
	}

	switch req.Header.Get("x-github-event") {
	case "push":
		return true
	default:
		return false
	}
}

func (h *Handler) isValidRepo(repo string) bool {
	for i, c := 0, len(h.cnf.Repositories); i < c; i++ {
		if h.cnf.Repositories[i] == repo {
			return true
		}
	}
	return false
}

func (h *Handler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {

	var id xid.ID = xid.New()

	resp.Header().Add("X-REQUEST-ID", id.String())
	h.logger.Info(fmt.Sprintf("[%s] new request on '%s'", id, req.URL.Path))

	if false == h.isValidRequest(req) {
		h.logger.Info(fmt.Sprintf("[%s] not processing because request is invalid", id))
		http.Error(resp, "", http.StatusBadRequest)
		return
	}

	defer req.Body.Close()

	var reader io.Reader = req.Body
	var hasher hash.Hash

	if "" != h.cnf.Secret {
		hasher = hmac.New(sha1.New, []byte(h.cnf.Secret))
		reader = io.TeeReader(req.Body, hasher)
	}

	raw, err := ioutil.ReadAll(reader)

	if err != nil {
		h.logger.Error(fmt.Sprintf("[%s] failed to read request body", id))
		http.Error(resp, "", http.StatusInternalServerError)
		return
	}

	if nil != hasher {
		if e := "sha1=" + hex.EncodeToString(hasher.Sum(nil)); e != req.Header.Get("x-hub-signature") {
			h.logger.Error(fmt.Sprintf("[%s] invalid payload signature %s' != '%s'", id, e, req.Header.Get("x-hub-signature")))
			http.Error(resp, "", http.StatusNotAcceptable)
			return
		}
	}

	switch req.Header.Get("x-github-event") {
	case "push":
		var data struct {
			N string `json:"ref"`
			R struct {
				N string `json:"full_name"`
			} `json:"repository"`
			S struct {
				L string `json:"login"`
			} `json:"sender"`
		}

		if err := json.Unmarshal(raw, &data); err != nil {
			h.logger.Error(fmt.Sprintf("[%s] failed to decode payload: %s", id, err.Error()))
			http.Error(resp, "", http.StatusBadRequest)
			return
		}

		h.logger.Info(fmt.Sprintf("[%s] new push on repo '%s' by '%s'", id, data.R.N, data.S.L))

		if h.isValidRepo(data.R.N) {
			h.logger.Info(fmt.Sprintf("[%s] processing repo", id))

			if err := BuildSatis(h.ctx, h.cli, h.usr, h.cnf, h.logger, data.R.N); err != nil {
				h.logger.Error(fmt.Sprintf("[%s] failed to build image: %s", id, err.Error()))
				http.Error(resp, "", http.StatusBadRequest)
				return
			}

		} else {
			h.logger.Info(fmt.Sprintf("[%s] not processing because repo is no managed", id))
		}
	}
}
