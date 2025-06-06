package main

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/sdk"
	sdktest "github.com/open-policy-agent/opa/v1/sdk/test"
	"github.com/open-policy-agent/opa/v1/storage/inmem"
	"github.com/open-policy-agent/opa/v1/util"
)

func main() {
	//withServer()
	regoOnly()
}

func regoOnly() {
	// not sure why this gives a linter warning "style/messy-rule"
	module := `
		package example.authz

		default can_manage := false
		default can_view := false

		can_manage if {
			"can-manage" in data.claims[input.user][input.company]
		}

		can_view if {
			can_manage
		}

		can_view if {
			"can-view" in data.claims[input.user][input.company]
		}
		`

	data := `{
		"claims": {
			"alice": {
				"company-1": ["can-manage"],
				"company-2": ["can-view"]
			},
			"bob": {
				"company-2": ["can-manage"]
			}
		}
    }`

	var dataJson map[string]any

	err := util.UnmarshalJSON([]byte(data), &dataJson)
	if err != nil {
		log.Fatal("unmarshal data: " + err.Error())
	}

	store := inmem.NewFromObject(dataJson)

	ctx := context.Background()

	query, err := rego.New(
		rego.Module("example.rego", module),
		rego.Query("x = data.example.authz.can_view"),
		rego.Store(store),
	).PrepareForEval(ctx)

	if err != nil {
		log.Fatal("new rego: " + err.Error())
	}

	input := `{
		"user": "alice",
		"company": "company-1"
	}
	`
	var inputJson map[string]any
	err = util.UnmarshalJSON([]byte(input), &inputJson)
	if err != nil {
		log.Fatal("unmarshal data: " + err.Error())
	}
	results, err := query.Eval(ctx,
		rego.EvalInput(inputJson),
	)
	if err != nil {
		log.Fatal("eval: " + err.Error())
	}

	log.Print("result: ", results[0].Bindings["x"].(bool))
}

func withServer() {
	ctx := context.Background()

	// create a mock HTTP bundle server
	server, err := sdktest.NewServer(sdktest.MockBundle("/bundles/bundle.tar.gz", map[string]string{
		"example.rego": `
				package authz

				default allow := false

				allow if input.open == "sesame"
			`,
	}))
	if err != nil {
		log.Fatal("new server: " + err.Error())
	}

	defer server.Stop()

	// provide the OPA configuration which specifies
	// fetching policy bundles from the mock server
	// and logging decisions locally to the console
	config := []byte(fmt.Sprintf(`{
		"services": {
			"test": {
				"url": %q
			}
		},
		"bundles": {
			"test": {
				"resource": "/bundles/bundle.tar.gz"
			}
		},
		"decision_logs": {
			"console": true
		}
	}`, server.URL()))

	// create an instance of the OPA object
	opa, err := sdk.New(ctx, sdk.Options{
		ID:     "opa-test-1",
		Config: bytes.NewReader(config),
	})
	if err != nil {
		log.Fatal("new sdk: " + err.Error())
	}

	defer opa.Stop(ctx)

	// get the named policy decision for the specified input
	if result, err := opa.Decision(ctx, sdk.DecisionOptions{Path: "/authz/allow", Input: map[string]interface{}{"open": "sesame"}}); err != nil {
		log.Fatal("decistion: " + err.Error())
	} else if decision, ok := result.Result.(bool); !ok || !decision {
		log.Fatal("decision result is not a bool")
	}
}
