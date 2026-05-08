package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const defaultTokenTTL = 15 * time.Minute

type Module struct {
	tokenTTL     time.Duration
	privateKey   *rsa.PrivateKey
	publicKeyPEM string
	users        *UserStore
	configured   bool
}

type issueTokenRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Subject      string `json:"subject"`
}

type issueTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type publicKeyResponse struct {
	Algorithm    string `json:"algorithm"`
	KeyClaimName string `json:"key_claim_name"`
	RSAPublicKey string `json:"rsa_public_key"`
}

type errorResponse struct {
	Message string `json:"message"`
}

func NewModuleFromEnv() (*Module, error) {
	tokenTTL := defaultTokenTTL

	if rawTokenTTL := os.Getenv("AUTH_TOKEN_TTL"); rawTokenTTL != "" {
		parsedTokenTTL, err := time.ParseDuration(rawTokenTTL)
		if err != nil {
			return nil, fmt.Errorf("parse AUTH_TOKEN_TTL: %w", err)
		}

		tokenTTL = parsedTokenTTL
	}

	module := &Module{
		tokenTTL: tokenTTL,
		users:    NewUserStore(defaultUsers()),
	}

	privateKeyPEM := os.Getenv("AUTH_JWT_PRIVATE_KEY_PEM")
	if privateKeyPEM == "" {
		log.Printf("auth module is disabled until AUTH_JWT_PRIVATE_KEY_PEM is configured")
		return module, nil
	}

	privateKey, publicKeyPEM, err := parsePrivateKey(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse AUTH_JWT_PRIVATE_KEY_PEM: %w", err)
	}

	module.privateKey = privateKey
	module.publicKeyPEM = publicKeyPEM
	module.configured = true

	return module, nil
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /auth/public-key", m.handlePublicKey)
	mux.HandleFunc("POST /auth/token", m.handleIssueToken)
}

func (m *Module) handlePublicKey(w http.ResponseWriter, r *http.Request) {
	if !m.configured {
		writeError(w, http.StatusServiceUnavailable, "auth module is not configured")
		return
	}

	writeJSON(w, http.StatusOK, publicKeyResponse{
		Algorithm:    jwt.SigningMethodRS256.Alg(),
		KeyClaimName: "iss",
		RSAPublicKey: m.publicKeyPEM,
	})
}

func (m *Module) handleIssueToken(w http.ResponseWriter, r *http.Request) {
	log.Printf("auth token issuance requested")

	if !m.configured {
		writeError(w, http.StatusServiceUnavailable, "auth module is not configured")
		return
	}

	request, err := decodeIssueTokenRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, ok := m.users.FindByClientCredentials(request.ClientID, request.ClientSecret)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid client credentials")
		return
	}

	subject := request.Subject
	if subject == "" {
		subject = user.ID
	}

	now := time.Now().UTC()
	expiresAt := now.Add(m.tokenTTL)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": user.ClientID,
		"sub": subject,
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"exp": expiresAt.Unix(),
	})

	signedToken, err := token.SignedString(m.privateKey)
	if err != nil {
		log.Printf("unable to issue auth token: %v", err)
		writeError(w, http.StatusInternalServerError, "unable to issue auth token")
		return
	}

	writeJSON(w, http.StatusOK, issueTokenResponse{
		AccessToken: signedToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(m.tokenTTL.Seconds()),
	})
}

func decodeIssueTokenRequest(r *http.Request) (issueTokenRequest, error) {
	var request issueTokenRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		return issueTokenRequest{}, errors.New("invalid request body")
	}

	if request.ClientID == "" || request.ClientSecret == "" {
		return issueTokenRequest{}, errors.New("client_id and client_secret are required")
	}

	return request, nil
}

func parsePrivateKey(rawPEM string) (*rsa.PrivateKey, string, error) {
	normalizedPEM := strings.ReplaceAll(rawPEM, "\\n", "\n")
	block, _ := pem.Decode([]byte(normalizedPEM))
	if block == nil {
		return nil, "", errors.New("invalid PEM block")
	}

	privateKey, err := parseRSAPrivateKey(block.Bytes)
	if err != nil {
		return nil, "", err
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, "", fmt.Errorf("marshal public key: %w", err)
	}

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes})

	return privateKey, string(publicKeyPEM), nil
}

func parseRSAPrivateKey(privateKeyBytes []byte) (*rsa.PrivateKey, error) {
	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBytes)
	if err == nil {
		return privateKey, nil
	}

	parsedKey, pkcs8Err := x509.ParsePKCS8PrivateKey(privateKeyBytes)
	if pkcs8Err != nil {
		return nil, errors.New("private key must be a PKCS#1 or PKCS#8 RSA key")
	}

	rsaPrivateKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key must be an RSA key")
	}

	return rsaPrivateKey, nil
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, errorResponse{Message: message})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("response err: %v", err)
	}
}
