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

package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	validate "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/protoschema-plugins/internal/gen/proto/buf/validate/conformance/harness"
	"github.com/kaptinlin/jsonschema"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
)

type Config struct {
	SchemaDir string
}

func main() {
	config := Config{
		SchemaDir: os.Getenv("JSONSCHEMA_DIR"),
	}

	if err := run(config); err != nil {
		if errString := err.Error(); errString != "" {
			_, _ = fmt.Fprintln(os.Stderr, errString)
		}
		os.Exit(1)
	}
}

func run(config Config) error {
	req := &harness.TestConformanceRequest{}
	if data, err := io.ReadAll(os.Stdin); err != nil {
		return err
	} else if err = proto.Unmarshal(data, req); err != nil {
		return err
	}

	resp, err := TestConformance(req, config)
	if err != nil {
		return err
	}

	data, err := proto.Marshal(resp)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(data)
	return err
}

func TestConformance(req *harness.TestConformanceRequest, config Config) (*harness.TestConformanceResponse, error) {
	files, err := protodesc.NewFiles(req.GetFdset())
	if err != nil {
		err = fmt.Errorf("failed to parse file descriptors: %w", err)
		return nil, err
	}
	registry := &protoregistry.Types{}
	files.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		for i := range file.Extensions().Len() {
			if err = registry.RegisterExtension(
				dynamicpb.NewExtensionType(file.Extensions().Get(i)),
			); err != nil {
				return false
			}
		}
		return err == nil
	})
	if err != nil {
		return nil, err
	}
	compiler := jsonschema.NewCompiler()
	if err != nil {
		err = fmt.Errorf("failed to initialize validator: %w", err)
		return nil, err
	}
	resp := &harness.TestConformanceResponse{Results: map[string]*harness.TestResult{}}
	for caseName, testCase := range req.GetCases() {
		resp.Results[caseName] = TestCase(compiler, files, testCase, config)
	}
	return resp, nil
}

func TestCase(compiler *jsonschema.Compiler, files *protoregistry.Files, testCase *anypb.Any, config Config) *harness.TestResult {
	urlParts := strings.Split(testCase.GetTypeUrl(), "/")
	fullName := protoreflect.FullName(urlParts[len(urlParts)-1])
	desc, err := files.FindDescriptorByName(fullName)
	if err != nil {
		return unexpectedErrorResult("unable to find descriptor: %v", err)
	}
	msgDesc, ok := desc.(protoreflect.MessageDescriptor)
	if !ok {
		return unexpectedErrorResult("expected message descriptor, got %T", desc)
	}

	dyn := dynamicpb.NewMessage(msgDesc)
	if err = anypb.UnmarshalTo(testCase, dyn, proto.UnmarshalOptions{}); err != nil {
		return unexpectedErrorResult("unable to unmarshal test case: %v", err)
	}

	schema, err := os.ReadFile(config.SchemaDir + "/" + string(fullName) + ".jsonschema.strict.bundle.json")
	if err != nil {
		return unexpectedErrorResult("failed to load JSON Schema: %v", err)
	}

	json, err := protojson.Marshal(dyn)
	if err != nil {
		return unexpectedErrorResult("failed to marshal test case to JSON: %v", err)
	}

	val, err := compiler.Compile(schema)
	if err != nil {
		return unexpectedErrorResult("failed to compile JSON Schema: %v", err)
	}

	result := val.Validate(json)
	if result.IsValid() {
		return &harness.TestResult{
			Result: &harness.TestResult_Success{
				Success: true,
			},
		}
	}

	violations := []*validate.Violation{}
	for _, err := range result.Errors {
		violations = append(
			violations,
			&validate.Violation{Message: &err.Message},
		)
	}

	return &harness.TestResult{
		Result: &harness.TestResult_ValidationError{
			ValidationError: &validate.Violations{
				Violations: violations,
			},
		},
	}
}

func unexpectedErrorResult(format string, args ...any) *harness.TestResult {
	return &harness.TestResult{
		Result: &harness.TestResult_UnexpectedError{
			UnexpectedError: fmt.Sprintf(format, args...),
		},
	}
}
