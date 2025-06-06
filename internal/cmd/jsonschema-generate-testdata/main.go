// Copyright 2023 Buf Technologies, Inc.
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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/bufbuild/protoschema-plugins/internal/gen/proto/bufext/cel/expr/conformance/proto3"
	"github.com/bufbuild/protoschema-plugins/internal/protoschema/golden"
	"github.com/bufbuild/protoschema-plugins/internal/protoschema/jsonschema"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func main() {
	if err := run(); err != nil {
		if errString := err.Error(); errString != "" {
			_, _ = fmt.Fprintln(os.Stderr, errString)
		}
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: %s [output dir]", os.Args[0])
	}
	outputDir := os.Args[1]
	// Make sure the directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	testDescs, err := golden.GetTestDescriptors("./internal/testdata")
	if err != nil {
		return err
	}
	protoNameGenerator := jsonschema.NewGenerator()
	protoNameBundleGenerator := jsonschema.NewGenerator(jsonschema.WithBundle())
	protoNameStrictGenerator := jsonschema.NewGenerator(jsonschema.WithStrict())
	protoNameStrictBundleGenerator := jsonschema.NewGenerator(jsonschema.WithStrict(), jsonschema.WithBundle())
	jsonNameGenerator := jsonschema.NewGenerator(jsonschema.WithJSONNames())
	jsonNameBundleGenerator := jsonschema.NewGenerator(jsonschema.WithJSONNames(), jsonschema.WithBundle())
	jsonNameStrictGenerator := jsonschema.NewGenerator(jsonschema.WithJSONNames(), jsonschema.WithStrict())
	jsonNameStrictBundleGenerator := jsonschema.NewGenerator(jsonschema.WithJSONNames(), jsonschema.WithStrict(), jsonschema.WithBundle())
	generators := []*jsonschema.Generator{
		protoNameGenerator,
		protoNameBundleGenerator,
		protoNameStrictGenerator,
		protoNameStrictBundleGenerator,
		jsonNameGenerator,
		jsonNameBundleGenerator,
		jsonNameStrictGenerator,
		jsonNameStrictBundleGenerator,
	}
	for _, testDesc := range testDescs {
		for _, generator := range generators {
			if err := generator.Add(testDesc); err != nil {
				return err
			}
		}
	}

	for _, generator := range generators {
		if err := writeJSONSchema(outputDir, generator.Generate()); err != nil {
			return err
		}
	}
	return nil
}

func writeJSONSchema(outputDir string, schema map[protoreflect.FullName]map[string]any) error {
	for _, jsonSchema := range schema {
		// Serialize the JSON
		data, err := json.MarshalIndent(jsonSchema, "", "  ")
		if err != nil {
			return err
		}
		identifier, ok := jsonSchema["$id"].(string)
		if !ok {
			return errors.New("expected $id to be a string")
		}
		if identifier == "" {
			return errors.New("expected $id to be non-empty")
		}
		filePath := filepath.Join(outputDir, identifier)
		if err := golden.GenerateGolden(filePath, string(data)+"\n"); err != nil {
			return err
		}
	}
	return nil
}
