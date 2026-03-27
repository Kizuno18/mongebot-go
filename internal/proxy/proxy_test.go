package proxy

import (
	"testing"
)

func TestParseProxy_IPPort(t *testing.T) {
	p, err := ParseProxy("1.2.3.4:8080")
	if err != nil {
		t.Fatalf("ParseProxy error: %v", err)
	}
	if p.Host != "1.2.3.4" || p.Port != "8080" {
		t.Errorf("expected 1.2.3.4:8080, got %s:%s", p.Host, p.Port)
	}
	if p.Type != ProxyHTTP {
		t.Errorf("expected HTTP type, got %v", p.Type)
	}
}

func TestParseProxy_IPPortUserPass(t *testing.T) {
	p, err := ParseProxy("1.2.3.4:8080:user:pass")
	if err != nil {
		t.Fatalf("ParseProxy error: %v", err)
	}
	if p.Username != "user" || p.Password != "pass" {
		t.Errorf("expected user:pass, got %s:%s", p.Username, p.Password)
	}
}

func TestParseProxy_URL(t *testing.T) {
	p, err := ParseProxy("socks5://admin:secret@10.0.0.1:1080")
	if err != nil {
		t.Fatalf("ParseProxy error: %v", err)
	}
	if p.Host != "10.0.0.1" || p.Port != "1080" {
		t.Errorf("expected 10.0.0.1:1080, got %s:%s", p.Host, p.Port)
	}
	if p.Type != ProxySOCKS5 {
		t.Errorf("expected SOCKS5, got %v", p.Type)
	}
	if p.Username != "admin" {
		t.Errorf("expected username=admin, got %s", p.Username)
	}
}

func TestParseProxy_Invalid(t *testing.T) {
	_, err := ParseProxy("")
	if err == nil {
		t.Error("expected error for empty string")
	}

	_, err = ParseProxy("invalid")
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestProxyURL(t *testing.T) {
	p := &Proxy{Host: "1.2.3.4", Port: "8080", Type: ProxyHTTP}
	if p.URL() != "http://1.2.3.4:8080" {
		t.Errorf("expected http://1.2.3.4:8080, got %s", p.URL())
	}

	p.Username = "user"
	p.Password = "pass"
	if p.URL() != "http://user:pass@1.2.3.4:8080" {
		t.Errorf("expected http://user:pass@1.2.3.4:8080, got %s", p.URL())
	}
}

func TestManagerAcquireRelease(t *testing.T) {
	mgr := NewManager(RotationRoundRobin)
	mgr.AddBulk([]string{"1.1.1.1:8080", "2.2.2.2:8080", "3.3.3.3:8080"})

	p1 := mgr.Acquire()
	if p1 == nil {
		t.Fatal("expected proxy, got nil")
	}

	total, available, inUse := mgr.Count()
	if total != 3 || available != 2 || inUse != 1 {
		t.Errorf("expected 3/2/1, got %d/%d/%d", total, available, inUse)
	}

	mgr.Release(p1)
	_, available2, inUse2 := mgr.Count()
	if available2 != 3 || inUse2 != 0 {
		t.Errorf("after release: expected 3/0, got %d/%d", available2, inUse2)
	}
}

func TestManagerDeduplication(t *testing.T) {
	mgr := NewManager(RotationRandom)
	added, _ := mgr.AddBulk([]string{"1.1.1.1:8080", "1.1.1.1:8080", "2.2.2.2:8080"})
	if added != 2 {
		t.Errorf("expected 2 added (deduped), got %d", added)
	}
}

func TestManagerLoadFromFile(t *testing.T) {
	// Test that missing file doesn't error
	mgr := NewManager(RotationRandom)
	err := mgr.LoadFromFile("/nonexistent/path/proxies.txt")
	if err != nil {
		t.Errorf("expected no error for missing file, got %v", err)
	}
}
