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
