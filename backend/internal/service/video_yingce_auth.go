package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	base "github.com/LuckyKuang/sub2api-plus/internal/pkg/volcsign"
	yp "github.com/LuckyKuang/sub2api-plus/internal/pkg/yingceprotocol"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func yingceDefault(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
func applyYingceProtocolAuth(req *http.Request, account *Account, token string, auth yp.ManifestAuth) error {
	req.Header.Del("Authorization")
	req.Header.Del("X-Api-Key")
	req.Header.Del("X-Goog-Api-Key")
	switch auth.Type {
	case "bearer":
		req.Header.Set(yingceDefault(auth.Header, "Authorization"), yingceDefault(auth.Prefix, "Bearer ")+token)
	case "header":
		req.Header.Set(auth.Header, auth.Prefix+token)
	case "google-api-key":
		req.Header.Set(yingceDefault(auth.Header, "x-goog-api-key"), token)
	case "tc3":
		return signProtocolTC3(req, token, account.GetCredential("secret_key"), auth)
	case "volcengine-v4":
		secret := account.GetCredential("secret_key")
		if secret == "" {
			return fmt.Errorf("此视频协议需要在账号配置 Secret Key")
		}
		credentials := base.Credentials{AccessKeyID: token, SecretAccessKey: secret, Region: auth.Region, Service: auth.Service}
		signed := credentials.Sign(req)
		*req = *signed
	default:
		return fmt.Errorf("unknown video authentication type %s", auth.Type)
	}
	return nil
}
func signProtocolTC3(req *http.Request, secretID, secretKey string, auth yp.ManifestAuth) error {
	if strings.TrimSpace(secretID) == "" || strings.TrimSpace(secretKey) == "" {
		return errors.New("腾讯云 TC3 鉴权需要 SecretId 和 SecretKey")
	}
	serviceName := yingceDefault(strings.TrimSpace(auth.Service), "hunyuan")
	payload, err := protocolRequestPayload(req)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	timestamp := now.Unix()
	dateStamp := now.Format("2006-01-02")
	contentType := yingceDefault(req.Header.Get("Content-Type"), "application/json")
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-TC-Timestamp", strconv.FormatInt(timestamp, 10))
	if region := strings.TrimSpace(auth.Region); region != "" {
		req.Header.Set("X-TC-Region", region)
	}
	canonicalHeaders := "content-type:" + strings.ToLower(strings.TrimSpace(contentType)) + "\n" + "host:" + strings.ToLower(req.URL.Host) + "\n"
	signedHeaders := "content-type;host"
	canonicalRequest := strings.Join([]string{req.Method, yingceDefault(req.URL.EscapedPath(), "/"), req.URL.Query().Encode(), canonicalHeaders, signedHeaders, sha256Hex(payload)}, "\n")
	scope := dateStamp + "/" + serviceName + "/tc3_request"
	stringToSign := strings.Join([]string{"TC3-HMAC-SHA256", strconv.FormatInt(timestamp, 10), scope, sha256Hex([]byte(canonicalRequest))}, "\n")
	secretDate := protocolHMAC([]byte("TC3"+secretKey), dateStamp)
	secretService := protocolHMAC(secretDate, serviceName)
	secretSigning := protocolHMAC(secretService, "tc3_request")
	signature := hex.EncodeToString(protocolHMAC(secretSigning, stringToSign))
	req.Header.Set("Authorization", fmt.Sprintf("TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", secretID, scope, signedHeaders, signature))
	return nil
}

func protocolRequestPayload(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}
	var reader io.ReadCloser
	var err error
	if req.GetBody != nil {
		reader, err = req.GetBody()
	} else {
		reader = req.Body
	}
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(reader)
	if req.GetBody != nil {
		_ = reader.Close()
	} else {
		req.Body = io.NopCloser(bytes.NewReader(data))
	}
	return data, err
}

func protocolHMAC(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

func sha256Hex(value []byte) string {
	hash := sha256.Sum256(value)
	return hex.EncodeToString(hash[:])
}
