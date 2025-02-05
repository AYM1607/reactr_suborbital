package wasmtest

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rcap"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/reactr/rwasm"
)

func TestASEcho(t *testing.T) {
	r := rt.New()

	// test a WASM module that is loaded directly instead of through the bundle
	doWasm := r.Register("as-echo", rwasm.NewRunner("../testdata/as-echo/as-echo.wasm"))

	res, err := doWasm("from AssemblyScript!").Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	fmt.Println(string(res.([]byte)))

	if string(res.([]byte)) != "hello, from AssemblyScript!" {
		t.Error("as-echo failed, got:", string(res.([]byte)))
	}
}

func TestASFetch(t *testing.T) {
	r := rt.New()

	// test a WASM module that is loaded directly instead of through the bundle
	doWasm := r.Register("as-fetch", rwasm.NewRunner("../testdata/as-fetch/as-fetch.wasm"))

	res, err := doWasm("https://1password.com").Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	if string(res.([]byte)[:100]) != "<!doctype html><html lang=en data-language-url=/><head><meta charset=utf-8><meta name=viewport conte" {
		t.Error("as-fetch failed, got:", string(res.([]byte)[:100]))
	}
}

func TestASJSON(t *testing.T) {
	r := rt.New()

	// test a WASM module that is loaded directly instead of through the bundle
	doWasm := r.Register("as-json", rwasm.NewRunner("../testdata/as-json/as-json.wasm"))

	res, err := doWasm("").Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	if string(res.([]byte)) != `{"firstName":"Connor","lastName":"Hicks","age":26,"meta":{"country":"Canada","province":"Ontario","isAwesome":true},"tags":["hello","world"]}` {
		t.Error("as-json failed, got:", string(res.([]byte)))
	}
}

func TestASGraphql(t *testing.T) {
	// bail out if GitHub auth is not set up (i.e. in Travis)
	if _, ok := os.LookupEnv("GITHUB_TOKEN"); !ok {
		return
	}

	r := rt.New()

	caps := r.DefaultCaps()
	caps.Auth = rcap.DefaultAuthProvider(rcap.AuthConfig{
		Enabled: true,
		Headers: map[string]rcap.AuthHeader{
			"api.github.com": {
				HeaderType: "bearer",
				Value:      "env(GITHUB_TOKEN)",
			},
		},
	})

	// test a WASM module that is loaded directly instead of through the bundle
	r.RegisterWithCaps("as-graphql", rwasm.NewRunner("../testdata/as-graphql/as-graphql.wasm"), caps)

	res, err := r.Do(rt.NewJob("as-graphql", nil)).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	if string(res.([]byte)) != `{"data":{"repository":{"name":"reactr","nameWithOwner":"suborbital/reactr"}}}` {
		t.Error("as-graphql failed, got:", string(res.([]byte)))
	}
}

func TestASLargeData(t *testing.T) {
	r := rt.New()

	// test a WASM module that is loaded directly instead of through the bundle
	doWasm := r.Register("as-echo", rwasm.NewRunner("../testdata/as-echo/as-echo.wasm"))

	res, err := doWasm(largeInput).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	if string(res.([]byte)) != "hello, "+largeInput {
		t.Error("as-test failed, got:", string(res.([]byte)))
	}
}

func TestASRunnerWithRequest(t *testing.T) {
	r := rt.New()

	doWasm := r.Register("wasm", rwasm.NewRunner("../testdata/as-req/as-req.wasm"))

	body := testBody{
		Username: "cohix",
	}

	bodyJSON, _ := json.Marshal(body)

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/hello/world",
		ID:     uuid.New().String(),
		Body:   bodyJSON,
		State: map[string][]byte{
			"hello": []byte("what is up"),
		},
	}

	reqJSON, err := req.ToJSON()
	if err != nil {
		t.Error("failed to ToJSON", err)
	}

	res, err := doWasm(reqJSON).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	resp := res.(*request.CoordinatedResponse)

	if string(resp.Output) != "hello what is up" {
		t.Error(fmt.Errorf("expected 'hello, what is up', got %s", string(res.([]byte))))
	}
}
