// Copyright 2024 Buf Technologies, Inc.
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

package golden

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// GetTestFileDescriptorSet returns the FileDescriptorSet descriptors that were generated from the
// ./internal/proto directory.
func GetTestFileDescriptorSet(testdataPath string) (*descriptorpb.FileDescriptorSet, error) {
	inputPath := filepath.Join(filepath.FromSlash(testdataPath), "codegenrequest", "input.json")
	input, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open input file descritpor set at %q: %w", inputPath, err)
	}
	fdset := &descriptorpb.FileDescriptorSet{}
	if err := (&protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(input, fdset); err != nil {
		return nil, fmt.Errorf("failed to parse file descriptor set at %q: %w", inputPath, err)
	}
	return fdset, nil
}

// GetTestFiles returns the protoregistry.Files for the test files defined in internal/proto.
func GetTestFiles(testdataPath string) (*protoregistry.Files, error) {
	fdset, err := GetTestFileDescriptorSet(testdataPath)
	if err != nil {
		return nil, err
	}
	files, err := protodesc.NewFiles(fdset)
	if err != nil {
		return nil, fmt.Errorf("failed to link file descriptor set at %q: %w", testdataPath, err)
	}
	return files, nil
}

// GetTestDescriptors returns the descriptors for specific test messages defined in internal/proto.
func GetTestDescriptors(testdataPath string) ([]protoreflect.MessageDescriptor, error) {
	files, err := GetTestFiles(testdataPath)
	if err != nil {
		return nil, err
	}
	types := dynamicpb.NewTypes(files)

	fqns := []protoreflect.FullName{
		"bufext.cel.expr.conformance.proto3.TestAllTypes",
		"bufext.cel.expr.conformance.proto3.NestedTestAllTypes",
		"buf.protoschema.test.v1.NestedReference",
		"buf.protoschema.test.v1.CustomOptions",
		"buf.protoschema.test.v1.IgnoreField",
	}

	msgs := make([]protoreflect.MessageDescriptor, len(fqns))
	for i, fqn := range fqns {
		mType, err := types.FindMessageByName(fqn)
		if err != nil {
			return nil, fmt.Errorf("failed to find message %q: %w", fqn, err)
		}
		msgs[i] = mType.Descriptor()
	}
	return msgs, nil
}

// CheckGolden checks the golden file exists and matches the given data.
func CheckGolden(filePath string, data string) error {
	if _, err := os.Stat(filePath); err != nil {
		return err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	actualBytes, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	actualText := string(actualBytes)
	if actualText != data {
		return fmt.Errorf("file %s does not match expected contents", filePath)
	}
	return nil
}

// GenerateGolden writes the given data to the golden file.
func GenerateGolden(filePath string, data string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	// Write the data to the file
	if _, err := file.WriteString(data); err != nil {
		return err
	}
	return nil
}
