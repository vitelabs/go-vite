package main

import (
	"log"
	"net/http"
	"time"
)

type Config struct {
	PrivateKey        string
	Seed              bool
	ListenAddress     string
	PublicAddress     string
	CertFile, KeyFile string
	Timeout           time.Duration
}

type Server struct {
	Config
	srv http.Server
}

func New(cfg Config) *Server {
	s := &Server{
		Config: cfg,
		srv: http.Server{
			Addr:              cfg.ListenAddress,
			Handler:           nil,
			ReadTimeout:       cfg.Timeout,
			ReadHeaderTimeout: cfg.Timeout,
			WriteTimeout:      cfg.Timeout,
			IdleTimeout:       0,
			MaxHeaderBytes:    0,
			TLSNextProto:      nil,
			ConnState:         nil,
			ErrorLog:          log.New(""),
		},
	}

	return s
}

func (s *Server) Start() {

}

func (s *Server) serve() {
	err := s.srv.ListenAndServeTLS(s.Config.CertFile, s.Config.KeyFile)
}

func (s *Server) handle() {

}

func main() {

}
