// SPDX-License-Identifier: AGPL-3.0-only

// Package contracttest checks answers against the public contract,
// api/openapi.yaml. The HTTP handler and MCP server tests use it to prove
// that what they really answer is what the contract says.
package contracttest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"

	"github.com/arnakoff/api/api"
)

// ErrUndocumented means the contract has no operation for the request's
// method and path.
var ErrUndocumented = errors.New("operation not in the contract")

// Contract is the public contract, ready to check exchanges against.
type Contract struct {
	doc *openapi3.T
}

// Load reads and validates the contract of package api. In its copy, every
// object schema of a response rejects undocumented properties: clients
// must ignore the fields they do not know, but the API must not send any.
// Request schemas stay open.
func Load() (*Contract, error) {
	doc, err := openapi3.NewLoader().LoadFromData(api.Spec())
	if err != nil {
		return nil, fmt.Errorf("load the contract: %w", err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("validate the contract: %w", err)
	}
	closeResponses(doc)
	return &Contract{doc: doc}, nil
}

// ErrRequestValid means CheckRefused was given a request the contract
// accepts: Check is the one to hold it.
var ErrRequestValid = errors.New("request follows the contract")

// Check returns an error unless req and res follow the contract: the
// operation is documented, the request's parameters and body are valid,
// and the response's status, headers, content type and body are the
// documented ones. It reads and closes both bodies.
func (c *Contract) Check(ctx context.Context, req *http.Request, res *http.Response) error {
	return c.hold(ctx, req, res, true)
}

// CheckRefused returns an error unless req is a documented operation the
// contract refuses, such as one without a required parameter, and res is a
// documented answer to it. It is how the problems answering invalid
// requests are held to the contract. It reads and closes both bodies.
func (c *Contract) CheckRefused(ctx context.Context, req *http.Request, res *http.Response) error {
	return c.hold(ctx, req, res, false)
}

// hold returns an error unless req is a documented operation, valid when
// valid is true and refused otherwise, and res a documented answer to it.
// It reads and closes both bodies.
func (c *Contract) hold(ctx context.Context, req *http.Request, res *http.Response, valid bool) error {
	defer func() { _ = res.Body.Close() }() // The body is read in full; a close error changes nothing.
	in, err := c.input(req)
	if err != nil {
		return err
	}
	err = openapi3filter.ValidateRequest(ctx, in)
	switch {
	case valid && err != nil:
		return fmt.Errorf("request %s %s: %w", req.Method, req.URL, err)
	case !valid && err == nil:
		return fmt.Errorf("%w: %s %s", ErrRequestValid, req.Method, req.URL)
	}
	return validateResponse(ctx, in, res)
}

// input returns the validation input of req, or ErrUndocumented.
func (c *Contract) input(req *http.Request) (*openapi3filter.RequestValidationInput, error) {
	route, err := c.route(req)
	if err != nil {
		return nil, err
	}
	return &openapi3filter.RequestValidationInput{
		Request: req,
		Route:   route,
		Options: &openapi3filter.Options{
			IncludeResponseStatus: true, MultiError: true, AuthenticationFunc: bearerPresent,
		},
	}, nil
}

// ErrNoCredentials means a request lacks the Authorization field its
// operation's security requirement asks for.
var ErrNoCredentials = errors.New("no credentials of the required scheme")

// bearerPresent holds a request to an HTTP bearer security requirement: it
// carries an Authorization field of the Bearer scheme with a token. Whether
// the key is valid is the API's to say, not the contract's.
func bearerPresent(_ context.Context, in *openapi3filter.AuthenticationInput) error {
	s := in.SecurityScheme
	if s.Type != "http" || !strings.EqualFold(s.Scheme, "bearer") {
		return fmt.Errorf("security scheme %s %s: not supported by the harness", s.Type, s.Scheme)
	}
	scheme, token, _ := strings.Cut(in.RequestValidationInput.Request.Header.Get("Authorization"), " ")
	if !strings.EqualFold(scheme, "Bearer") || token == "" {
		return ErrNoCredentials
	}
	return nil
}

// validateResponse returns an error unless res is a documented answer to
// the request of in.
func validateResponse(ctx context.Context, in *openapi3filter.RequestValidationInput, res *http.Response) error {
	out := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: in,
		Status:                 res.StatusCode,
		Header:                 res.Header,
		Body:                   res.Body,
		Options:                in.Options,
	}
	if err := openapi3filter.ValidateResponse(ctx, out); err != nil {
		req := in.Request
		return fmt.Errorf("response %d to %s %s: %w", res.StatusCode, req.Method, req.URL, err)
	}
	return nil
}

// ErrUnknownSchema means the contract has no component schema of the
// name CheckSchema is given.
var ErrUnknownSchema = errors.New("schema not in the contract")

// CheckSchema returns an error unless v, once written in JSON, follows the
// contract's component schema name, closed as a response's is. Answers
// that are not HTTP responses, such as the MCP server's, are held to the
// contract with it.
func (c *Contract) CheckSchema(name string, v any) error {
	ref, ok := c.doc.Components.Schemas[name]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownSchema, name)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("write the value as JSON: %w", err)
	}
	var value any
	if err := json.Unmarshal(b, &value); err != nil {
		return fmt.Errorf("read the value back: %w", err)
	}
	var opts []openapi3.SchemaValidationOption
	if c.doc.IsOpenAPI31OrLater() {
		opts = append(opts, openapi3.EnableJSONSchema2020())
	}
	if err := ref.Value.VisitJSON(value, opts...); err != nil {
		return fmt.Errorf("value against schema %s: %w", name, err)
	}
	return nil
}

// route returns the operation of the contract that req calls. The paths of
// the contract have no template: the request's path is looked up as is.
func (c *Contract) route(req *http.Request) (*routers.Route, error) {
	var op *openapi3.Operation
	item := c.doc.Paths.Value(req.URL.Path)
	if item != nil {
		op = item.GetOperation(req.Method)
	}
	if op == nil {
		return nil, fmt.Errorf("%w: %s %s", ErrUndocumented, req.Method, req.URL.Path)
	}
	return &routers.Route{
		Spec: c.doc, Path: req.URL.Path, PathItem: item, Method: req.Method, Operation: op,
	}, nil
}

// closeResponses closes the object schemas of every documented response.
func closeResponses(doc *openapi3.T) {
	seen := make(map[*openapi3.Schema]bool)
	for _, item := range doc.Paths.Map() {
		for _, op := range item.Operations() {
			for _, res := range op.Responses.Map() {
				for _, media := range res.Value.Content {
					closeObjects(media.Schema, seen)
				}
			}
		}
	}
}

// closeObjects forbids undocumented properties in ref's object schemas and
// in those they hold. Objects under allOf are left open: closing one would
// reject the properties of its siblings.
func closeObjects(ref *openapi3.SchemaRef, seen map[*openapi3.Schema]bool) {
	if ref == nil || ref.Value == nil || seen[ref.Value] {
		return
	}
	s := ref.Value
	seen[s] = true
	if len(s.Properties) > 0 && s.AdditionalProperties.Has == nil && s.AdditionalProperties.Schema == nil {
		closed := false
		s.AdditionalProperties.Has = &closed
	}
	for _, p := range s.Properties {
		closeObjects(p, seen)
	}
	closeObjects(s.Items, seen)
	closeObjects(s.AdditionalProperties.Schema, seen)
	for _, branch := range slices.Concat(s.OneOf, s.AnyOf) {
		closeObjects(branch, seen)
	}
}
