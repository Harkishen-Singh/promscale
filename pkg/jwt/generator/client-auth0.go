package generator

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dgrijalva/jwt-go/v4"
)

type Auth0_Response struct {
	Message string `json:"message"`
}

type Auth0_Jwks struct {
	Keys []Auth0_JSONWebKeys `json:"keys"`
}

type Auth0_JSONWebKeys struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

func publicKeyFromAuth0(domain string, token *jwt.Token) (*rsa.PublicKey, error) {
	cert, err := getPemCert(domain, token)
	if err != nil {
		panic(err.Error())
	}

	result, err := jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
	if err != nil {
		return nil, fmt.Errorf("parse RSA from public key from PEM: %w", err)
	}
	return result, nil
}

func getPemCert(domain string, token *jwt.Token) (string, error) {
	cert := ""
	resp, err := http.Get(fmt.Sprintf(`https://%s/.well-known/jwks.json`, domain))

	if err != nil {
		return cert, err
	}
	defer resp.Body.Close()

	var jwks = Auth0_Jwks{}
	err = json.NewDecoder(resp.Body).Decode(&jwks)

	if err != nil {
		return cert, err
	}

	for k, _ := range jwks.Keys {
		if token.Header["kid"] == jwks.Keys[k].Kid {
			cert = "-----BEGIN CERTIFICATE-----\n" + jwks.Keys[k].X5c[0] + "\n-----END CERTIFICATE-----"
		}
	}

	if cert == "" {
		err := fmt.Errorf("unable to find appropriate key")
		return cert, err
	}

	return cert, nil
}
