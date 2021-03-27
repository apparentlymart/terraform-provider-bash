package bash

import (
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tftypes"
)

type bashScriptConfig struct {
	Source    string
	Variables map[string]tftypes.Value
}

var bashScriptType = tftypes.Object{
	AttributeTypes: map[string]tftypes.Type{
		"source":    tftypes.String,
		"variables": tftypes.DynamicPseudoType,
		"result":    tftypes.String,
	},
}

var mapOfString = tftypes.Map{
	AttributeType: tftypes.String,
}

var listOfString = tftypes.List{
	ElementType: tftypes.String,
}

func newBashScriptConfig(raw *tfprotov5.DynamicValue) (*bashScriptConfig, []*tfprotov5.Diagnostic) {
	ret := &bashScriptConfig{}
	var diags []*tfprotov5.Diagnostic

	lessRaw, err := raw.Unmarshal(bashScriptType)
	if err != nil {
		// This particular error shouldn't happen because Terraform ought to
		// have verified that the configuration matches our schema.
		diags = append(diags, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Invalid configuration",
			Detail:   fmt.Sprintf("The given configuration doesn't match the expected schema: %s.", err),
		})
		return ret, diags
	}

	var obj map[string]tftypes.Value
	err = lessRaw.As(&obj)
	if err != nil {
		// Similarly, this indicates a bug in Terraform's validation.
		diags = append(diags, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Invalid configuration",
			Detail:   fmt.Sprintf("The given configuration doesn't match the expected schema: %s.", err),
		})
		return ret, diags
	}

	// If we get down here then obj should be a three-element map with
	// elements matching the bashScriptType shape. Therefore we assume that
	// some second-level conversions should always succeed.
	err = obj["source"].As(&ret.Source)
	if err != nil {
		panic("source isn't a string")
	}

	// "variables" is typed as DynamicPseudoType, so Terraform will allow it
	// to be anything in principle. We need it to be an object type though,
	// because we'll be using the attribute names as variable names.
	err = obj["variables"].As(&ret.Variables)
	if err != nil {
		diags = append(diags, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Invalid variables",
			Detail:   "The \"variables\" argument must be an object with one attribute per variable you wish to declare for the Bash script.",
			Attribute: &tftypes.AttributePath{
				Steps: []tftypes.AttributePathStep{
					tftypes.AttributeName("variables"),
				},
			},
		})
	}

	for name, val := range ret.Variables {
		if len(name) == 0 {
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Invalid variable name",
				Detail:   "The empty string is not a valid Bash variable name.",
				Attribute: &tftypes.AttributePath{
					Steps: []tftypes.AttributePathStep{
						tftypes.AttributeName("variables"),
						tftypes.AttributeName(name),
					},
				},
			})
			continue
		}
		if !validVariableName(name) {
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Invalid variable name",
				Detail:   fmt.Sprintf("Cannot use %q as a Bash variable name.", name),
				Attribute: &tftypes.AttributePath{
					Steps: []tftypes.AttributePathStep{
						tftypes.AttributeName("variables"),
						tftypes.AttributeName(name),
					},
				},
			})
			continue
		}
		switch {
		case val.Is(tftypes.String): // okay
		case val.Is(tftypes.Number):
			var f big.Float
			if err := val.As(&f); err != nil {
				// Weird!
				diags = append(diags, &tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Summary:  "Invalid variable value",
					Detail:   fmt.Sprintf("Failed to decode %q as a number: %s.", name, err),
					Attribute: &tftypes.AttributePath{
						Steps: []tftypes.AttributePathStep{
							tftypes.AttributeName("variables"),
							tftypes.AttributeName(name),
						},
					},
				})
				continue
			} else {
				if !f.IsInt() {
					diags = append(diags, &tfprotov5.Diagnostic{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Invalid variable value",
						Detail:   fmt.Sprintf("Can't use %s as value of %q: Bash doesn't support floating-point numbers.", f.Text('f', -1), name),
						Attribute: &tftypes.AttributePath{
							Steps: []tftypes.AttributePathStep{
								tftypes.AttributeName("variables"),
								tftypes.AttributeName(name),
							},
						},
					})
				}
				continue
			}
		case val.Is(listOfString):
		case val.Is(mapOfString):
		default:
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Invalid variable value",
				Detail:   fmt.Sprintf("Invalid value for Bash variable %q: Bash only supports strings, whole numbers, lists of strings, and maps of strings.", name),
				Attribute: &tftypes.AttributePath{
					Steps: []tftypes.AttributePathStep{
						tftypes.AttributeName("variables"),
						tftypes.AttributeName(name),
					},
				},
			})
			continue
		}
	}

	return ret, diags
}

func (c *bashScriptConfig) ResultObject(result string) tftypes.Value {
	vty := variablesType(c.Variables)
	return tftypes.NewValue(bashScriptType, map[string]tftypes.Value{
		"source":    tftypes.NewValue(tftypes.String, c.Source),
		"variables": tftypes.NewValue(vty, c.Variables),
		"result":    tftypes.NewValue(tftypes.String, result),
	})
}

func (c *bashScriptConfig) ResultDynamicValue(result string) *tfprotov5.DynamicValue {
	v, err := tfprotov5.NewDynamicValue(bashScriptType, c.ResultObject(result))
	if err != nil {
		// We control all of the inputs here, so any error represents a bug.
		panic(fmt.Sprintf("failed to build dynamic value: %s", err))
	}
	return &v
}

// variablesType is a helper to work around the fact that tftypes doesn't,
// at least at the time of writing this, have a way to ask for the type
// of a value, and so instead we assume all variables will be one of the
// types we accept during newBashScriptConfig and thus re-calculate the
// effective object type for the given variables, so we can ultimately
// construct a valid DynamicValue to return in responses.
func variablesType(vars map[string]tftypes.Value) tftypes.Type {
	if len(vars) == 0 {
		return tftypes.Object{
			AttributeTypes: nil,
		}
	}
	atys := make(map[string]tftypes.Type, len(vars))
	for k, v := range vars {
		switch {
		case v.Is(tftypes.String):
			atys[k] = tftypes.String
		case v.Is(tftypes.Number):
			atys[k] = tftypes.Number
		case v.Is(listOfString):
			atys[k] = listOfString
		case v.Is(mapOfString):
			atys[k] = mapOfString
		default:
			// DynamicPseudoType isn't actually valid to use here but
			// we don't care because we shouldn't ever get here if there's
			// a variable with a type other than the ones handled above.
			atys[k] = tftypes.DynamicPseudoType
		}
	}
	return tftypes.Object{
		AttributeTypes: atys,
	}
}
