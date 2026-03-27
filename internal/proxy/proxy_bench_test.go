package proxy

import (
	"testing"
)

func BenchmarkParseProxy_IPPort(b *testing.B) {
	for b.Loop() {
		ParseProxy("192.168.1.1:8080")
	}
}

func BenchmarkParseProxy_IPPortUserPass(b *testing.B) {
	for b.Loop() {
		ParseProxy("192.168.1.1:8080:admin:secret123")
	}
}

func BenchmarkParseProxy_URL(b *testing.B) {
	for b.Loop() {
		ParseProxy("socks5://admin:secret@10.0.0.1:1080")
	}
}

func BenchmarkManagerAcquire_RoundRobin(b *testing.B) {
	mgr := NewManager(RotationRoundRobin)
	for i := range 100 {
		mgr.AddBulk([]string{
			"1.1.1." + string(rune('0'+i%10)) + ":8080",
		})
	}
	b.ResetTimer()
	for b.Loop() {
		p := mgr.Acquire()
		if p != nil {
			mgr.Release(p)
		}
	}
}

func BenchmarkManagerAcquire_Random(b *testing.B) {
	mgr := NewManager(RotationRandom)
	mgr.AddBulk([]string{
		"1.1.1.1:8080", "2.2.2.2:8080", "3.3.3.3:8080",
		"4.4.4.4:8080", "5.5.5.5:8080", "6.6.6.6:8080",
	})
	b.ResetTimer()
	for b.Loop() {
		p := mgr.Acquire()
		if p != nil {
			mgr.Release(p)
		}
	}
}

func BenchmarkProxyURL(b *testing.B) {
	p := &Proxy{Host: "1.2.3.4", Port: "8080", Username: "user", Password: "pass", Type: ProxyHTTP}
	for b.Loop() {
		_ = p.URL()
	}
}
