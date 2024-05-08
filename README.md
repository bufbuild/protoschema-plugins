# protoschema-plugins

[![Build](https://github.com/bufbuild/protoschema-plugins/actions/workflows/ci.yaml/badge.svg?branch=main)][badges_ci]
[![Report Card](https://goreportcard.com/badge/github.com/bufbuild/protoschema-plugins)][badges_goreportcard]
[![GoDoc](https://pkg.go.dev/badge/github.com/bufbuild/protoschema-plugins.svg)][badges_godoc]
[![Slack](https://img.shields.io/badge/slack-buf-%23e01563)][badges_slack]

The protoschema-plugins repository contains a collection of Protobuf plugins that generate different
types of schema from protobuf files. This includes:

- [PubSub](#pubsub-protobuf-schema)
- [JSON Schema](#json-schema)

## PubSub Protobuf Schema

Generates a schema for a given protobuf file that can be used as a PubSub schema in the form of a
single self-contained messaged normalized to proto2.

Install the `protoc-gen-pubsub` plugin directly:

```sh
go install github.com/bufbuild/protoschema-plugins/cmd/protoc-gen-pubsub@latest
```

Or reference it as a [Remote Plugin](https://buf.build/docs/generate/remote-plugins) in `buf.gen.yaml`:

```yaml
version: v1
plugins:
  - plugin: buf.build/bufbuild/protoschema-pubsub
    out: ./gen
```

For examples see [testdata](/internal/testdata/pubsub/) which contains the generated schema for
test case definitions found in [proto](/internal/proto/).

## JSON Schema

Generates a [JSON Schema](https://json-schema.org/) for a given protobuf file. This implementation
uses the latest [JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12/release-notes).

Install the `protoc-gen-jsonschema` directly:

```sh
go install github.com/bufbuild/protoschema-plugins/cmd/protoc-gen-jsonschema@latest
```

Or reference it as a [Remote Plugin](https://buf.build/docs/generate/remote-plugins) in `buf.gen.yaml`:

```yaml
version: v1
plugins:
  - plugin: buf.build/bufbuild/protoschema-jsonschema
    out: ./gen
```

For examples see [testdata](/internal/testdata/jsonschema/) which contains the generated schema for
test case definitions found in [proto](/internal/proto/).

Here is a simple generated schema from the following protobuf:

```proto
message Product {
  message Location {
    float lat = 1;
    float long = 2;
  }

  int32 product_id = 1;
  string product_name = 2;
  float price = 3;
  repeated string tags = 4;
  Location location = 5;
}
```

Results in the following JSON Schema files:

<details>
<summary>Product.schema.json</summary>

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "product_id": {
      "type": "integer"
    },
    "product_name": {
      "type": "string"
    },
    "price": {
      "type": "number"
    },
    "tags": {
      "type": "array",
      "items": {
        "type": "string"
      }
    },
    "location": {
      "type": "object",
      "properties": {
        "lat": {
          "type": "number"
        },
        "long": {
          "type": "number"
        }
      },
      "required": ["lat", "long"]
    }
  },
  "required": ["product_id", "product_name", "price", "tags", "location"]
}
```

</details>

<details>
<summary>Product.Location.schema.json</summary>

```json
{
  "$id": "Product.Location.schema.json",
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "additionalProperties": false,
  "properties": {
    "lat": {
      "anyOf": [
        {
          "type": "number"
        },
        {
          "type": "string"
        },
        {
          "enum": ["NaN", "Infinity", "-Infinity"],
          "type": "string"
        }
      ]
    },
    "long": {
      "anyOf": [
        {
          "type": "number"
        },
        {
          "type": "string"
        },
        {
          "enum": ["NaN", "Infinity", "-Infinity"],
          "type": "string"
        }
      ]
    }
  },
  "type": "object"
}
```

</details>

### Ignoring Fields

To exclude fields in a message from being included in the JSON Schema, add a 
leading or trailing comment to the field containing `jsonschema:ignore`:

```proto
message Config {
  string host = 1;
  uint32 port = 2;
  bool debug_mode = 3; // jsonschema:ignore
}
```

## Community

For help and discussion around Protobuf, best practices, and more, join us
on [Slack][badges_slack].

## Status

This project is currently in **alpha**. The API should be considered unstable and likely to change.

## Legal

Offered under the [Apache 2 license][license].

[badges_ci]: https://github.com/bufbuild/protoschema-plugins/actions/workflows/ci.yaml
[badges_goreportcard]: https://goreportcard.com/report/github.com/bufbuild/protoschema-plugins
[badges_godoc]: https://pkg.go.dev/github.com/bufbuild/protoschema-plugins
[badges_slack]: https://join.slack.com/t/bufbuild/shared_invite/zt-f5k547ki-dW9LjSwEnl6qTzbyZtPojw
[license]: https://github.com/bufbuild/protoschema-plugins/blob/main/LICENSE.txt
