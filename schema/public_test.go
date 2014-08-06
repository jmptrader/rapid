package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchrcom/testify/assert"

	"github.com/alecthomas/rapid"
	"github.com/alecthomas/rapid/schema"
)

type TestSchemaToPublicRequestType struct {
	KV map[string]string `json:"kv"`
}

func TestSchemaToJSON(t *testing.T) {
	s := rapid.Define("Test")
	s.Route("Index").Get("/{id}").Response(&TestSchemaToPublicRequestType{})
	public := schema.SchemaToPublic(s.Schema)
	b, _ := json.Marshal(public)
	assert.Equal(t, string(b), `{"routes":[{"request_type":null,"response_type":{"kind":"struct","name":"TestSchemaToPublicRequestType","fields":[{"kind":"map","name":"KV","key":{"kind":"string"},"value":{"kind":"string"},"annotation":"json:\"kv\""}]},"query_type":null,"path_type":null,"name":"Index","description":"","path":"/{id}","method":"GET","streaming_response":false,"success_status":0}],"name":"Test","description":""}`)
}