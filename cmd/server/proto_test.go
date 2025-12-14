package main

import (
	v1 "hanashite/api/v1"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestProtobuf(t *testing.T) {
	env, _ := anypb.New(&v1.ConnectRequest{})

	data, _ := proto.Marshal(env)
	t.Fatalf("Size: %v", data)

}
