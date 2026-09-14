package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"web3-service-agent/internal/agent"
	"web3-service-agent/internal/config"
	"web3-service-agent/internal/httpapi"
	"web3-service-agent/internal/llm/provider"
	"web3-service-agent/internal/logging"
	"web3-service-agent/internal/marketdata"
	"web3-service-agent/internal/mcp"
	"web3-service-agent/internal/prompt"
	"web3-service-agent/internal/session"
	"web3-service-agent/internal/tool"
	"web3-service-agent/tools"
	wallettool "web3-service-agent/tools/wallet"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	if cfg.Service == nil {
		_, _ = fmt.Fprintln(os.Stderr, "service config is required")
		os.Exit(1)
	}

	logger, err := logging.Setup(cfg.Log.Path)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "setup logging: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := logger.Close(); err != nil {
			logging.Errorf("close logger: %v", err)
		}
	}()

	fatalf := func(format string, args ...any) {
		logging.Errorf(format, args...)
		os.Exit(1)
	}

	if err := marketdata.Configure(&cfg); err != nil {
		fatalf("configure market data provider: %v", err)
	}
	wallettool.Configure(&cfg)

	sessionManager := session.NewManager()
	promptBuilder, err := prompt.NewBuilder(cfg.Service.Name)
	if err != nil {
		fatalf("create prompt builder: %v", err)
	}
	toolRegistry := tool.NewRegistry()
	mcpRegistry := mcp.NewRegistry()
	_ = mcpRegistry

	if err := tools.RegisterAll(toolRegistry); err != nil {
		fatalf("register tools: %v", err)
	}

	llmClient, err := provider.NewClient(&cfg)
	if err != nil {
		fatalf("create llm client: %v", err)
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
		ErrorLog:          logging.StandardLogger(logging.LevelError),
		ReadHeaderTimeout: 10 * time.Second,
	}
	homeURL := homepageURL(cfg.Service.ListenAddr())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logging.Infof("web3-service-agent listening on %s", cfg.Service.ListenAddr())
		logging.Infof("web3-service-agent home page: %s", homeURL)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			fatalf("http server failed: %v", err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			fatalf("http shutdown failed: %v", err)
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
