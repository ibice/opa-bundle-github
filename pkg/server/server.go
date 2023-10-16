package server

import (
	"encoding/json"
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
	lastRevision := r.Header.Get("If-None-Match")

	data, revision, err := server.repository.Get(r.Context(), lastRevision)
	if err != nil {
		server.error(w, err.Error(), http.StatusInternalServerError)
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

type serverError struct {
	Error string `json:"error"`
}

func (server Server) error(w http.ResponseWriter, err string, code int) {
	slog.Error(err)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(&serverError{Error: err})
}
