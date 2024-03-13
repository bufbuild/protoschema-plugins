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

	"cloud.google.com/go/bigquery"
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
	if len(os.Args) != 5 {
		return fmt.Errorf("usage: %s [project] [dataset] [table] [*.TableSchema.json]", os.Args[0])
	}
	projectID := os.Args[1]
	datasetID := os.Args[2]
	tableID := os.Args[3]
	schemaPath := os.Args[4]

	fileInfo, err := os.Stat(schemaPath)
	if err != nil {
		return err
	} else if fileInfo.IsDir() {
		return fmt.Errorf("expected %s to be a file", fileInfo)
	}
	// Read all the contents of the file
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return err
	}

	schema, err := bigquery.SchemaFromJSON(data)
	if err != nil {
		return err
	}

	if err := createTable(projectID, datasetID, tableID, &bigquery.TableMetadata{
		Schema: schema,
	}); err != nil {
		return err
	}
	return nil
}

func createTable(projectID, datasetID, tableID string, tableMetadata *bigquery.TableMetadata) error {
	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return err
	}
	defer client.Close()
	tableRef := client.Dataset(datasetID).Table(tableID)
	if err := tableRef.Create(ctx, tableMetadata); err != nil {
		return err
	}
	return nil
}
