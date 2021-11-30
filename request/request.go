package request

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/directive/executable"
	"github.com/suborbital/vektor/vk"
)

const (
	atmoHeadlessStateHeader  = "X-Atmo-State"
	atmoHeadlessParamsHeader = "X-Atmo-Params"
	atmoRequestIDHeader      = "X-Atmo-RequestID"
)

// CoordinatedRequest represents a request whose fulfillment can be coordinated across multiple hosts
// and is serializable to facilitate interoperation with Wasm Runnables and transmissible over the wire
type CoordinatedRequest struct {
	Method      string                  `json:"method"`
	URL         string                  `json:"url"`
	ID          string                  `json:"request_id"`
	Body        []byte                  `json:"body"`
	Headers     map[string]string       `json:"headers"`
	RespHeaders map[string]string       `json:"resp_headers"`
	Params      map[string]string       `json:"params"`
	State       map[string][]byte       `json:"state"`
	Sequence    []executable.Executable `json:"sequence,omitempty"`

	bodyValues map[string]interface{} `json:"-"`
}

// FromVKRequest creates a CoordinatedRequest from an VK request handler
func FromVKRequest(r *http.Request, ctx *vk.Ctx) (*CoordinatedRequest, error) {
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, vk.E(http.StatusInternalServerError, "failed to read request body")
	}

	defer r.Body.Close()

	flatHeaders := map[string]string{}
	for k, v := range r.Header {
		//we lowercase the key to have case-insensitive lookup later
		flatHeaders[strings.ToLower(k)] = v[0]
	}

	flatParams := map[string]string{}
	for _, p := range ctx.Params {
		flatParams[p.Key] = p.Value
	}

	req := &CoordinatedRequest{
		Method:      r.Method,
		URL:         r.URL.RequestURI(),
		ID:          ctx.RequestID(),
		Body:        reqBody,
		Headers:     flatHeaders,
		RespHeaders: map[string]string{},
		Params:      flatParams,
		State:       map[string][]byte{},
	}

	return req, nil
}

func (c *CoordinatedRequest) UseHeadlessHeaders(r *http.Request, ctx *vk.Ctx) error {
	// fill in initial state from the state header
	if stateJSON := r.Header.Get(atmoHeadlessStateHeader); stateJSON != "" {
		state := map[string]string{}
		byteState := map[string][]byte{}

		if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
			return errors.Wrap(err, "failed to Unmarshal X-Atmo-State header")
		} else {
			// iterate over the state and convert each field to bytes
			for k, v := range state {
				byteState[k] = []byte(v)
			}
		}

		c.State = byteState
	}

	// fill in the URL params from the Params header
	if paramsJSON := r.Header.Get(atmoHeadlessParamsHeader); paramsJSON != "" {
		params := map[string]string{}

		if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
			return errors.Wrap(err, "failed to Unmarshal X-Atmo-Params header")
		} else {
			c.Params = params
		}
	}

	// add the request ID as a response header
	ctx.RespHeaders.Add(atmoRequestIDHeader, ctx.RequestID())

	return nil
}

// BodyField returns a field from the request body as a string
func (c *CoordinatedRequest) BodyField(key string) (string, error) {
	if c.bodyValues == nil {
		if len(c.Body) == 0 {
			return "", nil
		}

		vals := map[string]interface{}{}

		if err := json.Unmarshal(c.Body, &vals); err != nil {
			return "", errors.Wrap(err, "failed to Unmarshal request body")
		}

		// cache the parsed body
		c.bodyValues = vals
	}

	interfaceVal, ok := c.bodyValues[key]
	if !ok {
		return "", fmt.Errorf("body does not contain field %s", key)
	}

	stringVal, ok := interfaceVal.(string)
	if !ok {
		return "", fmt.Errorf("request body value %s is not a string", key)
	}

	return stringVal, nil
}

// SetBodyField sets a field in the JSON body to a new value
func (c *CoordinatedRequest) SetBodyField(key, val string) error {
	if c.bodyValues == nil {
		if len(c.Body) == 0 {
			return nil
		}

		vals := map[string]interface{}{}

		if err := json.Unmarshal(c.Body, &vals); err != nil {
			return errors.Wrap(err, "failed to Unmarshal request body")
		}

		// cache the parsed body
		c.bodyValues = vals
	}

	c.bodyValues[key] = val

	newJSON, err := json.Marshal(c.bodyValues)
	if err != nil {
		return errors.Wrap(err, "failed to Marshal new body")
	}

	c.Body = newJSON

	return nil
}

// FromJSON unmarshalls a CoordinatedRequest from JSON
func FromJSON(jsonBytes []byte) (*CoordinatedRequest, error) {
	req := CoordinatedRequest{}
	if err := json.Unmarshal(jsonBytes, &req); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal request")
	}

	if req.Method == "" || req.URL == "" || req.ID == "" {
		return nil, errors.New("JSON is not CoordinatedRequest, required fields are empty")
	}

	return &req, nil
}

// ToJSON returns a JSON representation of a CoordinatedRequest
func (c *CoordinatedRequest) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}
