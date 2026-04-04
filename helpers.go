package main

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func newErrorResult(err error) *mcp.CallToolResult {
	result := &mcp.CallToolResult{}
	result.SetError(err)
	return result
}
