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

package bigquery

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/bufbuild/protoschema-plugins/internal/protoschema/golden"
	"github.com/stretchr/testify/require"
)

func TestBigQueryGolden(t *testing.T) {
	t.Parallel()
	dirPath := filepath.FromSlash("../../testdata/bigquery")
	for _, testDesc := range golden.GetTestDescriptors() {
		filePath := filepath.Join(dirPath, fmt.Sprintf("%s.%s", testDesc.FullName(), FileExtension))
		schema, _, err := Generate(testDesc)
		require.NoError(t, err)
		data, err := schema.ToJSONFields()
		require.NoError(t, err)
		if len(data) == 0 || string(data) == "null" {
			continue
		}
		err = golden.CheckGolden(filePath, string(data))
		require.NoError(t, err)
	}
}
