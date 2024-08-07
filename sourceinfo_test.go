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

package protoschema

import (
	"embed"
	"testing"

	_ "github.com/bufbuild/protoschema-plugins/internal/gen/proto/buf/protoschema/test/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed internal/testdata/sourceinfo/**
var sourceInfoTestData embed.FS

func TestEmbeddedSourceInfo(t *testing.T) {
	t.Parallel()
	err := RegisterEmbeddedSourceInfo(sourceInfoTestData, "internal/testdata/sourceinfo")
	require.NoError(t, err)

	msgType, err := SourceInfoGlobalTypes.FindMessageByName(
		"buf.protoschema.test.v1.NestedReference",
	)
	require.NoError(t, err)
	parentFile := msgType.Descriptor().ParentFile()
	locs := parentFile.SourceLocations().ByDescriptor(msgType.Descriptor())
	assert.Equal(t, " A message comment.\n", locs.LeadingComments)
	fieldDesc := msgType.Descriptor().Fields().ByName("nested_message")
	require.NotNil(t, fieldDesc)
	locs = parentFile.SourceLocations().ByDescriptor(fieldDesc)
	assert.Equal(t, " A field comment.\n", locs.LeadingComments)
}
