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
	"fmt"
	"path/filepath"
	"testing"

	"github.com/bufbuild/protoschema-plugins/internal/protoschema/golden"
	"github.com/stretchr/testify/require"
)

func TestPubSubGolden(t *testing.T) {
	t.Parallel()
	dirPath := filepath.FromSlash("../../testdata/pubsub")
	testDescs, err := golden.GetTestDescriptors("../../testdata")
	require.NoError(t, err)
	for _, testDesc := range testDescs {
		filePath := filepath.Join(dirPath, string(testDesc.FullName()))
		data, err := Generate(testDesc)
		require.NoError(t, err)
		err = golden.CheckGolden(fmt.Sprintf("%s.%s", filePath, FileExtension), data)
		require.NoError(t, err)
	}
}
