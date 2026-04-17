package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// ensureCert loads cert+key from dir if they exist, otherwise generates a
// new self-signed ECDSA P-256 certificate valid for 10 years and saves it.
// All local LAN IPs are included in the SAN so the cert is valid on any
// interface without needing to regenerate it when the IP changes.
func ensureCert(dir string) (tls.Certificate, error) {
	cf := filepath.Join(dir, "cert.pem")
	kf := filepath.Join(dir, "key.pem")

	// Reuse existing cert
	if _, err := os.Stat(cf); err == nil {
		if _, err2 := os.Stat(kf); err2 == nil {
			cert, err := tls.LoadX509KeyPair(cf, kf)
			if err == nil {
				debugf("SSL cert loaded from disk (%s)", cf)
				return cert, nil
			}
		}
	}

	printInfo("Generating self-signed SSL certificate (one-time)…")

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("keygen: %w", err)
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "barcodehid"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}

	// Add all non-loopback IPs so cert is valid regardless of which
	// interface the phone connects through
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && !ip.IsLoopback() && ip.To4() != nil {
				tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
			}
		}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create cert: %w", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("marshal key: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM  := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	_ = os.WriteFile(cf, certPEM, 0644)
	_ = os.WriteFile(kf, keyPEM, 0600)

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("keypair: %w", err)
	}

	printOK(fmt.Sprintf("Certificate saved → %s", cf))
	return cert, nil
}
