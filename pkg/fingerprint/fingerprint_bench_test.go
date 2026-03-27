package fingerprint

import (
	"testing"
)

func BenchmarkGenerateDeviceID(b *testing.B) {
	for b.Loop() {
		GenerateDeviceID()
	}
}

func BenchmarkGenerateNonce(b *testing.B) {
	for b.Loop() {
		GenerateNonce(30)
	}
}

func BenchmarkGenerateFingerprint(b *testing.B) {
	for b.Loop() {
		GenerateFingerprint()
	}
}

func BenchmarkRandomProfile(b *testing.B) {
	for b.Loop() {
		RandomProfile()
	}
}

func BenchmarkShuffleCiphers(b *testing.B) {
	ciphers := Profiles[0].CipherSuites
	for b.Loop() {
		shuffleCiphers(ciphers)
	}
}
