//go:build rsa

package security

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"github.com/byzk-project-deploy/main-server/errors"
	"github.com/go-base-lib/cacenter"
	"github.com/go-base-lib/coderutils"
	"github.com/tjfoc/gmsm/gmtls"
	sm2x509 "github.com/tjfoc/gmsm/x509"
	"golang.org/x/exp/slices"
	"net"
)

type rsaKeyInfo struct {
	producer      *cacenter.RsaCertProducer
	privateKeyPem string
	err           error
}

var (
	rootCert   *x509.Certificate
	rootKey    *rsa.PrivateKey
	rootPubKey *rsa.PublicKey

	singerType = []byte("signer")

	keyInfoChan = make(chan *rsaKeyInfo, 10)
)

func init() {
	var err error
	rootCert, err = certParser.ParseByCertPemBytes(rootPemCert)
	if err != nil {
		errors.ExitCertError.Println("根证书解析失败: %s", err.Error())
	}

	rootPubKey = rootCert.PublicKey.(*rsa.PublicKey)

	rootKey, err = certParser.ParseByRsaPriKeyPemBytes(rootPemKey)
	if err != nil {
		errors.ExitCertError.Println("解析根证书密钥失败: %s", err.Error())
	}

	rootCertNo = rootCert.SerialNumber.String()

	Instance = newTlsManagerCommonCache(&rsaTlsManagerImpl{})

	go func() {
		for {
			rsaCertProducer, privateKey, err := certProducer.Rsa(4096)
			keyInfoChan <- &rsaKeyInfo{
				rsaCertProducer, privateKey, err,
			}
		}

	}()
}

type rsaTlsManagerImpl struct {
}

func (r *rsaTlsManagerImpl) DataEnvelope(data []byte) ([]byte, error) {
	sm4RandomKey := coderutils.Sm4RandomKey()
	sm4EncBytes, err := coderutils.Sm4Encrypt(sm4RandomKey, data)
	if err != nil {
		return nil, err
	}
	sm2EncBytes, err := rsa.EncryptPKCS1v15(rand.Reader, rootPubKey, sm4RandomKey)
	if err != nil {
		return nil, err
	}

	return bytes.Join([][]byte{
		sm2EncBytes,
		sm4EncBytes,
	}, nil), nil
}

func (r *rsaTlsManagerImpl) DataUnEnvelope(data []byte) ([]byte, error) {
	dataLen := len(data)
	if dataLen < 513 {
		return nil, fmt.Errorf("非法数据内容")
	}

	sm4Key, err := rsa.DecryptPKCS1v15(rand.Reader, rootKey, data[:512])
	if err != nil {
		return nil, fmt.Errorf("解析密钥格式失败: %s", err.Error())
	}

	decrypt, err := coderutils.Sm4Decrypt(sm4Key, data[512:])
	if err != nil {
		return nil, fmt.Errorf("数据解析失败: %s", err.Error())
	}
	return decrypt, nil
}

func (r *rsaTlsManagerImpl) GetTlsServerConfig(name string, dnsName string, ip net.IP) (*gmtls.Config, error) {
	keyPairs, err := Instance.GeneratorTlsServerKeyPairs(name, dnsName, ip)
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

			if len(signCertChains[0].IPAddresses) == 0 ||
				!signCertChains[0].IPAddresses[0].Equal(ip) ||
				signCertChains[0].Subject.CommonName != name ||
				!slices.Contains(signCertChains[0].Subject.Organization, "bypt") ||
				!slices.Contains(signCertChains[0].DNSNames, dnsName) {
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
	keyPairs, err := Instance.GeneratorTlsClientKeyPairs(name, dnsName, ip)
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
	keyInfo := <-keyInfoChan
	rsaCertProducer := keyInfo.producer
	privateKey := keyInfo.privateKeyPem
	err := keyInfo.err
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
	keyInfo := <-keyInfoChan
	rsaCertProducer := keyInfo.producer
	privateKey := keyInfo.privateKeyPem
	err := keyInfo.err
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
