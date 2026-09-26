package volcsign

import (
	"net/http"
	"net/url"
	"time"
)

const timeFormatV4 = "20060102T150405Z"

type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	Service         string
	Region          string
	SessionToken    string
}
type metadata struct {
	algorithm       string
	credentialScope string
	signedHeaders   string
	date            string
	region          string
	service         string
}
type RequestParam struct {
	IsSignUrl bool
	Body      []byte
	Method    string
	Date      time.Time
	Path      string
	Host      string
	QueryList url.Values
	Headers   http.Header
}
type SignRequest struct {
	XDate          string
	XNotSignBody   string
	XCredential    string
	XAlgorithm     string
	XSignedHeaders string
	XSignedQueries string
	XSignature     string
	XSecurityToken string

	Host           string
	ContentType    string
	XContentSha256 string
	Authorization  string
}
