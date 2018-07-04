// Copyright 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package source

// This is a bit crazy.
// Finding breaking examples is left as an exercise to the reader.

import (
	"fmt"
	"regexp"
)

var (
	typeParamRE       = regexp.MustCompile(`\$\w+`)
	typeParamQuotedRE = regexp.MustCompile(`\\\$\w+`)
)

// TypePattern represents a genericised type, e.g. $T, []$T, map[$K]$V.
type TypePattern struct {
	spec   string
	params []string
	re     *regexp.Regexp
}

// NewTypePattern parses a generic type into a TypePattern.
func NewTypePattern(spec string) *TypePattern {
	return &TypePattern{
		spec:   spec,
		params: typeParamRE.FindAllString(spec, -1),
		// "This'll never fail," he proclaimed, boldly...
		re: regexp.MustCompile(
			"^" + typeParamQuotedRE.ReplaceAllString(regexp.QuoteMeta(spec), `(.+?)`) + "$",
		),
	}
}

// Plain is true if the type pattern has no parameters.
func (p *TypePattern) Plain() bool {
	return len(p.params) == 0
}

// Params returns the slice of parameter names.
func (p *TypePattern) Params() []string {
	return p.params
}

// Match tests if the input type matches the pattern.
func (p *TypePattern) Match(t string) bool {
	return p.re.MatchString(t)
}

// FindParams matches the input type against the pattern, and produces a map of
// type parameters to the matching components, or an error if the
// type doesn't match the pattern.
func (p *TypePattern) FindParams(t string) (map[string][]string, error) {
	mt := p.re.FindStringSubmatch(t)
	if len(mt) == 0 {
		return nil, fmt.Errorf("type %q did not match pattern %q", t, p.spec)
	}
	mt = mt[1:]
	if len(mt) != len(p.params) {
		// This should be impossible, but here we are.
		return nil, fmt.Errorf("param count mismatch [%d != %d]", len(mt), len(p.params))
	}
	types := make(map[string][]string, len(p.params))
	for i, t := range mt {
		param := p.params[i]
		types[param] = append(types[param], t)
	}
	return types, nil
}

func (p *TypePattern) intersectsParams(types map[string]string) bool {
	for _, param := range p.params {
		if _, ok := types[param]; ok {
			return true
		}
	}
	return false
}

// Expand fills the pattern with the provided types. If a parameter is not
// in the input map, it is left unexpanded.
func (p *TypePattern) Expand(types map[string]string) string {
	if !p.intersectsParams(types) {
		return p.String()
	}
	return typeParamRE.ReplaceAllStringFunc(p.spec, func(in string) string {
		out := types[in]
		if out == "" {
			return in
		}
		return out
	})
}

// Curry is the same as Expand, but returns a new pattern.
// It checks if any of the provided type parameters are parameters of this pattern.
// If none are, it returns the existing pattern.
func (p *TypePattern) Curry(types map[string]string) *TypePattern {
	if !p.intersectsParams(types) {
		return p
	}
	// Do it the lazy way.
	return NewTypePattern(p.Expand(types))
}

// Lithify expands all parameters with a default.
func (p *TypePattern) Lithify(def string) string {
	if p.Plain() {
		return p.String()
	}
	return typeParamRE.ReplaceAllString(p.spec, def)
}

// Infer matches the input Go type against the pattern, and produces
// a map of type parameters to inferred types, or an error if the
// type doesn't match or there is a conflicting match (e.g.
// for type pattern `map[$T]$T`, when given `map[int]string`).
func (p *TypePattern) Infer(t string) (map[string]string, error) {
	mts, err := p.FindParams(t)
	if err != nil {
		return nil, err
	}
	types := make(map[string]string, len(p.params))
	for param, mt := range mts {
		cur := ""
		for _, t := range mt {
			if cur != "" && cur != t {
				return nil, fmt.Errorf("conflicting match for %s [%q != %q]", param, t, cur)
			}
			cur = t
		}
		if cur == "" {
			continue
		}
		types[param] = cur
	}
	return types, nil
}

func (p *TypePattern) String() string {
	if p == nil {
		return "<unspecified>"
	}
	return p.spec
}