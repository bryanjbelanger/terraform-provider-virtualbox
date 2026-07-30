package resources

import (
	"context"
	"fmt"
	"net"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// cidrValidator validates that a string attribute is a well-formed CIDR block.
type cidrValidator struct{}

func (v cidrValidator) Description(ctx context.Context) string {
	return "value must be a valid CIDR block (e.g. \"192.168.56.0/24\")"
}

func (v cidrValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v cidrValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if value == "" {
		return
	}
	if _, _, err := net.ParseCIDR(value); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid CIDR",
			fmt.Sprintf("%q is not a valid CIDR block: %s", value, err),
		)
	}
}

// validCIDR returns a validator that checks a string is a valid CIDR block.
func validCIDR() validator.String {
	return cidrValidator{}
}

// ipValidator validates that a string attribute is a well-formed IP address.
type ipValidator struct{}

func (v ipValidator) Description(ctx context.Context) string {
	return "value must be a valid IP address (e.g. \"192.168.56.100\")"
}

func (v ipValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v ipValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if value == "" {
		return
	}
	if net.ParseIP(value) == nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid IP Address",
			fmt.Sprintf("%q is not a valid IP address", value),
		)
	}
}

// validIP returns a validator that checks a string is a valid IP address.
func validIP() validator.String {
	return ipValidator{}
}
