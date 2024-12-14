// Copyright 2016 Google Inc.
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

package model

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode"

	"shenzhen-go/source"
)

var (
	multiplicityVarRE   = regexp.MustCompile(`\b[nN]\b`)
	multiplicityUsageRE = regexp.MustCompile(`\bmultiplicity\b`)
	instanceNumUsageRE  = regexp.MustCompile(`\binstanceNumber\b`)
)

// Node models a goroutine. This is the "real" model type for nodes.
// It can be marshalled and unmarshalled to JSON sensibly.
type Node struct {
	Part         Part
	Name         string
	Comment      string
	Enabled      bool
	Multiplicity string
	Wait         bool
	X, Y         float64
	Connections  map[string]string // Pin name -> channel name
	Impl         PartImpl          // Final implementation after type inference

	TypeParams map[string]*source.Type // Local type parameter -> stringy type
	PinTypes   map[string]*source.Type // Pin name -> inferred type of pin
}

// Copy returns a copy of this node, but with an empty name, nil connections, and a clone of the Part.
func (n *Node) Copy() *Node {
	n0 := &Node{
		Name:         "",
		Enabled:      n.Enabled,
		Multiplicity: n.Multiplicity,
		Wait:         n.Wait,
		Part:         n.Part.Clone(),
		// TODO: find a better location
		X: n.X + 8,
		Y: n.Y + 100,
	}
	n0.RefreshConnections()
	return n0
}

// ExpandedMult expands Multiplicity into an expression that uses calls runtime.NumCPU.
func (n *Node) ExpandedMult() string {
	// TODO: Do it with parser.ParseExpr(n.Multiplicity) and substituting in
	// &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent("runtime"), Sel: ast.NewIdent("NumCPU")}}
	// with a parentWalker...
	return multiplicityVarRE.ReplaceAllString(n.Multiplicity, "runtime.NumCPU()")
}

// UsesMultiplicity returns true if multiplicity != 1 or the head/body/tail use the multiplicity variable.
func (n *Node) UsesMultiplicity() bool {
	// Again, could do this more properly by parsing the code.
	return n.Multiplicity != "1" ||
		multiplicityUsageRE.MatchString(n.Impl.Head) ||
		multiplicityUsageRE.MatchString(n.Impl.Body) ||
		multiplicityUsageRE.MatchString(n.Impl.Tail)
}

// UsesInstanceNum returns true if the body uses the "instanceNum"
func (n *Node) UsesInstanceNum() bool {
	return instanceNumUsageRE.MatchString(n.Impl.Body)
}

// PinFullTypes is a map from pin names to full resolved types:
// pinName <-chan someType or pinName chan<- someType.
// Requires InferTypes to have been called.
func (n *Node) PinFullTypes() map[string]string {
	pins := n.Part.Pins()
	m := make(map[string]string, len(pins))
	for pn, p := range pins {
		m[pn] = p.Direction.Type() + " " + n.PinTypes[pn].String()
	}
	return m
}

// Mangle turns an arbitrary name into a similar-looking identifier.
func Mangle(name string) string {
	base := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return '_'
		}
		if !unicode.IsLetter(r) && r != '_' && !unicode.IsDigit(r) {
			return -1
		}
		return r
	}, name)
	var f rune
	for _, r := range base {
		f = r
		break
	}
	if unicode.IsDigit(f) {
		base = "_" + base
	}
	return base
}

// Identifier turns the name into a similar-looking identifier.
func (n *Node) Identifier() string {
	return Mangle(n.Name)
}

type jsonNode struct {
	*PartJSON
	Comment      string            `json:"comment,omitempty"`
	Enabled      bool              `json:"enabled"`
	Wait         bool              `json:"wait"`
	Multiplicity string            `json:"multiplicity,omitempty"`
	X            float64           `json:"x"`
	Y            float64           `json:"y"`
	Connections  map[string]string `json:"connections"`
}

// MarshalJSON encodes the node and part as JSON.
func (n *Node) MarshalJSON() ([]byte, error) {
	pj, err := MarshalPart(n.Part)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&jsonNode{
		PartJSON:     pj,
		Comment:      n.Comment,
		Enabled:      n.Enabled,
		Wait:         n.Wait,
		Multiplicity: n.Multiplicity,
		X:            n.X,
		Y:            n.Y,
		Connections:  n.Connections,
	})
}

// UnmarshalJSON decodes the node and part as JSON.
func (n *Node) UnmarshalJSON(j []byte) error {
	var mp jsonNode
	if err := json.Unmarshal(j, &mp); err != nil {
		return err
	}
	p, err := mp.PartJSON.Unmarshal()
	if err != nil {
		return err
	}
	if mp.Multiplicity == "" {
		mp.Multiplicity = "1"
	}
	n.Comment = mp.Comment
	n.Enabled = mp.Enabled
	n.Wait = mp.Wait
	n.Multiplicity = mp.Multiplicity
	n.Part = p
	n.X, n.Y = mp.X, mp.Y
	n.Connections = mp.Connections
	n.RefreshConnections()
	return nil
}

// RefreshConnections filters n.Connections to ensure only pins defined by the
// part are in the map, and any new ones are mapped to "nil".
func (n *Node) RefreshConnections() {
	pins := n.Part.Pins()
	conns := make(map[string]string, len(pins))
	for name := range pins {
		c := n.Connections[name]
		if c == "" {
			c = "nil"
		}
		conns[name] = c
	}
	n.Connections = conns
}

// RefreshImpl refreshes Impl from the Part.
func (n *Node) RefreshImpl() {
	n.Impl = n.Part.Impl(n)
}
