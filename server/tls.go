package server

import (
	_ "embed"
	"encoding/asn1"
	"github.com/go-base-lib/cacenter"
	"github.com/tjfoc/gmsm/gmtls"
	sm2x509 "github.com/tjfoc/gmsm/x509"
	"net"
)

//go:embed root.crt
var rootPemCert []byte

//go:embed root.key
var rootPemKey []byte

var (
	certParser   = new(cacenter.Parser)
	certProducer = new(cacenter.Producer)

	rootCertNo string
	rootPool   *sm2x509.CertPool

	certTypeId = asn1.ObjectIdentifier{2, 8, 28, 1}

	trustCertHash = make(map[string]struct{})
)

var TlsManager tlsManagerInterface

type certAlgType uint8

const (
	algTypeSm2 certAlgType = iota
	algTypeRsa
)

type CertResult struct {
	CertPem       string
	PrivateKeyPem string
}

type tlsManagerInterface interface {
	// GetTlsServerConfig 获取
	GetTlsServerConfig(name string, dnsName string, ip net.IP) (*gmtls.Config, error)
	GetTlsClientConfig(name string, dnsName string, ip net.IP) (*gmtls.Config, error)
	GeneratorTlsServerKeyPairs(name string, dnsName string, ip net.IP) ([]gmtls.Certificate, error)
	GeneratorTlsClientKeyPairs(name string, dnsName string, ip net.IP) ([]gmtls.Certificate, error)
	GeneratorServerPemCerts(name string, dnsName string, ip net.IP) ([]*CertResult, error)
	GeneratorClientPemCert(name string, dnsName string, ip net.IP) (*CertResult, error)
}

func newTlsManagerCommonCache(impl tlsManagerInterface) tlsManagerInterface {
	return &tlsManagerCommonCache{
		impl:                 impl,
		tlsClientConfigCache: make(map[string]*gmtls.Config, 16),
		tlsServerConfigCache: make(map[string]*gmtls.Config, 16),
	}
}

type tlsManagerCommonCache struct {
	impl                 tlsManagerInterface
	tlsServerConfigCache map[string]*gmtls.Config
	tlsClientConfigCache map[string]*gmtls.Config
}

func (t *tlsManagerCommonCache) GetTlsServerConfig(name string, dnsName string, ip net.IP) (*gmtls.Config, error) {
	if config, ok := t.tlsServerConfigCache[name]; ok {
		return config, nil
	}

	config, err := t.impl.GetTlsServerConfig(name, dnsName, ip)
	if err != nil {
		return nil, err
	}
	t.tlsServerConfigCache[name] = config
	return config, nil
}

func (t *tlsManagerCommonCache) GetTlsClientConfig(name string, dnsName string, ip net.IP) (*gmtls.Config, error) {
	if config, ok := t.tlsClientConfigCache[name]; ok {
		return config, nil
	}

	config, err := t.impl.GetTlsClientConfig(name, dnsName, ip)
	if err != nil {
		return nil, err
	}
	t.tlsClientConfigCache[name] = config
	return config, nil
}

func (t *tlsManagerCommonCache) GeneratorTlsServerKeyPairs(name string, dnsName string, ip net.IP) (res []gmtls.Certificate, err error) {
	certs, err := t.impl.GeneratorServerPemCerts(name, dnsName, ip)
	if err != nil {
		return nil, err
	}

	res = make([]gmtls.Certificate, 0, len(certs))
	for i := range certs {
		certInfo := certs[i]
		pair, err := gmtls.X509KeyPair([]byte(certInfo.CertPem), []byte(certInfo.PrivateKeyPem))
		if err != nil {
			return nil, err
		}
		res = append(res, pair)
	}
	return
}

func (t *tlsManagerCommonCache) GeneratorTlsClientKeyPairs(name string, dnsName string, ip net.IP) (res []gmtls.Certificate, err error) {
	certs, err := t.impl.GeneratorServerPemCerts(name, dnsName, ip)
	if err != nil {
		return nil, err
	}

	res = make([]gmtls.Certificate, 0, len(certs))
	for i := range certs {
		certInfo := certs[i]
		pair, err := gmtls.X509KeyPair([]byte(certInfo.CertPem), []byte(certInfo.PrivateKeyPem))
		if err != nil {
			return nil, err
		}
		res = append(res, pair)
	}
	return
}

func (t *tlsManagerCommonCache) GeneratorServerPemCerts(name string, dnsName string, ip net.IP) ([]*CertResult, error) {
	return t.impl.GeneratorServerPemCerts(name, dnsName, ip)
}

func (t *tlsManagerCommonCache) GeneratorClientPemCert(name string, dnsName string, ip net.IP) (*CertResult, error) {
	return t.impl.GeneratorClientPemCert(name, dnsName, ip)
}

func init() {
	rootPool = sm2x509.NewCertPool()
	rootPool.AppendCertsFromPEM(rootPemCert)
}
