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

package pluginjsonschema

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/protoplugin"
	"github.com/bufbuild/protoschema-plugins/internal/protoschema/jsonschema"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Handle implements protoplugin.Handler and is the main entry point for the plugin.
func Handle(
	_ context.Context,
	_ protoplugin.PluginEnv,
	responseWriter protoplugin.ResponseWriter,
	request protoplugin.Request,
) error {
	fileDescriptors, err := request.FileDescriptorsToGenerate()
	if err != nil {
		return err
	}

	// Parse the parameters from the request.
	optionsWithJSONNames, err := parseOptions(request.Parameter())
	if err != nil {
		return err
	}
	optionsWithJSONNames = append(optionsWithJSONNames, jsonschema.WithJSONNames())
	// Also create options with protobuf names.
	options := optionsWithJSONNames[:len(optionsWithJSONNames)-1]

	protoNameGenerator := jsonschema.NewGenerator(options...)
	jsonNameGenerator := jsonschema.NewGenerator(optionsWithJSONNames...)

	// Generate the JSON schema for each message descriptor.
	for _, fileDescriptor := range fileDescriptors {
		for i := range fileDescriptor.Messages().Len() {
			messageDescriptor := fileDescriptor.Messages().Get(i)
			if err := protoNameGenerator.Add(messageDescriptor); err != nil {
				return err
			}
			if err := jsonNameGenerator.Add(messageDescriptor); err != nil {
				return err
			}
		}
	}

	if err := writeFiles(responseWriter, protoNameGenerator.Generate()); err != nil {
		return err
	}
	if err := writeFiles(responseWriter, jsonNameGenerator.Generate()); err != nil {
		return err
	}

	responseWriter.SetFeatureProto3Optional()
	responseWriter.SetFeatureSupportsEditions(descriptorpb.Edition_EDITION_2023, descriptorpb.Edition_EDITION_2023)
	return nil
}

func writeFiles(
	responseWriter protoplugin.ResponseWriter,
	schema map[protoreflect.FullName]map[string]any,
) error {
	for _, entry := range schema {
		data, err := json.MarshalIndent(entry, "", "  ")
		if err != nil {
			return err
		}
		identifier, ok := entry["$id"].(string)
		if !ok {
			return fmt.Errorf("expected unique id to be a string, got type %T", entry["$id"])
		}
		if identifier == "" {
			return errors.New("expected unique id to be a non-empty string")
		}
		responseWriter.AddFile(
			identifier,
			string(data)+"\n",
		)
	}
	return nil
}

func parseOptions(param string) ([]jsonschema.GeneratorOption, error) {
	var options []jsonschema.GeneratorOption
	if param == "" {
		return options, nil
	}
	// Params are in the form of "key1=value1,key2=value2"
	params := strings.Split(param, ",")
	for _, param := range params {
		// Split the param into key and value.
		pos := strings.Index(param, "=")
		if pos == -1 {
			return nil, fmt.Errorf("invalid parameter %q, expected key=value", param)
		}
		key := strings.TrimSpace(param[:pos])
		value := strings.TrimSpace(param[pos+1:])
		switch key {
		case "additional_properties":
			if value, err := parseBoolean(value); err != nil {
				return nil, err
			} else if value {
				options = append(options, jsonschema.WithAdditionalProperties())
			}
		case "strict":
			if value, err := parseBoolean(value); err != nil {
				return nil, err
			} else if value {
				options = append(options, jsonschema.WithStrict())
			}
		case "bundle":
			if value, err := parseBoolean(value); err != nil {
				return nil, err
			} else if value {
				options = append(options, jsonschema.WithBundle())
			}
		default:
			return nil, fmt.Errorf("unknown parameter %q", param)
		}
	}
	return options, nil
}

func parseBoolean(value string) (bool, error) {
	switch value {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value %q, expected true or false", value)
	}
}
