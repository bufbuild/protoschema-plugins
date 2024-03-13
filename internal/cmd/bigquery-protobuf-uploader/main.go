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

package main

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/bigquery/storage/managedwriter"
	testv1 "github.com/bufbuild/protoschema-plugins/internal/gen/proto/buf/protoschema/test/v1"
	"github.com/bufbuild/protoschema-plugins/internal/protoschema/bigquery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
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
	if len(os.Args) != 4 {
		return fmt.Errorf("usage: %s [project] [dataset] [table]", os.Args[0])
	}
	projectID := os.Args[1]
	datasetID := os.Args[2]
	tableID := os.Args[3]
	tableName := fmt.Sprintf("projects/%s/datasets/%s/tables/%s", projectID, datasetID, tableID)

	ctx := context.Background()
	client, err := managedwriter.NewClient(ctx, projectID)
	if err != nil {
		return err
	}

	// Define a couple of messages.
	msgs := []*testv1.BigQueryWellknownTypeTest{
		{
			JsonValue: &structpb.Value{
				Kind: &structpb.Value_NumberValue{
					NumberValue: 1.0,
				},
			},
		},
	}

	_, msgDesc, err := bigquery.Generate(msgs[0].ProtoReflect().Descriptor())
	if err != nil {
		return err
	}

	managedStream, err := client.NewManagedStream(ctx,
		managedwriter.WithDestinationTable(tableName),
		managedwriter.WithType(managedwriter.DefaultStream),
		managedwriter.WithSchemaDescriptor(msgDesc))
	if err != nil {
		return err
	}

	// Encode the messages into binary format.
	encoded := make([][]byte, len(msgs))
	for k, v := range msgs {
		b, err := proto.Marshal(v)
		if err != nil {
			return err
		}
		encoded[k] = b
	}

	// Send the rows to the service, and specify an offset for managing deduplication.
	result, err := managedStream.AppendRows(ctx, encoded)
	if err != nil {
		return err
	}

	// Block until the write is complete and return the result.
	resp, err := result.FullResponse(ctx)
	if err != nil {
		return err
	}
	fmt.Println(resp)

	defer client.Close()
	return nil
}
