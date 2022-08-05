//go:build rsa

package server

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"github.com/byzk-project-deploy/main-server/errors"
	"github.com/go-base-lib/cacenter"
	"github.com/tjfoc/gmsm/gmtls"
	sm2x509 "github.com/tjfoc/gmsm/x509"
	"golang.org/x/exp/slices"
	"net"
)

var (
	rootCert *x509.Certificate
	rootKey  *rsa.PrivateKey

	singerType = []byte("signer")
)

func init() {
	var err error
	rootCert, err = certParser.ParseByCertPemBytes(rootPemCert)
	if err != nil {
		errors.ExitCertError.Println("根证书解析失败: %s", err.Error())
	}

	rootKey, err = certParser.ParseByRsaPriKeyPemBytes(rootPemKey)
	if err != nil {
		errors.ExitCertError.Println("解析根证书密钥失败: %s", err.Error())
	}

	rootCertNo = rootCert.SerialNumber.String()

	TlsManager = newTlsManagerCommonCache(&rsaTlsManagerImpl{})
}

type rsaTlsManagerImpl struct {
}

func (r *rsaTlsManagerImpl) GetTlsServerConfig(name string, dnsName string, ip net.IP) (*gmtls.Config, error) {
	keyPairs, err := TlsManager.GeneratorTlsServerKeyPairs(name, dnsName, ip)
	if err != nil {
		return nil, err
	}

	return &gmtls.Config{
		ClientCAs:          rootPool,
		ClientAuth:         gmtls.RequireAndVerifyClientCert,
		Certificates:       keyPairs,
		InsecureSkipVerify: false,
		ServerName:         dnsName,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*sm2x509.Certificate) (err error) {
			if verifiedChains == nil || len(verifiedChains) == 0 {
				return fmt.Errorf("verified chains fail")
			}

			certRawStr := string(bytes.Join(rawCerts, nil))
			if _, ok := trustCertHash[certRawStr]; ok {
				return nil
			}

			signCertChains := verifiedChains[0]
			if len(signCertChains) != 2 {
				return fmt.Errorf("client signer chains fail")
			}

			if signCertChains[0].Subject.CommonName != name ||
				!slices.Contains(signCertChains[0].Subject.Organization, "bypt") ||
				!slices.Contains(signCertChains[0].DNSNames, dnsName) ||
				len(signCertChains[0].IPAddresses) == 0 ||
				!signCertChains[0].IPAddresses[0].Equal(ip) {
				return fmt.Errorf("client signer cert fail")
			}

			for i := range signCertChains[0].Extensions {
				extension := signCertChains[0].Extensions[i]
				if extension.Id.Equal(certTypeId) {
					if bytes.Equal(extension.Value, singerType) {
						goto EndVerify
					}
					break
				}
			}

			return fmt.Errorf("client type verified fail")

		EndVerify:
			if signCertChains[1].SerialNumber.String() != rootCertNo {
				return fmt.Errorf("client root cert verified fail")
			}

			trustCertHash[certRawStr] = struct{}{}
			return nil
		},
	}, nil

}

func (r *rsaTlsManagerImpl) GetTlsClientConfig(name string, dnsName string, ip net.IP) (*gmtls.Config, error) {
	keyPairs, err := TlsManager.GeneratorTlsClientKeyPairs(name, dnsName, ip)
	if err != nil {
		return nil, err
	}

	return &gmtls.Config{
		RootCAs:            rootPool,
		ClientAuth:         gmtls.RequireAndVerifyClientCert,
		Certificates:       keyPairs,
		InsecureSkipVerify: false,
		ServerName:         dnsName,
	}, nil

}

func (r *rsaTlsManagerImpl) GeneratorTlsServerKeyPairs(name string, dnsName string, ip net.IP) ([]gmtls.Certificate, error) {
	// tlsManagerCommonCache 中已经实现, 此处无需再次实现
	return nil, nil
}

func (r *rsaTlsManagerImpl) GeneratorTlsClientKeyPairs(name string, dnsName string, ip net.IP) (res []gmtls.Certificate, err error) {
	// tlsManagerCommonCache 中已经实现, 此处无需再次实现
	return nil, nil
}

func (r *rsaTlsManagerImpl) GeneratorServerPemCerts(name string, dnsName string, ip net.IP) ([]*CertResult, error) {
	rsaCertProducer, privateKey, err := certProducer.Rsa(4096)
	if err != nil {
		return nil, err
	}

	certPem, err := rsaCertProducer.
		WithParent(rootCert, rootKey).
		WithExpire(&rootCert.NotBefore, &rootCert.NotAfter).
		WithSubject(pkix.Name{
			CommonName:   name,
			Organization: []string{"bypt"},
		}).
		WithTLSServer(cacenter.TLSAddrWithDNSName(dnsName), cacenter.TLSAddrWithIp(ip)).
		SettingCertInfo(func(parent, current *x509.Certificate) {
			current.ExtraExtensions = append(current.Extensions, pkix.Extension{
				Id:       certTypeId,
				Critical: false,
				Value:    []byte("signer"),
			})

		}).
		ToCertPem()

	if err != nil {
		return nil, err
	}

	return []*CertResult{
		{CertPem: certPem, PrivateKeyPem: privateKey},
	}, nil
}

func (r *rsaTlsManagerImpl) GeneratorClientPemCert(name string, dnsName string, ip net.IP) (*CertResult, error) {
	rsaCertProducer, privateKey, err := certProducer.Rsa(4096)
	if err != nil {
		return nil, err
	}

	certPem, err := rsaCertProducer.
		WithParent(rootCert, rootKey).
		WithExpire(&rootCert.NotBefore, &rootCert.NotAfter).
		WithSubject(pkix.Name{
			CommonName:   name,
			Organization: []string{"bypt"},
		}).
		WithTLSClient().
		SettingCertInfo(func(parent, current *x509.Certificate) {
			current.ExtraExtensions = append(current.Extensions, pkix.Extension{
				Id:       certTypeId,
				Critical: false,
				Value:    []byte("signer"),
			})
			current.IPAddresses = []net.IP{ip}
			current.DNSNames = []string{dnsName}
		}).
		ToCertPem()
	if err != nil {
		return nil, err
	}
	return &CertResult{
		CertPem:       certPem,
		PrivateKeyPem: privateKey,
	}, nil
}
