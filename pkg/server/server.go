package server

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/ibice/opa-bundle-github/pkg/log"
	"github.com/ibice/opa-bundle-github/pkg/repository"
)

type Server struct {
	port       uint
	address    string
	repository repository.Interface
	logger     *slog.Logger
}

func New(address string, port uint, repository repository.Interface) *Server {
	logger := log.Logger
	logger.Debug("Creating server", "address", address, "port", port)
	return &Server{
		port:       port,
		address:    address,
		repository: repository,
		logger:     logger,
	}
}

func (server Server) Run() error {
	mux := http.NewServeMux()

	mux.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("200 OK"))
	}))

	mux.Handle("/", server)

	server.logger.Info("Listening")
	return http.ListenAndServe(fmt.Sprintf("%s:%d", server.address, server.port), mux)
}

func (server Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		err := fmt.Sprintf("Method not allowed: %v", r.Method)
		server.logger.Error(err)
		http.Error(w, err, http.StatusMethodNotAllowed)
		return
	}

	lastRevision := r.Header.Get("If-None-Match")
	server.logger.Debug("Serving bundle request", "lastRevision", lastRevision)

	data, revision, err := server.repository.Get(r.Context(), lastRevision)
	if err != nil {
		server.logger.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Etag", revision)

	if revision == lastRevision {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "application/gzip")
	io.Copy(w, data)
}
