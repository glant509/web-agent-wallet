package main

import "testing"

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
