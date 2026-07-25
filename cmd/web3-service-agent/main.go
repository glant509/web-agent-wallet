package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"web3-service-agent/internal/agent"
	"web3-service-agent/internal/config"
	"web3-service-agent/internal/httpapi"
	"web3-service-agent/internal/llm/provider"
	"web3-service-agent/internal/marketdata"
	"web3-service-agent/internal/mcp"
	"web3-service-agent/internal/prompt"
	"web3-service-agent/internal/session"
	"web3-service-agent/internal/tool"
	"web3-service-agent/tools"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if cfg.Service == nil {
		log.Fatal("service config is required")
	}
	if err := marketdata.Configure(&cfg); err != nil {
		log.Fatalf("configure market data provider: %v", err)
	}

	sessionManager := session.NewManager()
	promptBuilder, err := prompt.NewBuilder(cfg.Service.Name)
	if err != nil {
		log.Fatalf("create prompt builder: %v", err)
	}
	toolRegistry := tool.NewRegistry()
	mcpRegistry := mcp.NewRegistry()
	_ = mcpRegistry

	if err := tools.RegisterAll(toolRegistry); err != nil {
		log.Fatalf("register tools: %v", err)
	}

	llmClient, err := provider.NewClient(&cfg)
	if err != nil {
		log.Fatalf("create llm client: %v", err)
	}
	runtimeEngine := agent.New(agent.Dependencies{
		Sessions:      sessionManager,
		PromptBuilder: promptBuilder,
		LLM:           llmClient,
		Tools:         toolRegistry,
		MaxSteps:      cfg.Service.AgentMaxSteps,
	})

	server := &http.Server{
		Addr:              cfg.Service.ListenAddr(),
		Handler:           httpapi.New(runtimeEngine, sessionManager),
		ReadHeaderTimeout: 10 * time.Second,
	}
	homeURL := homepageURL(cfg.Service.ListenAddr())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Printf("web3-service-agent listening on %s", cfg.Service.ListenAddr())
		log.Printf("web3-service-agent home page: %s", homeURL)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server failed: %v", err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("http shutdown failed: %v", err)
		}
	}
}

func homepageURL(listenAddr string) string {
	addr := strings.TrimSpace(listenAddr)
	if addr == "" {
		return "http://localhost/"
	}
	if strings.HasPrefix(addr, ":") {
		return "http://localhost" + addr + "/"
	}
	if strings.Contains(addr, "://") {
		if strings.HasSuffix(addr, "/") {
			return addr
		}
		return addr + "/"
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		if strings.HasSuffix(addr, "/") {
			return "http://" + addr
		}
		return "http://" + addr + "/"
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, port) + "/"
}
