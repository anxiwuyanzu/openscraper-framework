package tests

import (
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

const DefaultPort = 7599

// Server 单元测试用服务器。勿用于生产环境。
type Server struct {
	Port int
	Mux  map[string]http.HandlerFunc

	server *http.Server
}

func (s *Server) Start() {
	if s.Port <= 0 {
		s.Port = DefaultPort
	}
	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.Port),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	if s.Mux != nil {
		mux := http.NewServeMux()
		for path, handler := range s.Mux {
			mux.HandleFunc(path, handler)
		}
		s.server.Handler = mux
	}
	go s.server.ListenAndServe()
	log.Infof("test server running on port: %d", s.Port)
}

func (s *Server) Close() {
	if err := s.server.Close(); err != nil {
		log.WithError(err).Error("failed to close test server.")
	}
}
