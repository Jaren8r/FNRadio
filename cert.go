package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	crypt32                              = syscall.NewLazyDLL("crypt32.dll")
	procCertAddEncodedCertificateToStore = crypt32.NewProc("CertAddEncodedCertificateToStore")
)

func importCAToSystemRoot(cert *x509.Certificate) error {
	data := cert.Raw

	root, err := syscall.UTF16PtrFromString("root")
	if err != nil {
		return err
	}

	store, err := syscall.CertOpenStore(10, 0, 0, windows.CERT_SYSTEM_STORE_CURRENT_USER, uintptr(unsafe.Pointer(root)))
	if err != nil {
		return err
	}

	defer syscall.CertCloseStore(store, 0) // nolint:errcheck

	_, _, err = procCertAddEncodedCertificateToStore.Call(uintptr(store), 1, uintptr(unsafe.Pointer(&data[0])), uintptr(uint(len(data))), 4, 0)
	if !errors.Is(err, windows.ERROR_SUCCESS) {
		return err
	}

	return nil
}

func systemHasCertificate(certToFind *x509.Certificate) bool {
	root, err := syscall.UTF16PtrFromString("root")
	if err != nil {
		return false
	}

	store, err := syscall.CertOpenStore(10, 0, 0, windows.CERT_SYSTEM_STORE_CURRENT_USER, uintptr(unsafe.Pointer(root)))
	if err != nil {
		return false
	}

	defer syscall.CertCloseStore(store, 0) // nolint:errcheck

	var cert *syscall.CertContext

	for {
		cert, err = syscall.CertEnumCertificatesInStore(store, cert)
		if err != nil {
			break
		}

		buf := (*[1 << 20]byte)(unsafe.Pointer(cert.EncodedCert))[:]
		buf2 := make([]byte, cert.Length)

		copy(buf2, buf)

		c, err := x509.ParseCertificate(buf2)
		if err != nil {
			panic(err)
		}

		if c.Equal(certToFind) {
			return true
		}
	}

	return false
}

func setupSSL() *tls.Certificate {
	var err error

	certificate, _ := getCertificate()

	if certificate != nil {
		x509Certificate, err := x509.ParseCertificate(certificate.Certificate[0])
		if err != nil {
			panic(err)
		}

		if !systemHasCertificate(x509Certificate) {
			err = importCAToSystemRoot(x509Certificate)
			if err != nil {
				panic(err)
			}
		}

		return certificate
	}

	certificate, err = createCACertificate()
	if err != nil {
		panic(err)
	}

	return certificate
}

func getCertificate() (*tls.Certificate, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `SOFTWARE\FNRadio`, registry.QUERY_VALUE)
	if err != nil {
		return nil, err
	}

	regCert, _, err := k.GetBinaryValue("SSLCertificate")
	if err != nil {
		return nil, err
	}

	regKey, _, err := k.GetBinaryValue("SSLPrivateKey")
	if err != nil {
		return nil, err
	}

	key, err := x509.ParsePKCS1PrivateKey(regKey)
	if err != nil {
		return nil, err
	}

	return &tls.Certificate{
		Certificate: [][]byte{regCert},
		PrivateKey:  key,
	}, nil
}

func createCACertificate() (*tls.Certificate, error) {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(0),
		Subject: pkix.Name{
			CommonName: "FNRadio",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}

	parsed, err := x509.ParseCertificate(caBytes)
	if err != nil {
		return nil, err
	}

	err = importCAToSystemRoot(parsed)
	if err != nil {
		return nil, err
	}

	k, _, err := registry.CreateKey(registry.CURRENT_USER, `SOFTWARE\FNRadio`, registry.SET_VALUE)
	if err != nil {
		panic(err)
	}

	_ = k.SetBinaryValue("SSLCertificate", caBytes)

	_ = k.SetBinaryValue("SSLPrivateKey", x509.MarshalPKCS1PrivateKey(caPrivKey))

	_ = k.Close()

	return &tls.Certificate{
		Certificate: [][]byte{caBytes},
		PrivateKey:  caPrivKey,
	}, nil
}
