// SPDX-License-Identifier: AGPL-3.0-only

package mcp

import (
	"context"
	"errors"

	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/arnakoff/api/internal/adapter/wire"
	"github.com/arnakoff/api/internal/domain"
)

// checkArgs are the arguments of the check tool, the q and type
// parameters of GET /v1/check.
type checkArgs struct {
	Input string `json:"input"`
	Type  string `json:"type,omitempty"`
}

// checkTool describes the check tool. Its input schema names the kinds
// type accepts, the contract's InputType, and refuses any other argument.
func checkTool() *sdk.Tool {
	names := domain.KindNames()
	kinds := make([]any, 0, len(names))
	for _, name := range names {
		kinds = append(kinds, name)
	}
	in := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"input": {Type: "string", Description: "L'entrée telle que l'utilisateur l'a collée : un domaine, " +
				"une URL, une adresse e-mail, un numéro de téléphone, un IBAN, un SIREN ou un SIRET."},
			"type": {Type: "string", Enum: kinds, Description: "Force le type de l'entrée au lieu de le " +
				"détecter. Une URL est vérifiée comme son domaine (domain)."},
		},
		Required:             []string{"input"},
		AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
	}
	return &sdk.Tool{
		Name:  "check",
		Title: "Vérifier une entrée",
		Description: "Détecte le type de l'entrée (sauf si type est donné), la normalise, puis consulte " +
			"les sources qui la couvrent. Une source qui ne répond pas est marquée indisponible dans " +
			"sources : la vérification n'échoue pas pour autant.",
		InputSchema: in,
		Annotations: &sdk.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: new(true)},
	}
}

// check answers the check tool: the report of the input, of the kind type
// names when it is given, detected otherwise. An input of no kind, or not
// of the kind forced, is a tool error with the API's wording.
func check(c Checker) sdk.ToolHandlerFor[checkArgs, wire.Report] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, args checkArgs) (*sdk.CallToolResult, wire.Report, error) {
		kind := domain.KindAuto
		if args.Type != "" {
			// The input schema lists the kinds: an unknown one never gets here.
			k, err := domain.ParseKind(args.Type)
			if err != nil {
				return nil, wire.Report{}, err
			}
			kind = k
		}
		in, err := domain.ParseInput(args.Input, kind)
		if err != nil {
			// The reason never quotes the input, which may be personal data.
			return nil, wire.Report{}, errors.New(wire.InvalidInput(kind))
		}
		return nil, wire.ReportOf(in.Kind(), c.Check(ctx, in)), nil
	}
}
