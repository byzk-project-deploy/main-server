package security

import (
	_ "embed"
	"encoding/asn1"
	"github.com/go-base-lib/cacenter"
	"github.com/tjfoc/gmsm/gmtls"
	sm2x509 "github.com/tjfoc/gmsm/x509"
	"net"
)

const (
	LinkLocalFlag  = "BYPT LOCAL SERVER"
	LinkRemoteFlag = "BYPT REMOTE SERVER"

	LinkRemoteDnsFlag = "command.manager.bypt"
)

func GetRemoteServerName(ip net.IP) string {
	return LinkRemoteFlag + " " + ip.String()
}

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

var Instance securityInterface

type certAlgType uint8

const (
	algTypeSm2 certAlgType = iota
	algTypeRsa
)

type CertResult struct {
	CertPem       string
	PrivateKeyPem string
}

type securityInterface interface {
	// GetTlsServerConfig 获取
	GetTlsServerConfig(name string, dnsName string, ip net.IP) (*gmtls.Config, error)
	GetTlsClientConfig(name string, dnsName string, ip net.IP) (*gmtls.Config, error)
	GeneratorTlsServerKeyPairs(name string, dnsName string, ip net.IP) ([]gmtls.Certificate, error)
	GeneratorTlsClientKeyPairs(name string, dnsName string, ip net.IP) ([]gmtls.Certificate, error)
	GeneratorServerPemCerts(name string, dnsName string, ip net.IP) ([]*CertResult, error)
	GeneratorClientPemCert(name string, dnsName string, ip net.IP) (*CertResult, error)
	DataEnvelope(data []byte) ([]byte, error)
	DataUnEnvelope(data []byte) ([]byte, error)
}

func newTlsManagerCommonCache(impl securityInterface) securityInterface {
	return &managerCommonCache{
		impl:                 impl,
		tlsClientConfigCache: make(map[string]*gmtls.Config, 16),
		tlsServerConfigCache: make(map[string]*gmtls.Config, 16),
	}
}

type managerCommonCache struct {
	impl                 securityInterface
	tlsServerConfigCache map[string]*gmtls.Config
	tlsClientConfigCache map[string]*gmtls.Config
}

func (t *managerCommonCache) DataEnvelope(data []byte) ([]byte, error) {
	return t.impl.DataEnvelope(data)
}

func (t *managerCommonCache) DataUnEnvelope(data []byte) ([]byte, error) {
	return t.impl.DataUnEnvelope(data)
}

func (t *managerCommonCache) GetTlsServerConfig(name string, dnsName string, ip net.IP) (*gmtls.Config, error) {
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

func (t *managerCommonCache) GetTlsClientConfig(name string, dnsName string, ip net.IP) (*gmtls.Config, error) {
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

func (t *managerCommonCache) GeneratorTlsServerKeyPairs(name string, dnsName string, ip net.IP) (res []gmtls.Certificate, err error) {
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

func (t *managerCommonCache) GeneratorTlsClientKeyPairs(name string, dnsName string, ip net.IP) (res []gmtls.Certificate, err error) {
	certs, err := t.impl.GeneratorClientPemCert(name, dnsName, ip)
	if err != nil {
		return nil, err
	}

	pair, err := gmtls.X509KeyPair([]byte(certs.CertPem), []byte(certs.PrivateKeyPem))
	if err != nil {
		return nil, err
	}

	//res = make([]gmtls.Certificate, 0, len(certs))
	//for i := range certs {
	//	certInfo := certs[i]
	//	pair, err := gmtls.X509KeyPair([]byte(certInfo.CertPem), []byte(certInfo.PrivateKeyPem))
	//	if err != nil {
	//		return nil, err
	//	}
	//	res = append(res, pair)
	//}
	return []gmtls.Certificate{pair}, nil
}

func (t *managerCommonCache) GeneratorServerPemCerts(name string, dnsName string, ip net.IP) ([]*CertResult, error) {
	return t.impl.GeneratorServerPemCerts(name, dnsName, ip)
}

func (t *managerCommonCache) GeneratorClientPemCert(name string, dnsName string, ip net.IP) (*CertResult, error) {
	return t.impl.GeneratorClientPemCert(name, dnsName, ip)
}

func init() {
	rootPool = sm2x509.NewCertPool()
	rootPool.AppendCertsFromPEM(rootPemCert)
}
