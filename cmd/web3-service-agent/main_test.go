package main

import (
	"net"
	"strings"
	"testing"

	qrcode "github.com/skip2/go-qrcode"
)

func TestHomepageURL(t *testing.T) {
	testCases := []struct {
		name       string
		listenAddr string
		want       string
	}{
		{
			name:       "port only",
			listenAddr: ":8080",
			want:       "http://localhost:8080/",
		},
		{
			name:       "wildcard host",
			listenAddr: "0.0.0.0:9090",
			want:       "http://localhost:9090/",
		},
		{
			name:       "explicit host",
			listenAddr: "127.0.0.1:7001",
			want:       "http://127.0.0.1:7001/",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := homepageURL(testCase.listenAddr)
			if got != testCase.want {
				t.Fatalf("homepageURL(%q) = %q, want %q", testCase.listenAddr, got, testCase.want)
			}
		})
	}
}

func TestLANHomepageURLs(t *testing.T) {
	addresses := []net.IP{net.ParseIP("192.168.1.25"), net.ParseIP("10.0.0.8"), net.ParseIP("127.0.0.1")}
	for _, listenAddr := range []string{":8081", "[::]:8081", "0.0.0.0:8081"} {
		urls := lanHomepageURLs(listenAddr, addresses)
		if len(urls) != 2 || urls[0] != "http://192.168.1.25:8081/" || urls[1] != "http://10.0.0.8:8081/" {
			t.Fatalf("lanHomepageURLs(%q) = %v", listenAddr, urls)
		}
	}
	if urls := lanHomepageURLs("127.0.0.1:8081", addresses); len(urls) != 0 {
		t.Fatalf("loopback-only listener must not advertise LAN URL: %v", urls)
	}
}

func TestTerminalQRCode(t *testing.T) {
	url := "http://192.168.1.25:8081/"
	qr, err := terminalQRCode(url)
	if err != nil {
		t.Fatalf("generate QR code: %v", err)
	}
	full, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		t.Fatalf("generate full-size QR code: %v", err)
	}
	fullRows := strings.Count(full.ToString(false), "\n")
	rows := strings.Split(strings.TrimSuffix(qr, "\n"), "\n")
	if !strings.ContainsAny(qr, "█▀▄") || len(rows) != (fullRows+1)/2 {
		t.Fatalf("expected printable QR code, got %q", qr)
	}
	blocks := map[rune]int{' ': 0, '▀': 1, '▄': 2, '█': 3}
	bitmap := full.Bitmap()
	for rowIndex, row := range rows {
		cells := []rune(row)
		if len(cells) != len(bitmap) {
			t.Fatalf("expected one text column per QR module, got width %d and module count %d", len(cells), len(bitmap))
		}
		if len(cells) < 2*len(rows)-1 || len(cells) > 2*len(rows) {
			t.Fatalf("expected visually square terminal QR, got %d columns and %d rows", len(cells), len(rows))
		}
		for columnIndex, cell := range cells {
			mask, ok := blocks[cell]
			if !ok {
				t.Fatalf("unexpected QR character %q", cell)
			}
			for dy := 0; dy < 2; dy++ {
				y := rowIndex*2 + dy
				if y < len(bitmap) && (mask&(1<<dy) != 0) != bitmap[y][columnIndex] {
					t.Fatalf("QR module changed at row %d, column %d", y, columnIndex)
				}
			}
		}
	}
	colored := colorizeTerminalQRCode(qr)
	if !strings.Contains(colored, "\x1b[47m\x1b[30m") || !strings.Contains(colored, "\x1b[0m") {
		t.Fatalf("expected high-contrast terminal QR code")
	}
}
