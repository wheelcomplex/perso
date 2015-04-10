package main

import (
	"io"
	"log"
	"net/http"
)

type errorNotFound string

func (e errorNotFound) Error() string {
	return string(e)
}

var errNotFound = errorNotFound("Not found")

type httpHandler struct {
	help    *help
	cache   *caches
	config  *config
	indexer *mailIndexer
}

func newHttpHandler(help *help, cache *caches, config *config, indexer *mailIndexer) *httpHandler {
	return &httpHandler{
		help:    help,
		cache:   cache,
		config:  config,
		indexer: indexer,
	}
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/help" {
		h.help.write(w)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	if err := h.writeFromURL(r.URL.Path, io.Writer(w)); err != nil {
		switch err.(type) {
		case errorRedirect:
			http.Redirect(w, r, err.Error(), 307)
		case errorNotFound:
			http.NotFound(w, r)
		default:
			log.Print(r.URL.Path, ": ", err)
			http.NotFound(w, r)
		}
	}
}

func (h *httpHandler) writeFromURL(url string, w io.Writer) error {
	cacheReq, err := makeCacheRequest(url)
	if err != nil {
		return err
	}

	if !h.indexer.keys.has(cacheReq.header) {
		return errInvalidURL
	}

	cacheReq.match = h.indexer.keys.keyType(cacheReq.header)

	// Copy request object as it will be modified
	h.cache.requestCh <- *cacheReq
	data := <-cacheReq.data

	if data == nil || len(data) == 0 {
		return errNotFound
	}

	data.writeTo(w, h.config)
	return nil
}
