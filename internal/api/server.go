// Package api provides the WebSocket JSON-RPC server for IPC with the Tauri frontend.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/Kizuno18/mongebot-go/internal/config"
	"github.com/Kizuno18/mongebot-go/internal/engine"
	"github.com/Kizuno18/mongebot-go/internal/logger"
	"github.com/Kizuno18/mongebot-go/internal/proxy"
)

// Server is the WebSocket JSON-RPC server for frontend communication.
type Server struct {
	cfg      config.APIConfig
	engine   *engine.Engine
	proxyMgr *proxy.Manager
	appCfg   *config.AppConfig
	logRing  *logger.RingBuffer
	logger   *slog.Logger

	// Connected clients
	clientsMu sync.RWMutex
	clients   map[*websocket.Conn]bool
}

// JSONRPCRequest represents a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      *int            `json:"id,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  any         `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      *int        `json:"id,omitempty"`
}

// RPCError represents a JSON-RPC error.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewServer creates a new API server.
func NewServer(cfg config.APIConfig, eng *engine.Engine, proxyMgr *proxy.Manager, appCfg *config.AppConfig, logRing *logger.RingBuffer, log *slog.Logger) *Server {
	return &Server{
		cfg:      cfg,
		engine:   eng,
		proxyMgr: proxyMgr,
		appCfg:   appCfg,
		logRing:  logRing,
		logger:   log.With("component", "api"),
		clients:  make(map[*websocket.Conn]bool),
	}
}

// Start begins listening for WebSocket connections.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		s.handleWebSocket(w, r)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Prometheus metrics endpoint
	mux.HandleFunc("/metrics", PrometheusHandler(s))

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", addr, err)
	}

	s.logger.Info("API server started", "addr", addr)

	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Subscribe to log events and broadcast to clients
	go s.broadcastLogs(ctx)

	// Subscribe to engine metrics and broadcast to clients
	s.engine.OnMetrics(func(m *engine.AggregatedMetrics) {
		s.broadcast("event.metrics", m)
	})

	go func() {
		<-ctx.Done()
		server.Close()
	}()

	return server.Serve(listener)
}

// handleWebSocket upgrades HTTP to WebSocket and handles JSON-RPC messages.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"}, // Tauri localhost
	})
	if err != nil {
		s.logger.Error("websocket accept failed", "error", err)
		return
	}
	defer conn.CloseNow()

	s.clientsMu.Lock()
	s.clients[conn] = true
	s.clientsMu.Unlock()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, conn)
		s.clientsMu.Unlock()
	}()

	s.logger.Info("client connected")

	ctx := r.Context()
	for {
		var req JSONRPCRequest
		if err := wsjson.Read(ctx, conn, &req); err != nil {
			break
		}

		resp := s.handleRequest(ctx, &req)
		if err := wsjson.Write(ctx, conn, resp); err != nil {
			break
		}
	}
}

// handleRequest routes a JSON-RPC request to the appropriate handler.
func (s *Server) handleRequest(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	handler, exists := s.getHandler(req.Method)
	if !exists {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error:   &RPCError{Code: -32601, Message: fmt.Sprintf("method %q not found", req.Method)},
			ID:      req.ID,
		}
	}

	result, err := handler(ctx, req.Params)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error:   &RPCError{Code: -32000, Message: err.Error()},
			ID:      req.ID,
		}
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	}
}

// broadcast sends an event to all connected clients.
func (s *Server) broadcast(method string, params any) {
	msg := JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  map[string]any{"method": method, "params": params},
	}

	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	for conn := range s.clients {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		wsjson.Write(ctx, conn, msg)
		cancel()
	}
}

// broadcastLogs subscribes to the log ring buffer and broadcasts new entries.
func (s *Server) broadcastLogs(ctx context.Context) {
	_, ch := s.logRing.Subscribe(100)

	for {
		select {
		case <-ctx.Done():
			return
		case entry := <-ch:
			s.broadcast("event.log", entry)
		}
	}
}
