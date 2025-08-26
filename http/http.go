// Copyright 2025 Rahmad Afandi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package http

import (
	"encoding/json"

	"github.com/valyala/fasthttp"
)

type Http struct {
	BaseURL string
	Headers map[string]string
	Client  *fasthttp.Client
}

var (
	Post    = fasthttp.MethodPost
	Get     = fasthttp.MethodGet
	Put     = fasthttp.MethodPut
	Patch   = fasthttp.MethodPatch
	Delete  = fasthttp.MethodDelete
	Options = fasthttp.MethodOptions
	Head    = fasthttp.MethodHead
	Connect = fasthttp.MethodConnect
	Trace   = fasthttp.MethodTrace
)

func New(baseURL string) *Http {
	return &Http{
		BaseURL: baseURL,
		Headers: make(map[string]string),
		Client:  &fasthttp.Client{},
	}
}

func (h *Http) SetHeader(key, value string) {
	h.Headers[key] = value
}

func (h *Http) Get(path string, response interface{}) error {
	return h.request(fasthttp.MethodGet, path, nil, response)
}

func (h *Http) Post(path string, body, response interface{}) error {
	return h.request(fasthttp.MethodPost, path, body, response)
}

func (h *Http) Put(path string, body, response interface{}) error {
	return h.request(fasthttp.MethodPut, path, body, response)
}

func (h *Http) Patch(path string, body, response interface{}) error {
	return h.request(fasthttp.MethodPatch, path, body, response)
}

func (h *Http) Delete(path string, response interface{}) error {
	return h.request(fasthttp.MethodDelete, path, nil, response)
}

func (h *Http) FireAndForget(method, path string, body interface{}) {
	go h.request(method, path, body, nil)
}

func (h *Http) request(method, path string, body, response interface{}) error {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(h.BaseURL + path)
	req.Header.SetMethod(method)

	for key, value := range h.Headers {
		req.Header.Set(key, value)
	}

	if body != nil {
		reqBody, err := json.Marshal(body)
		if err != nil {
			return err
		}
		req.SetBody(reqBody)
		req.Header.SetContentType("application/json")
	}

	if err := h.Client.Do(req, resp); err != nil {
		return err
	}

	if response != nil {
		if err := json.Unmarshal(resp.Body(), response); err != nil {
			return err
		}
	}

	return nil
}
