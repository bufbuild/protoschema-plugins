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

package jsonschema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/protoschema-plugins/internal/protoschema/golden"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestJSONSchemaGolden(t *testing.T) {
	t.Parallel()
	dirPath := filepath.FromSlash("../../testdata/jsonschema")
	testDescs, err := golden.GetTestDescriptors("../../testdata")
	require.NoError(t, err)
	for _, testDesc := range testDescs {
		for _, jsonSchema := range Generate(testDesc) {
			// Serialize the JSON
			data, err := json.MarshalIndent(jsonSchema, "", "  ")
			require.NoError(t, err)

			identifier, ok := jsonSchema["$id"].(string)
			require.True(t, ok)
			require.NotEmpty(t, identifier)

			filePath := filepath.Join(dirPath, identifier)
			err = golden.CheckGolden(filePath, string(data)+"\n")
			require.NoError(t, err)
		}
	}
}

func TestTitle(t *testing.T) {
	t.Parallel()
	require.Equal(t, "Foo", generateTitle("Foo"))
	require.Equal(t, "Foo Bar", generateTitle("FooBar"))
	require.Equal(t, "foo Bar", generateTitle("fooBar"))
	require.Equal(t, "Foo Bar Baz", generateTitle("FooBarBaz"))
	require.Equal(t, "FOO Bar", generateTitle("FOOBar"))
	require.Equal(t, "U Int64 Value", generateTitle("UInt64Value"))
	require.Equal(t, "Uint64 Value", generateTitle("Uint64Value"))
	require.Equal(t, "FOO", generateTitle("FOO"))
}

func TestConstraints(t *testing.T) {
	t.Parallel()
	schemaPath := filepath.FromSlash("../../testdata/jsonschema/buf.protoschema.test.v1.ConstraintTests.schema.json")
	testPath := filepath.FromSlash("../../testdata/jsonschema-doc/test.ConstraintTests.yaml")
	expectedPath := filepath.FromSlash("../../testdata/jsonschema-doc/test.ConstraintTests.txt")
	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile(schemaPath)
	require.NoError(t, err)

	yamlData, err := os.ReadFile(testPath)
	require.NoError(t, err)
	expectedData, err := os.ReadFile(expectedPath)
	require.NoError(t, err)

	var jsonData map[string]interface{}
	err = yaml.Unmarshal(yamlData, &jsonData)
	require.NoError(t, err)

	err = schema.Validate(jsonData)
	require.Error(t, err)
	errStr := err.Error()
	// Remove the first line of the error message, which contains the path to the schema file.
	if pos := strings.Index(errStr, "\n"); pos != -1 {
		errStr = errStr[pos+1:]
	}
	expectedStr := string(expectedData)
	expectedStr = strings.TrimSpace(expectedStr)
	require.Equal(t, expectedStr, errStr, errStr)
}
