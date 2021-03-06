package webrtc

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"math/big"
	"time"

	"github.com/pions/webrtc/pkg/rtcerr"
)

// RTCCertificate represents a x509Cert used to authenticate WebRTC communications.
type RTCCertificate struct {
	privateKey crypto.PrivateKey
	x509Cert   *x509.Certificate
}

// NewRTCCertificate generates a new x509 compliant RTCCertificate to be used
// by DTLS for encrypting data sent over the wire. This method differs from
// GenerateCertificate by allowing to specify a template x509.Certificate to
// be used in order to define certificate parameters.
func NewRTCCertificate(key crypto.PrivateKey, tpl x509.Certificate) (*RTCCertificate, error) {
	var err error
	var certDER []byte
	switch sk := key.(type) {
	case *rsa.PrivateKey:
		pk := sk.Public()
		tpl.SignatureAlgorithm = x509.SHA256WithRSA
		certDER, err = x509.CreateCertificate(rand.Reader, &tpl, &tpl, pk, sk)
		if err != nil {
			return nil, &rtcerr.UnknownError{Err: err}
		}
	case *ecdsa.PrivateKey:
		pk := sk.Public()
		tpl.SignatureAlgorithm = x509.ECDSAWithSHA256
		certDER, err = x509.CreateCertificate(rand.Reader, &tpl, &tpl, pk, sk)
		if err != nil {
			return nil, &rtcerr.UnknownError{Err: err}
		}
	default:
		return nil, &rtcerr.NotSupportedError{Err: ErrPrivateKeyType}
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, &rtcerr.UnknownError{Err: err}
	}

	return &RTCCertificate{privateKey: key, x509Cert: cert}, nil
}

// Equals determines if two certificates are identical by comparing both the
// secretKeys and x509Certificates.
func (c RTCCertificate) Equals(o RTCCertificate) bool {
	switch cSK := c.privateKey.(type) {
	case *rsa.PrivateKey:
		if oSK, ok := o.privateKey.(*rsa.PrivateKey); ok {
			if cSK.N.Cmp(oSK.N) != 0 {
				return false
			}
			return c.x509Cert.Equal(o.x509Cert)
		}
		return false
	case *ecdsa.PrivateKey:
		if oSK, ok := o.privateKey.(*ecdsa.PrivateKey); ok {
			if cSK.X.Cmp(oSK.X) != 0 || cSK.Y.Cmp(oSK.Y) != 0 {
				return false
			}
			return c.x509Cert.Equal(o.x509Cert)
		}
		return false
	default:
		return false
	}
}

// Expires returns the timestamp after which this certificate is no longer valid.
func (c RTCCertificate) Expires() time.Time {
	if c.x509Cert == nil {
		return time.Time{}
	}
	return c.x509Cert.NotAfter
}

// GetFingerprints returns the list of certificate fingerprints, one of which
// is computed with the digest algorithm used in the certificate signature.
func (c RTCCertificate) GetFingerprints() {
	panic("not implemented yet.") // nolint
}

// GenerateCertificate causes the creation of an X.509 certificate and
// corresponding private key.
func GenerateCertificate(secretKey crypto.PrivateKey) (*RTCCertificate, error) {
	origin := make([]byte, 16)
	/* #nosec */
	if _, err := rand.Read(origin); err != nil {
		return nil, &rtcerr.UnknownError{Err: err}
	}

	// Max random value, a 130-bits integer, i.e 2^130 - 1
	maxBigInt := new(big.Int)
	/* #nosec */
	maxBigInt.Exp(big.NewInt(2), big.NewInt(130), nil).Sub(maxBigInt, big.NewInt(1))
	/* #nosec */
	serialNumber, err := rand.Int(rand.Reader, maxBigInt)
	if err != nil {
		return nil, &rtcerr.UnknownError{Err: err}
	}

	return NewRTCCertificate(secretKey, x509.Certificate{
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
		BasicConstraintsValid: true,
		NotBefore:             time.Now(),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		NotAfter:              time.Now().AddDate(0, 1, 0),
		SerialNumber:          serialNumber,
		Version:               2,
		Subject:               pkix.Name{CommonName: hex.EncodeToString(origin)},
		IsCA:                  true,
	})
}
