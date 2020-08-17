package main

import (
	"bytes"
	"crypto/rand"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/letsencrypt/boulder/pkcs11helpers"
	"github.com/letsencrypt/boulder/test"
	"github.com/miekg/pkcs11"
)

// samplePubkey returns a slice of bytes containing an encoded
// SubjectPublicKeyInfo for an example public key.
func samplePubkey() []byte {
	pubKey, err := hex.DecodeString("3059301306072a8648ce3d020106082a8648ce3d03010703420004b06745ef0375c9c54057098f077964e18d3bed0aacd54545b16eab8c539b5768cc1cea93ba56af1e22a7a01c33048c8885ed17c9c55ede70649b707072689f5e")
	if err != nil {
		panic(err)
	}
	return pubKey
}

func realRand(_ pkcs11.SessionHandle, length int) ([]byte, error) {
	r := make([]byte, length)
	_, err := rand.Read(r)
	return r, err
}

func TestParseOID(t *testing.T) {
	_, err := parseOID("")
	test.AssertError(t, err, "parseOID accepted an empty OID")
	_, err = parseOID("a.b.c")
	test.AssertError(t, err, "parseOID accepted an OID containing non-ints")
	oid, err := parseOID("1.2.3")
	test.AssertNotError(t, err, "parseOID failed with a valid OID")
	test.Assert(t, oid.Equal(asn1.ObjectIdentifier{1, 2, 3}), "parseOID returned incorrect OID")
}

func TestMakeTemplate(t *testing.T) {
	s, ctx := pkcs11helpers.NewSessionWithMock()
	profile := &certProfile{}
	randReader := newRandReader(s)
	pubKey := samplePubkey()

	profile.NotBefore = "1234"
	_, err := makeTemplate(randReader, profile, pubKey, rootCert)
	test.AssertError(t, err, "makeTemplate didn't fail with invalid not before")

	profile.NotBefore = "2018-05-18 11:31:00"
	profile.NotAfter = "1234"
	_, err = makeTemplate(randReader, profile, pubKey, rootCert)
	test.AssertError(t, err, "makeTemplate didn't fail with invalid not after")

	profile.NotAfter = "2018-05-18 11:31:00"
	profile.SignatureAlgorithm = "nope"
	_, err = makeTemplate(randReader, profile, pubKey, rootCert)
	test.AssertError(t, err, "makeTemplate didn't fail with invalid signature algorithm")

	profile.SignatureAlgorithm = "SHA256WithRSA"
	ctx.GenerateRandomFunc = func(pkcs11.SessionHandle, int) ([]byte, error) {
		return nil, errors.New("bad")
	}
	_, err = makeTemplate(randReader, profile, pubKey, rootCert)
	test.AssertError(t, err, "makeTemplate didn't fail when GenerateRandom failed")

	ctx.GenerateRandomFunc = realRand

	_, err = makeTemplate(randReader, profile, pubKey, rootCert)
	test.AssertError(t, err, "makeTemplate didn't fail with empty key usages")

	profile.KeyUsages = []string{"asd"}
	_, err = makeTemplate(randReader, profile, pubKey, rootCert)
	test.AssertError(t, err, "makeTemplate didn't fail with invalid key usages")

	profile.KeyUsages = []string{"Digital Signature", "CRL Sign"}
	profile.Policies = []policyInfoConfig{{}}
	_, err = makeTemplate(randReader, profile, pubKey, rootCert)
	test.AssertError(t, err, "makeTemplate didn't fail with invalid policy OID")

	profile.Policies = []policyInfoConfig{{OID: "1.2.3"}, {OID: "1.2.3.4", CPSURI: "hello"}}
	profile.CommonName = "common name"
	profile.Organization = "organization"
	profile.Country = "country"
	profile.OCSPURL = "http://ocsp.example.com"
	profile.CRLURL = "http://crl.example.com"
	profile.IssuerURL = "http://issuer.example.com"
	cert, err := makeTemplate(randReader, profile, pubKey, rootCert)
	test.AssertNotError(t, err, "makeTemplate failed when everything worked as expected")
	test.AssertEquals(t, cert.Subject.CommonName, profile.CommonName)
	test.AssertEquals(t, len(cert.Subject.Organization), 1)
	test.AssertEquals(t, cert.Subject.Organization[0], profile.Organization)
	test.AssertEquals(t, len(cert.Subject.Country), 1)
	test.AssertEquals(t, cert.Subject.Country[0], profile.Country)
	test.AssertEquals(t, len(cert.OCSPServer), 1)
	test.AssertEquals(t, cert.OCSPServer[0], profile.OCSPURL)
	test.AssertEquals(t, len(cert.CRLDistributionPoints), 1)
	test.AssertEquals(t, cert.CRLDistributionPoints[0], profile.CRLURL)
	test.AssertEquals(t, len(cert.IssuingCertificateURL), 1)
	test.AssertEquals(t, cert.IssuingCertificateURL[0], profile.IssuerURL)
	test.AssertEquals(t, cert.KeyUsage, x509.KeyUsageDigitalSignature|x509.KeyUsageCRLSign)
	test.AssertEquals(t, len(cert.ExtraExtensions), 1)
	test.AssertEquals(t, len(cert.ExtKeyUsage), 0)

	cert, err = makeTemplate(randReader, profile, pubKey, intermediateCert)
	test.AssertNotError(t, err, "makeTemplate failed when everything worked as expected")
	test.Assert(t, cert.MaxPathLenZero, "MaxPathLenZero not set in intermediate template")
	test.AssertEquals(t, len(cert.ExtKeyUsage), 2)
	test.AssertEquals(t, cert.ExtKeyUsage[0], x509.ExtKeyUsageClientAuth)
	test.AssertEquals(t, cert.ExtKeyUsage[1], x509.ExtKeyUsageServerAuth)
}

func TestMakeTemplateCrossCertificate(t *testing.T) {
	s, ctx := pkcs11helpers.NewSessionWithMock()
	randReader := newRandReader(s)
	pubKey := samplePubkey()
	profile := &certProfile{
		SignatureAlgorithm: "SHA256WithRSA",
		CommonName:         "common name",
		Organization:       "organization",
		Country:            "country",
		KeyUsages:          []string{"Digital Signature", "CRL Sign"},
		OCSPURL:            "http://ocsp.example.com",
		CRLURL:             "http://crl.example.com",
		IssuerURL:          "http://issuer.example.com",
		NotAfter:           "2018-05-18 11:31:00",
		NotBefore:          "2018-05-18 11:31:00",
	}

	ctx.GenerateRandomFunc = realRand

	cert, err := makeTemplate(randReader, profile, pubKey, crossCert)
	test.AssertNotError(t, err, "makeTemplate failed when everything worked as expected")
	test.Assert(t, !cert.MaxPathLenZero, "MaxPathLenZero was set in cross-sign")
	test.AssertEquals(t, len(cert.ExtKeyUsage), 0)
}

func TestMakeTemplateOCSP(t *testing.T) {
	s, ctx := pkcs11helpers.NewSessionWithMock()
	ctx.GenerateRandomFunc = realRand
	randReader := newRandReader(s)
	profile := &certProfile{
		SignatureAlgorithm: "SHA256WithRSA",
		CommonName:         "common name",
		Organization:       "organization",
		Country:            "country",
		OCSPURL:            "http://ocsp.example.com",
		CRLURL:             "http://crl.example.com",
		IssuerURL:          "http://issuer.example.com",
		NotAfter:           "2018-05-18 11:31:00",
		NotBefore:          "2018-05-18 11:31:00",
	}
	pubKey := samplePubkey()

	cert, err := makeTemplate(randReader, profile, pubKey, ocspCert)
	test.AssertNotError(t, err, "makeTemplate failed")

	test.Assert(t, !cert.IsCA, "IsCA is set")
	// Check KU is only KeyUsageDigitalSignature
	test.AssertEquals(t, cert.KeyUsage, x509.KeyUsageDigitalSignature)
	// Check there is a single EKU with id-kp-OCSPSigning
	test.AssertEquals(t, len(cert.ExtKeyUsage), 1)
	test.AssertEquals(t, cert.ExtKeyUsage[0], x509.ExtKeyUsageOCSPSigning)
	// Check ExtraExtensions contains a single id-pkix-ocsp-nocheck
	hasExt := false
	asnNULL := []byte{5, 0}
	for _, ext := range cert.ExtraExtensions {
		if ext.Id.Equal(oidOCSPNoCheck) {
			if hasExt {
				t.Error("template contains multiple id-pkix-ocsp-nocheck extensions")
			}
			hasExt = true
			if !bytes.Equal(ext.Value, asnNULL) {
				t.Errorf("id-pkix-ocsp-nocheck has unexpected content: want %x, got %x", asnNULL, ext.Value)
			}
		}
	}
	test.Assert(t, hasExt, "template doesn't contain id-pkix-ocsp-nocheck extensions")
}

func TestMakeTemplateCRL(t *testing.T) {
	s, ctx := pkcs11helpers.NewSessionWithMock()
	ctx.GenerateRandomFunc = realRand
	randReader := newRandReader(s)
	profile := &certProfile{
		SignatureAlgorithm: "SHA256WithRSA",
		CommonName:         "common name",
		Organization:       "organization",
		Country:            "country",
		OCSPURL:            "http://ocsp.example.com",
		CRLURL:             "http://crl.example.com",
		IssuerURL:          "http://issuer.example.com",
		NotAfter:           "2018-05-18 11:31:00",
		NotBefore:          "2018-05-18 11:31:00",
	}
	pubKey := samplePubkey()

	cert, err := makeTemplate(randReader, profile, pubKey, crlCert)
	test.AssertNotError(t, err, "makeTemplate failed")

	test.Assert(t, !cert.IsCA, "IsCA is set")
	test.AssertEquals(t, cert.KeyUsage, x509.KeyUsageCRLSign)
}

func TestVerifyProfile(t *testing.T) {
	for _, tc := range []struct {
		profile     certProfile
		certType    certType
		expectedErr string
	}{
		{
			profile:     certProfile{},
			certType:    intermediateCert,
			expectedErr: "not-before is required",
		},
		{
			profile: certProfile{
				NotBefore: "a",
			},
			certType:    intermediateCert,
			expectedErr: "not-after is required",
		},
		{
			profile: certProfile{
				NotBefore: "a",
				NotAfter:  "b",
			},
			certType:    intermediateCert,
			expectedErr: "signature-algorithm is required",
		},
		{
			profile: certProfile{
				NotBefore:          "a",
				NotAfter:           "b",
				SignatureAlgorithm: "c",
			},
			certType:    intermediateCert,
			expectedErr: "common-name is required",
		},
		{
			profile: certProfile{
				NotBefore:          "a",
				NotAfter:           "b",
				SignatureAlgorithm: "c",
				CommonName:         "d",
			},
			certType:    intermediateCert,
			expectedErr: "organization is required",
		},
		{
			profile: certProfile{
				NotBefore:          "a",
				NotAfter:           "b",
				SignatureAlgorithm: "c",
				CommonName:         "d",
				Organization:       "e",
			},
			certType:    intermediateCert,
			expectedErr: "country is required",
		},
		{
			profile: certProfile{
				NotBefore:          "a",
				NotAfter:           "b",
				SignatureAlgorithm: "c",
				CommonName:         "d",
				Organization:       "e",
				Country:            "f",
				OCSPURL:            "http://ocsp.example.com",
			},
			certType:    intermediateCert,
			expectedErr: "crl-url is required for intermediates",
		},
		{
			profile: certProfile{
				NotBefore:          "a",
				NotAfter:           "b",
				SignatureAlgorithm: "c",
				CommonName:         "d",
				Organization:       "e",
				Country:            "f",
				OCSPURL:            "http://ocsp.example.com",
				CRLURL:             "http://crl.example.com",
			},
			certType:    intermediateCert,
			expectedErr: "issuer-url is required for intermediates",
		},
		{
			profile: certProfile{
				NotBefore:          "a",
				NotAfter:           "b",
				SignatureAlgorithm: "c",
				CommonName:         "d",
				Organization:       "e",
				Country:            "f",
			},
			certType: rootCert,
		},
		{
			profile: certProfile{
				NotBefore:          "a",
				NotAfter:           "b",
				SignatureAlgorithm: "c",
				CommonName:         "d",
				Organization:       "e",
				Country:            "f",
				IssuerURL:          "http://issuer.example.com",
				KeyUsages:          []string{"j"},
			},
			certType:    ocspCert,
			expectedErr: "key-usages cannot be set for a delegated signer",
		},
		{
			profile: certProfile{
				NotBefore:          "a",
				NotAfter:           "b",
				SignatureAlgorithm: "c",
				CommonName:         "d",
				Organization:       "e",
				Country:            "f",
				IssuerURL:          "http://issuer.example.com",
				CRLURL:             "http://crl.example.com",
			},
			certType:    ocspCert,
			expectedErr: "crl-url cannot be set for a delegated signer",
		},
		{
			profile: certProfile{
				NotBefore:          "a",
				NotAfter:           "b",
				SignatureAlgorithm: "c",
				CommonName:         "d",
				Organization:       "e",
				Country:            "f",
				IssuerURL:          "http://issuer.example.com",
				OCSPURL:            "http://ocsp.example.com",
			},
			certType:    ocspCert,
			expectedErr: "ocsp-url cannot be set for a delegated signer",
		},
		{
			profile: certProfile{
				NotBefore:          "a",
				NotAfter:           "b",
				SignatureAlgorithm: "c",
				CommonName:         "d",
				Organization:       "e",
				Country:            "f",
				IssuerURL:          "http://issuer.example.com",
			},
			certType: ocspCert,
		},
		{
			profile: certProfile{
				NotBefore:          "a",
				NotAfter:           "b",
				SignatureAlgorithm: "c",
				CommonName:         "d",
				Organization:       "e",
				Country:            "f",
				IssuerURL:          "http://issuer.example.com",
				KeyUsages:          []string{"j"},
			},
			certType:    crlCert,
			expectedErr: "key-usages cannot be set for a delegated signer",
		},
		{
			profile: certProfile{
				NotBefore:          "a",
				NotAfter:           "b",
				SignatureAlgorithm: "c",
				CommonName:         "d",
				Organization:       "e",
				Country:            "f",
				IssuerURL:          "http://issuer.example.com",
				CRLURL:             "http://crl.example.com",
			},
			certType:    crlCert,
			expectedErr: "crl-url cannot be set for a delegated signer",
		},
		{
			profile: certProfile{
				NotBefore:          "a",
				NotAfter:           "b",
				SignatureAlgorithm: "c",
				CommonName:         "d",
				Organization:       "e",
				Country:            "f",
				IssuerURL:          "http://issuer.example.com",
				OCSPURL:            "http://ocsp.example.com",
			},
			certType:    crlCert,
			expectedErr: "ocsp-url cannot be set for a delegated signer",
		},
		{
			profile: certProfile{
				NotBefore:          "a",
				NotAfter:           "b",
				SignatureAlgorithm: "c",
				CommonName:         "d",
				Organization:       "e",
				Country:            "f",
				IssuerURL:          "http://issuer.example.com",
			},
			certType: crlCert,
		},
	} {
		err := tc.profile.verifyProfile(tc.certType)
		if err != nil {
			if tc.expectedErr != err.Error() {
				t.Fatalf("Expected %q, got %q", tc.expectedErr, err.Error())
			}
		} else if tc.expectedErr != "" {
			t.Fatalf("verifyProfile didn't fail, expected %q", tc.expectedErr)
		}
	}
}

func TestValidateURLsSuccess(t *testing.T) {
	profile := &certProfile{CRLURL: "http://example.com/"}
	err := profile.validateURLs()
	test.AssertNotError(t, err, "error validating valid URL")
}

func TestValidateURLsNoParse(t *testing.T) {
	profile := &certProfile{OCSPURL: "://"}
	if err := profile.validateURLs(); err == nil {
		t.Error("expected error, got none")
	}
}

func TestValidateURLsBadScheme(t *testing.T) {
	profile := &certProfile{IssuerURL: "https://issuer.example.com"}
	if err := profile.validateURLs(); err == nil {
		t.Error("expected error, got none")
	}
}
