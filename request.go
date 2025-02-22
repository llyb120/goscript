package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/google/uuid"
)

var list []*StoredRequest

type StoredRequest struct {
	ID        string
	Method    string
	URL       string
	Headers   map[string][]string
	Body      []byte
	Timestamp int64
}

func saveRequest(req *http.Request) error {
	// 保存请求到数据库
	stored := &StoredRequest{
		ID:        uuid.New().String(),
		Method:    req.Method,
		URL:       req.URL.String(),
		Headers:   req.Header,
		Timestamp: time.Now().Unix(),
	}

	// 读取并保存请求体
	if req.Body != nil {
		body, _ := ioutil.ReadAll(req.Body)
		stored.Body = body
		req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	}

	list = append(list, stored)
	return nil
}
