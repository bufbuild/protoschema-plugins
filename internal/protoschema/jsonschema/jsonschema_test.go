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

package jsonschema

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/bufbuild/protoschema-plugins/internal/protoschema/golden"
	"github.com/stretchr/testify/require"
)

func TestJSONSchemaGolden(t *testing.T) {
	t.Parallel()
	dirPath := filepath.FromSlash("../../testdata/jsonschema")
	testDescs, err := golden.GetTestDescriptors("../../testdata")
	require.NoError(t, err)
	for _, testDesc := range testDescs {
		entries, err := Generate(testDesc)
		require.NoError(t, err)
		for _, entry := range entries {
			// Serialize the JSON
			data, err := json.MarshalIndent(entry, "", "  ")
			require.NoError(t, err)

			identifier, ok := entry["$id"].(string)
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
