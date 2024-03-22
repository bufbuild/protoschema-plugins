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

// GenerateOptions are the options for Generate.
type GenerateOptions interface {
	applyGenerateOptions(generateOptions *generateOptions)
}

// WithMaxDepth returns a GenerateOptions that sets the max depth.
func WithMaxDepth(maxDepth int) GenerateOptions {
	return generateOptionsFunc(func(options *generateOptions) {
		options.maxDepth = maxDepth
	})
}

// WithMaxRecursionDepth returns a GenerateOptions that sets the max recursion depth.
func WithMaxRecursionDepth(maxRecursionDepth int) GenerateOptions {
	return generateOptionsFunc(func(options *generateOptions) {
		options.maxRecursionDepth = maxRecursionDepth
	})
}

// WithGenerateAllMessages returns a GenerateOptions that generates all messages, not just those
// with the extension option.
func WithGenerateAllMessages() GenerateOptions {
	return generateOptionsFunc(func(options *generateOptions) {
		options.generateAllMessages = true
	})
}

type generateOptions struct {
	maxDepth            int
	maxRecursionDepth   int
	generateAllMessages bool
}

type generateOptionsFunc func(*generateOptions)

func (f generateOptionsFunc) applyGenerateOptions(generateOptions *generateOptions) {
	f(generateOptions)
}
