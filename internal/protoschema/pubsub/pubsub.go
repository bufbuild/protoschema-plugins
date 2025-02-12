// Copyright 2024-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pubsub

import (
	"github.com/bufbuild/protoschema-plugins/internal/protoschema/normalize"
	"github.com/jhump/protoreflect/desc" //nolint:staticcheck
	"github.com/jhump/protoreflect/desc/protoprint"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	// FileExtension is the file extension for the PubSub schema files.
	FileExtension = "pubsub.proto"
)

// Generate generates a PubSub schema in the form of a single self-contained messaged normalized to
// proto2 for the given message descriptor.
func Generate(input protoreflect.MessageDescriptor) (string, error) {
	normalizer := normalize.NewNormalizer()
	rootMsg, err := normalizer.Normalize(input)
	if err != nil {
		return "", err
	}
	file := &descriptorpb.FileDescriptorProto{
		Name: proto.String(FileExtension),
		// TODO: If/when Pub/Bub schemas support editions, use Edition 2023 so
		//       there is no loss of fidelity for syntax-specific semantics
		//       (such as field presence).
		Syntax: proto.String("proto2"),
		MessageType: []*descriptorpb.DescriptorProto{
			rootMsg,
		},
	}
	fileDesc, err := desc.CreateFileDescriptor(file)
	if err != nil {
		return "", err
	}
	printer := &protoprint.Printer{
		Compact: true,
	}
	return printer.PrintProtoToString(fileDesc)
}
