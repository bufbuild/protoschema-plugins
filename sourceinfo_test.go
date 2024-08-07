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
	// t.Parallel()
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
