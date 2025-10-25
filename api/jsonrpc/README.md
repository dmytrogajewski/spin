# JSON-RPC 2.0 Specification

This directory contains JSON Schema definitions for JSON-RPC 2.0 as used in Spin's MCP implementation.

## Overview

JSON-RPC is a stateless, light-weight remote procedure call (RPC) protocol. Primarily this specification defines several data structures and the rules around their processing.

**Specification:** https://www.jsonrpc.org/specification

**Version:** 2.0

## Files

- **`spec.json`** - JSON Schema for JSON-RPC 2.0 messages
- **`README.md`** - This file

## Message Types

### Request

A request is sent from client to server to invoke a method.

**Structure:**
```json
{
  "jsonrpc": "2.0",
  "method": "methodName",
  "params": {},
  "id": 1
}
```

**Fields:**
- `jsonrpc` (required): Must be exactly "2.0"
- `method` (required): String containing the method name
- `params` (optional): Structured value (object or array) with parameters
- `id` (required): String, Number, or NULL identifier

**Example:**
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "read_file",
    "arguments": {
      "path": "/etc/hosts"
    }
  },
  "id": 1
}
```

### Notification

A notification is a request without an `id` field. The server will not send a response.

**Structure:**
```json
{
  "jsonrpc": "2.0",
  "method": "methodName",
  "params": {}
}
```

**Example:**
```json
{
  "jsonrpc": "2.0",
  "method": "notifications/initialized"
}
```

### Response (Success)

A successful response from server to client.

**Structure:**
```json
{
  "jsonrpc": "2.0",
  "result": {},
  "id": 1
}
```

**Fields:**
- `jsonrpc` (required): Must be exactly "2.0"
- `result` (required): The result of the method invocation
- `id` (required): Must match the request `id`

**Example:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "tools": [
      {
        "name": "read_file",
        "description": "Read a file"
      }
    ]
  },
  "id": 1
}
```

### Response (Error)

An error response when something goes wrong.

**Structure:**
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32601,
    "message": "Method not found",
    "data": {}
  },
  "id": 1
}
```

**Fields:**
- `jsonrpc` (required): Must be exactly "2.0"
- `error` (required): Error object
  - `code` (required): Integer error code
  - `message` (required): String description
  - `data` (optional): Additional error information
- `id` (required): Must match the request `id`, or NULL if request id couldn't be determined

**Example:**
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32602,
    "message": "Invalid params",
    "data": {
      "expected": "object",
      "received": "string"
    }
  },
  "id": 1
}
```

### Batch

Multiple requests/responses can be sent together as an array.

**Batch Request:**
```json
[
  {
    "jsonrpc": "2.0",
    "method": "tools/list",
    "id": 1
  },
  {
    "jsonrpc": "2.0",
    "method": "resources/list",
    "id": 2
  }
]
```

**Batch Response:**
```json
[
  {
    "jsonrpc": "2.0",
    "result": {"tools": []},
    "id": 1
  },
  {
    "jsonrpc": "2.0",
    "result": {"resources": []},
    "id": 2
  }
]
```

**Note:** Spin's MCP implementation currently does not support batch requests, but the schema is included for completeness.

## Error Codes

### Standard JSON-RPC Errors

| Code | Message | Meaning |
|------|---------|---------|
| -32700 | Parse error | Invalid JSON was received by the server |
| -32600 | Invalid Request | The JSON sent is not a valid Request object |
| -32601 | Method not found | The method does not exist / is not available |
| -32602 | Invalid params | Invalid method parameter(s) |
| -32603 | Internal error | Internal JSON-RPC error |

### Server Errors

Server-specific errors use codes from -32000 to -32099.

**Spin uses:**
- `-32000`: Generic server error
- `-32001`: Tool execution error
- `-32002`: Resource not found
- `-32003`: Timeout error

**Example:**
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32001,
    "message": "Tool execution failed",
    "data": {
      "tool": "read_file",
      "reason": "Permission denied"
    }
  },
  "id": 1
}
```

## Parameter Structures

JSON-RPC supports two parameter structures:

### By-position (Array)

Parameters are matched by array index:

```json
{
  "jsonrpc": "2.0",
  "method": "subtract",
  "params": [42, 23],
  "id": 1
}
```

### By-name (Object)

Parameters are matched by name:

```json
{
  "jsonrpc": "2.0",
  "method": "subtract",
  "params": {
    "minuend": 42,
    "subtrahend": 23
  },
  "id": 2
}
```

**Spin's MCP Implementation:** Uses by-name (object) parameters exclusively.

## Implementation in Spin

### Client Implementation

The JSON-RPC client is implemented in `internal/mcp/client/stdio.go`:

```go
// jsonrpcRequest represents a JSON-RPC 2.0 request.
type jsonrpcRequest struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      int             `json:"id"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params,omitempty"`
}

// jsonrpcResponse represents a JSON-RPC 2.0 response.
type jsonrpcResponse struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      int             `json:"id"`
    Result  json.RawMessage `json:"result,omitempty"`
    Error   *jsonrpcError   `json:"error,omitempty"`
}

// jsonrpcError represents a JSON-RPC 2.0 error.
type jsonrpcError struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}
```

### Sending Requests

```go
func (c *StdioClient) sendRequest(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
    // Serialize params
    paramsJSON, err := json.Marshal(params)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal params: %w", err)
    }

    // Create request
    c.mu.Lock()
    c.requestID++
    reqID := c.requestID
    c.mu.Unlock()

    req := jsonrpcRequest{
        JSONRPC: "2.0",
        ID:      int(reqID),
        Method:  method,
        Params:  paramsJSON,
    }

    // Send to server via stdin
    reqJSON, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }

    // ... (write to stdin, wait for response)
}
```

### Receiving Responses

```go
func (c *StdioClient) readResponses() {
    scanner := bufio.NewScanner(c.stdout)
    for scanner.Scan() {
        line := scanner.Bytes()
        
        var resp jsonrpcResponse
        if err := json.Unmarshal(line, &resp); err != nil {
            // Handle parse error
            continue
        }

        // Check for error
        if resp.Error != nil {
            // Handle JSON-RPC error
            continue
        }

        // Route response to waiting request
        c.routeResponse(resp.ID, resp.Result)
    }
}
```

## Validation

### Using JSON Schema

Validate messages using the provided JSON Schema:

```bash
# Install ajv-cli
npm install -g ajv-cli

# Validate a request
echo '{"jsonrpc":"2.0","method":"test","id":1}' | \
  ajv validate -s api/jsonrpc/spec.json -d -
```

### Using Go

Validate in Go code:

```go
import "encoding/json"

// Validate that jsonrpc field is "2.0"
type BaseMessage struct {
    JSONRPC string `json:"jsonrpc"`
}

func validateJSONRPC(data []byte) error {
    var base BaseMessage
    if err := json.Unmarshal(data, &base); err != nil {
        return fmt.Errorf("invalid json: %w", err)
    }
    
    if base.JSONRPC != "2.0" {
        return fmt.Errorf("invalid jsonrpc version: %s", base.JSONRPC)
    }
    
    return nil
}
```

## Testing

### Unit Tests

Test JSON-RPC message handling:

```bash
go test ./internal/mcp/client/... -v -run TestJSONRPC
```

### Example Test

```go
func TestJSONRPCRequest(t *testing.T) {
    req := jsonrpcRequest{
        JSONRPC: "2.0",
        ID:      1,
        Method:  "test",
        Params:  json.RawMessage(`{"key":"value"}`),
    }

    data, err := json.Marshal(req)
    if err != nil {
        t.Fatalf("Failed to marshal: %v", err)
    }

    var decoded jsonrpcRequest
    if err := json.Unmarshal(data, &decoded); err != nil {
        t.Fatalf("Failed to unmarshal: %v", err)
    }

    if decoded.JSONRPC != "2.0" {
        t.Errorf("Expected jsonrpc=2.0, got %s", decoded.JSONRPC)
    }
}
```

## Best Practices

### 1. Always Check JSONRPC Version

```go
if resp.JSONRPC != "2.0" {
    return fmt.Errorf("unsupported JSON-RPC version: %s", resp.JSONRPC)
}
```

### 2. Match Response IDs

```go
if resp.ID != expectedID {
    return fmt.Errorf("id mismatch: expected %v, got %v", expectedID, resp.ID)
}
```

### 3. Handle Both Result and Error

```go
if resp.Error != nil {
    return nil, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
}

if resp.Result == nil {
    return nil, fmt.Errorf("no result in response")
}
```

### 4. Use Structured Error Data

```go
type ErrorData struct {
    Detail  string `json:"detail"`
    Field   string `json:"field,omitempty"`
}

func createError(code int, message string, data ErrorData) *jsonrpcError {
    return &jsonrpcError{
        Code:    code,
        Message: message,
        Data:    data,
    }
}
```

### 5. Validate Method Names

```go
func isValidMethod(method string) bool {
    // Method should match: ^[a-zA-Z0-9_/-]+$
    matched, _ := regexp.MatchString(`^[a-zA-Z0-9_/-]+$`, method)
    return matched
}
```

## Debugging

### Enable Request/Response Logging

```go
func (c *StdioClient) sendRequest(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
    // ... create request

    if c.config.Debug {
        log.Printf("→ Request: %s", string(reqJSON))
    }

    // ... send request
}

func (c *StdioClient) readResponses() {
    // ...
    if c.config.Debug {
        log.Printf("← Response: %s", string(line))
    }
    // ...
}
```

### Common Issues

**Issue: Parse Error (-32700)**
- Check that each message is a single line ending with `\n`
- Verify JSON is valid using `jq` or online validators

**Issue: Invalid Request (-32600)**
- Ensure `jsonrpc` field is exactly "2.0"
- Ensure `method` field is present and is a string

**Issue: Method Not Found (-32601)**
- Check method name spelling
- Verify server supports the method (use `initialize` to check capabilities)

**Issue: Invalid Params (-32602)**
- Validate parameter types match expected schema
- Check for missing required parameters

## Further Reading

- **JSON-RPC 2.0 Specification:** https://www.jsonrpc.org/specification
- **JSON Schema:** https://json-schema.org/
- **MCP Protocol:** See `api/mcp/README.md`

## Contributing

When modifying JSON-RPC handling:

1. Update type definitions in `internal/mcp/client/stdio.go`
2. Update this README with examples
3. Update `spec.json` if adding custom error codes
4. Add tests for new message types
5. Document any deviations from the standard

## License

See the main project LICENSE file.
