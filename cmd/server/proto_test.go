package main

import (
	v1 "hanashite/api/v1"
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestProtobuf(t *testing.T) {
	original := &v1.ConnectRequest{
		ClientKey: []byte{1, 2, 3},
	}

	// Serialize to bytes
	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Deserialize from bytes
	var decoded v1.ConnectRequest
	if err := proto.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Check equality
	if !proto.Equal(original, &decoded) {
		t.Errorf("expected %+v, got %+v", original, &decoded)
	}
}
