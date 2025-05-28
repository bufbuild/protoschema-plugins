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
// A product.
//
// A product is a good or service that is offered for sale.
message Product {
  // A point on the earth's surface.
  message Location {
    double lat = 1 [
      (buf.validate.field).double.finite = true,
      (buf.validate.field).double.gte = -90,
      (buf.validate.field).double.lte = 90
    ];
    double long = 2 [
      (buf.validate.field).double.finite = true,
      (buf.validate.field).double.gte = -180,
      (buf.validate.field).double.lte = 180
    ];
  }

  // The unique identifier for the product.
  int32 product_id = 1 [(buf.validate.field).required = true];
  // The name of the product.
  string product_name = 2 [(buf.validate.field).required = true];
  // The price of the product.
  float price = 3 [
    (buf.validate.field).float.finite = true,
    (buf.validate.field).float.gte = 0
  ];
  // The tags associated with the product.
  repeated string tags = 4;
  // The location of the product.
  Location location = 5 [(buf.validate.field).required = true];
}

```

By default, results in the following JSON Schema files:

- `*.schema.json` files are generated with protobuf field names (e.g. `product_id`, `product_name`)
- `*.schema.bundle.json` files include all dependencies in a single file with protobuf field names.
- `*.schema.strict.json` files are generated with protobuf field names, but do not allow aliases, string numbers, or any other non-normalized representation.
- `*.schema.strict.bundle.json` files include the strict schema with all dependencies in a single file with protobuf field names.
- `*.jsonschema.json` files are generated with JSON field names (e.g. `productId`, `productName`)
  other non-normalized representation.
- `*.jsonschema.bundle.json` files include all dependencies in a single file with the JSON field names.
- `*.jsonschema.strict.json` files are generated with JSON field names, but do not allow aliases, string numbers, or any other non-normalized representation.
- `*.jsonschema.strict.bundle.json` files include the strict JSON schema with all dependencies in a single file with JSON field names.

For example, the above protobuf generates the following `*.schema.json` files:

<details>
<summary>Product.schema.json</summary>

```json
{
  "$id": "Product.schema.json",
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "additionalProperties": false,
  "title": "A product.",
  "description": "A product is a good or service that is offered for sale.",
  "type": "object",
  "properties": {
    "product_id": {
      "description": "The unique identifier for the product.",
      "maximum": 2147483647,
      "minimum": -2147483648,
      "type": "integer"
    },
    "product_name": {
      "description": "The name of the product.",
      "type": "string"
    },
    "price": {
      "anyOf": [
        {
          "maximum": 3.4028234663852886e38,
          "minimum": 0,
          "type": "number"
        },
        {
          "pattern": "^-?[0-9]+(\\.[0-9]+)?([eE][+-]?[0-9]+)?$",
          "type": "string"
        }
      ],
      "default": 0,
      "description": "The price of the product."
    },
    "tags": {
      "description": "The tags associated with the product.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "location": {
      "$ref": "Product.Location.schema.json",
      "description": "The location of the product."
    }
  },
  "required": ["product_id", "product_name", "location"],
  "patternProperties": {
    "^(productId)$": {
      "description": "The unique identifier for the product.",
      "maximum": 2147483647,
      "minimum": -2147483648,
      "type": "integer"
    },
    "^(productName)$": {
      "description": "The name of the product.",
      "type": "string"
    }
  }
}
```

</details>

<details>
<summary>Product.Location.schema.json</summary>

```json
{
  "$id": "Location.schema.json",
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "additionalProperties": false,
  "title": "Location",
  "description": "A point on the earth's surface.",
  "type": "object",
  "properties": {
    "lat": {
      "anyOf": [
        {
          "maximum": 90,
          "minimum": -90,
          "type": "number"
        },
        {
          "pattern": "^-?[0-9]+(\\.[0-9]+)?([eE][+-]?[0-9]+)?$",
          "type": "string"
        }
      ],
      "default": 0
    },
    "long": {
      "anyOf": [
        {
          "maximum": 180,
          "minimum": -180,
          "type": "number"
        },
        {
          "pattern": "^-?[0-9]+(\\.[0-9]+)?([eE][+-]?[0-9]+)?$",
          "type": "string"
        }
      ],
      "default": 0
    }
  }
}
```

</details>

Or the following `*.jsonschema.strict.bundle.json` file:

<details>
<summary>Product.jsonschema.strict.bundle.json</summary>

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "buf.protoschema.test.v1.Product.jsonschema.strict.bundle.json",
  "$ref": "#/$defs/buf.protoschema.test.v1.Product.jsonschema.strict.json",
  "$defs": {
    "buf.protoschema.test.v1.Product.jsonschema.strict.json": {
      "$schema": "https://json-schema.org/draft/2020-12/schema",
      "title": "A product.",
      "description": "A product is a good or service that is offered for sale.",
      "type": "object",
      "properties": {
        "productId": {
          "description": "The unique identifier for the product.",
          "maximum": 2147483647,
          "minimum": -2147483648,
          "type": "integer"
        },
        "productName": {
          "description": "The name of the product.",
          "type": "string"
        },
        "price": {
          "description": "The price of the product.",
          "maximum": 3.4028234663852886e38,
          "minimum": 0,
          "type": "number"
        },
        "tags": {
          "description": "The tags associated with the product.",
          "items": {
            "type": "string"
          },
          "type": "array"
        },
        "location": {
          "$ref": "#/$defs/buf.protoschema.test.v1.Product.Location.jsonschema.strict.json",
          "description": "The location of the product."
        }
      },
      "required": ["productId", "productName", "price", "location"],
      "additionalProperties": false
    },
    "buf.protoschema.test.v1.Product.Location.jsonschema.strict.json": {
      "$schema": "https://json-schema.org/draft/2020-12/schema",
      "additionalProperties": true,
      "description": "A point on the earth's surface.",
      "properties": {
        "lat": {
          "maximum": 90,
          "minimum": -90,
          "type": "number"
        },
        "long": {
          "maximum": 180,
          "minimum": -180,
          "type": "number"
        }
      },
      "required": ["lat", "long"],
      "title": "Location",
      "type": "object"
    }
  }
}
```

</details>

### Options

The JSON Schema plugin supports the following options:

- `target` - Any of `proto`, `json`, `proto-bundle`, `json-bundle`, `proto-strict`, `json-strict`,
  `proto-strict-bundle`, `json-strict-bundle`, or `all` separated by `+` (e.g. `proto+json`). Defaults to `all`.
  - If `proto`, the schema will be generated with Protobuf field names (e.g. `product_id`,
    `product_name`).
  - If `json`, the schema will be generated with JSON field names (e.g. `productId`, `productName`).
  - If suffixed with `-bundle`, the schema will include all dependencies in a single file.
  - If suffixed with `-strict`, the schema will not allow aliases, string numbers, or any other
    non-normalized representation. Strict is useful when the validated JSON data is used directly
    instead of being converted to a Protobuf message. Requires the "always emit fields without
    presence" option when using [Protobuf JSON](https://protobuf.dev/programming-guides/json/#json-options).
- `strict-names` - One of `true` or `false`. Defaults to `false`. If `true`, the generated schema will
  not allow mixing of JSON and Protobuf field names.
- `additional_properties` - If `true`, the generated schema will set `additionalProperties` to
  `true`, causing unknown fields to be ignored instead of erroring. Defaults to `false`. Useful when a
  client/sender may have a different version the schema than the server/receiver. Similar to the
  "ignore unknown fields" option in [Protobuf JSON](https://protobuf.dev/programming-guides/json/#json-options).

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
