package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sort"
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

	qrcode "github.com/skip2/go-qrcode"
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

	var runtimeEngine *agent.Runtime
	if cfg.Service.AIEnabled {
		llmClient, err := provider.NewClient(&cfg)
		if err != nil {
			fatalf("create llm client: %v", err)
		}
		runtimeEngine = agent.New(agent.Dependencies{
			Sessions:      sessionManager,
			PromptBuilder: promptBuilder,
			LLM:           llmClient,
			Tools:         toolRegistry,
			MaxSteps:      cfg.Service.AgentMaxSteps,
		})
	}

	server := &http.Server{
		Addr:              cfg.Service.ListenAddr(),
		Handler:           httpapi.New(runtimeEngine, sessionManager, cfg.Service.AIEnabled),
		ErrorLog:          logging.StandardLogger(logging.LevelError),
		ReadHeaderTimeout: 10 * time.Second,
	}
	listener, err := net.Listen("tcp", cfg.Service.ListenAddr())
	if err != nil {
		fatalf("http server failed: %v", err)
	}
	defer listener.Close()
	homeURL := homepageURL(listener.Addr().String())
	logging.Infof("web3-service-agent listening on %s", listener.Addr())
	logging.Infof("web3-service-agent home page: %s", homeURL)
	lanIPs, err := localLANIPv4s()
	if err != nil {
		logging.Warnf("discover local network addresses: %v", err)
	}
	lanURLs := lanHomepageURLs(listener.Addr().String(), lanIPs)
	if len(lanURLs) == 0 {
		logging.Warnf("no LAN address available for QR access; check the active network interface and listener address")
	}
	for _, lanURL := range lanURLs {
		logging.Infof("web3-service-agent LAN home page: %s", lanURL)
		qr, qrErr := terminalQRCode(lanURL)
		if qrErr != nil {
			logging.Warnf("generate LAN access QR code: %v", qrErr)
			continue
		}
		if outputInfo, statErr := os.Stdout.Stat(); statErr == nil && outputInfo.Mode()&os.ModeCharDevice != 0 {
			qr = colorizeTerminalQRCode(qr)
		}
		_, _ = fmt.Fprintf(os.Stdout, "\nScan to open wallet: %s\n%s\n", lanURL, qr)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
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

func localLANIPv4s() ([]net.IP, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var addresses []net.IP
	seen := make(map[string]bool)
	for _, networkInterface := range interfaces {
		if networkInterface.Flags&net.FlagUp == 0 || networkInterface.Flags&net.FlagLoopback != 0 {
			continue
		}
		interfaceAddresses, err := networkInterface.Addrs()
		if err != nil {
			continue
		}
		for _, interfaceAddress := range interfaceAddresses {
			ipNet, ok := interfaceAddress.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil || !ip.IsPrivate() || seen[ip.String()] {
				continue
			}
			seen[ip.String()] = true
			addresses = append(addresses, ip)
		}
	}
	sort.Slice(addresses, func(left, right int) bool {
		return bytesCompareIP(addresses[left], addresses[right]) < 0
	})
	return addresses, nil
}

func bytesCompareIP(left, right net.IP) int {
	for index := 0; index < net.IPv4len; index++ {
		if left[index] != right[index] {
			return int(left[index]) - int(right[index])
		}
	}
	return 0
}

func lanHomepageURLs(listenAddr string, addresses []net.IP) []string {
	host, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return nil
	}
	if host != "" && host != "0.0.0.0" && host != "::" {
		return nil
	}
	urls := make([]string, 0, len(addresses))
	for _, address := range addresses {
		if ip := address.To4(); ip != nil && ip.IsPrivate() {
			urls = append(urls, "http://"+net.JoinHostPort(ip.String(), port)+"/")
		}
	}
	return urls
}

func terminalQRCode(value string) (string, error) {
	qr, err := qrcode.New(value, qrcode.Medium)
	if err != nil {
		return "", err
	}
	// Terminal cells are about half as wide as they are tall. Pack two QR
	// module rows into each text row, keeping the rendered modules square.
	return qr.ToSmallString(true), nil
}

func colorizeTerminalQRCode(qr string) string {
	var result strings.Builder
	for _, line := range strings.Split(strings.TrimSuffix(qr, "\n"), "\n") {
		result.WriteString("\x1b[47m\x1b[30m")
		result.WriteString(line)
		result.WriteString("\x1b[0m\n")
	}
	return result.String()
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
