package plugin

import (
	hcl "github.com/hashicorp/hcl/v2"
	tfplugin "github.com/terraform-linters/tflint-plugin-sdk/tflint"
	"github.com/terraform-linters/tflint/tflint"
	"github.com/zclconf/go-cty/cty"
)

// Server is a RPC server for responding to requests from plugins
type Server struct {
	runner  *tflint.Runner
	sources map[string][]byte
}

// NewServer initializes a RPC server for plugins
func NewServer(runner *tflint.Runner, sources map[string][]byte) *Server {
	return &Server{runner: runner, sources: sources}
}

// Attributes returns corresponding hcl.Attributes
func (s *Server) Attributes(req *tfplugin.AttributesRequest, resp *tfplugin.AttributesResponse) error {
	ret := []*tfplugin.Attribute{}
	err := s.runner.WalkResourceAttributes(req.Resource, req.AttributeName, func(attr *hcl.Attribute) error {
		ret = append(ret, &tfplugin.Attribute{
			Name:      attr.Name,
			Expr:      attr.Expr.Range().SliceBytes(s.sources[attr.Expr.Range().Filename]),
			ExprRange: attr.Expr.Range(),
			Range:     attr.Range,
			NameRange: attr.NameRange,
		})
		return nil
	})
	*resp = tfplugin.AttributesResponse{Attributes: ret, Err: err}
	return nil
}

// EvalExpr returns a value of the evaluated expression
func (s *Server) EvalExpr(req *tfplugin.EvalExprRequest, resp *tfplugin.EvalExprResponse) error {
	expr, diags := tflint.ParseExpression(req.Expr, req.ExprRange.Filename, req.ExprRange.Start)
	if diags.HasErrors() {
		return diags
	}

	val, err := s.runner.EvalExpr(expr, req.Ret, cty.Type{})
	if err != nil {
		if appErr, ok := err.(*tflint.Error); ok {
			err = tfplugin.Error(*appErr)
		}
	}
	*resp = tfplugin.EvalExprResponse{Val: val, Err: err}
	return nil
}

// EmitIssue reflects a issue to the Runner
func (s *Server) EmitIssue(req *tfplugin.EmitIssueRequest, resp *interface{}) error {
	expr, diags := tflint.ParseExpression(req.Expr, req.ExprRange.Filename, req.ExprRange.Start)
	if diags.HasErrors() {
		return diags
	}

	s.runner.WithExpressionContext(expr, func() error {
		s.runner.EmitIssue(req.Rule, req.Message, req.Location)
		return nil
	})
	return nil
}
