package server

import (
	"crypto/x509"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

const (
	testName    = "BYPT LOCAL SERVER"
	testDnsName = "unix"
)

var (
	testIp = net.IPv4(127, 0, 0, 1)
)

func TestGeneratorPemCerts(t *testing.T) {
	a := assert.New(t)

	certs, err := TlsManager.GeneratorServerPemCerts(testName, testDnsName, testIp)
	if !a.NoError(err) {
		return
	}

	certLen := len(certs)
	if certLen == 1 {
		testServerRsaCert(a, certs)
		clientCert, err := TlsManager.GeneratorClientPemCert(testName, testDnsName, testIp)
		if !a.NoError(err) {
			return
		}

		testClientRsaCert(a, clientCert)
	} else if certLen == 2 {
		t.Error("SM2 证书测试方法未实现")
	} else {
		t.Error("错误的证书制作结果")
	}

}

func testClientRsaCert(a *assert.Assertions, cert *CertResult) {
	fmt.Println(cert.CertPem)
	fmt.Println(cert.PrivateKeyPem)
	_, err := certParser.ParseByRsaPriKeyPem(cert.PrivateKeyPem)
	if !a.NoError(err) {
		return
	}

	certInfo, err := certParser.ParseByCertPem(cert.CertPem)
	if !a.NoError(err) {
		return
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(rootPemCert)
	_, err = certInfo.Verify(x509.VerifyOptions{
		DNSName:   testDnsName,
		Roots:     pool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	if !a.NoError(err) {
		return
	}

	for i := range certInfo.Extensions {
		extension := certInfo.Extensions[i]
		if extension.Id.Equal(certTypeId) {
			if !a.Equal(extension.Value, []byte("signer"), "证书类型比对错误") {
				return
			}
			goto End
		}
	}
	panic("测试失败, 缺失扩展项")
End:
}

func testServerRsaCert(a *assert.Assertions, certs []*CertResult) {
	_, err := certParser.ParseByRsaPriKeyPem(certs[0].PrivateKeyPem)
	if !a.NoError(err) {
		return
	}

	certInfo, err := certParser.ParseByCertPem(certs[0].CertPem)
	if !a.NoError(err) {
		return
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(rootPemCert)

	_, err = certInfo.Verify(x509.VerifyOptions{
		DNSName:   testDnsName,
		Roots:     pool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	if !a.NoError(err) {
		return
	}

	for i := range certInfo.Extensions {
		extension := certInfo.Extensions[i]
		if extension.Id.Equal(certTypeId) {
			if !a.Equal(extension.Value, []byte("signer"), "证书类型比对错误") {
				return
			}
			goto End
		}
	}
	panic("测试失败, 缺失扩展项")
End:
}
