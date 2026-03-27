// Package fingerprint - TLS fingerprint randomization and anti-detection features.
// Varies JA3/JA4 fingerprint by randomizing cipher suite order and TLS extensions.
package fingerprint

import (
	"crypto/tls"
	"math/rand/v2"
	"net/http"
	"time"
)

// BrowserProfile represents a set of TLS/HTTP parameters mimicking a specific browser.
type BrowserProfile struct {
	Name         string
	TLSVersion   uint16
	CipherSuites []uint16
	H2Settings   map[string]uint32
	WindowSize   int
}

// Profiles contains predefined browser fingerprints.
var Profiles = []BrowserProfile{
	{
		Name:       "Chrome 131",
		TLSVersion: tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		},
		WindowSize: 65535,
	},
	{
		Name:       "Firefox 134",
		TLSVersion: tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		},
		WindowSize: 131072,
	},
	{
		Name:       "Safari 18",
		TLSVersion: tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
		WindowSize: 65535,
	},
	{
		Name:       "Edge 131",
		TLSVersion: tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		WindowSize: 65535,
	},
}

// RandomProfile returns a random browser fingerprint profile.
func RandomProfile() BrowserProfile {
	return Profiles[rand.IntN(len(Profiles))]
}

// ApplyToTransport configures an http.Transport with the given browser profile's TLS settings.
func ApplyToTransport(transport *http.Transport, profile BrowserProfile) {
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	}

	transport.TLSClientConfig.MinVersion = tls.VersionTLS12
	transport.TLSClientConfig.MaxVersion = profile.TLSVersion
	transport.TLSClientConfig.CipherSuites = shuffleCiphers(profile.CipherSuites)

	// Randomize TLS session ticket support
	transport.TLSClientConfig.SessionTicketsDisabled = rand.IntN(10) < 2 // 20% chance disabled
}

// NewFingerprintedTransport creates an http.Transport with randomized TLS fingerprint.
func NewFingerprintedTransport() *http.Transport {
	profile := RandomProfile()
	transport := &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		MaxConnsPerHost:     10,
		ForceAttemptHTTP2:   true,
	}
	ApplyToTransport(transport, profile)
	return transport
}

// shuffleCiphers returns a shuffled copy of the cipher suite list.
// Slight randomization of order varies the JA3 fingerprint.
func shuffleCiphers(ciphers []uint16) []uint16 {
	shuffled := make([]uint16, len(ciphers))
	copy(shuffled, ciphers)

	// Fisher-Yates shuffle on the last N-1 elements (keep first TLS 1.3 cipher first)
	for i := len(shuffled) - 1; i > 1; i-- {
		j := 1 + rand.IntN(i)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}
	return shuffled
}

// GenerateCanvasHash creates a randomized canvas fingerprint hash.
func GenerateCanvasHash() string {
	return GenerateDeviceID() // 32-char hex is sufficient
}

// GenerateWebGLHash creates a randomized WebGL renderer hash.
func GenerateWebGLHash() string {
	renderers := []string{
		"ANGLE (Intel, Intel(R) UHD Graphics 630, OpenGL 4.5)",
		"ANGLE (NVIDIA, NVIDIA GeForce GTX 1660 Ti, OpenGL 4.5)",
		"ANGLE (AMD, AMD Radeon RX 580, OpenGL 4.5)",
		"ANGLE (Intel, Intel(R) Iris(R) Plus Graphics, OpenGL 4.1)",
		"ANGLE (Apple, Apple M1, OpenGL 4.1)",
		"ANGLE (NVIDIA, NVIDIA GeForce RTX 3060, OpenGL 4.5)",
	}
	return renderers[rand.IntN(len(renderers))]
}

// ViewerFingerprint holds all randomized fingerprint data for a viewer session.
type ViewerFingerprint struct {
	DeviceID     string `json:"deviceId"`
	CanvasHash   string `json:"canvasHash"`
	WebGLHash    string `json:"webglHash"`
	BrowserName  string `json:"browserName"`
	ScreenWidth  int    `json:"screenWidth"`
	ScreenHeight int    `json:"screenHeight"`
	ColorDepth   int    `json:"colorDepth"`
	Timezone     string `json:"timezone"`
	Language     string `json:"language"`
}

// GenerateFingerprint creates a complete randomized browser fingerprint.
func GenerateFingerprint() ViewerFingerprint {
	profile := RandomProfile()

	screens := [][2]int{{1920, 1080}, {2560, 1440}, {1366, 768}, {1536, 864}, {1440, 900}, {3840, 2160}}
	screen := screens[rand.IntN(len(screens))]

	timezones := []string{"America/New_York", "America/Chicago", "America/Denver", "America/Los_Angeles", "Europe/London", "Europe/Berlin", "Asia/Tokyo"}
	languages := []string{"en-US", "en-GB", "de-DE", "fr-FR", "es-ES", "pt-BR", "ja-JP"}

	return ViewerFingerprint{
		DeviceID:     GenerateDeviceID(),
		CanvasHash:   GenerateCanvasHash(),
		WebGLHash:    GenerateWebGLHash(),
		BrowserName:  profile.Name,
		ScreenWidth:  screen[0],
		ScreenHeight: screen[1],
		ColorDepth:   24,
		Timezone:     timezones[rand.IntN(len(timezones))],
		Language:     languages[rand.IntN(len(languages))],
	}
}
