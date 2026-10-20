package desk

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Node is one node of the desk's state tree: a type, its properties and its
// children. Path is the node's place from the root (child indices).
type Node struct {
	Type     string
	Props    map[string]Value
	Children []*Node
	Path     []int
}

// Get returns a property of the node.
func (n *Node) Get(name string) (Value, bool) {
	if n == nil {
		return Value{}, false
	}
	v, ok := n.Props[name]
	return v, ok
}

// Num reads a numeric property (0 if missing).
func (n *Node) Num(name string) float64 { v, _ := n.Get(name); return v.Num() }

// Int reads a whole-number property.
func (n *Node) Int(name string) int64 { v, _ := n.Get(name); return v.Int() }

// Bool reads a boolean property.
func (n *Node) Bool(name string) bool { v, _ := n.Get(name); return v.Truth() }

// Str reads a text property.
func (n *Node) Str(name string) string { v, _ := n.Get(name); return v.Str() }

// Has reports whether the node carries the property.
func (n *Node) Has(name string) bool { _, ok := n.Get(name); return ok }

// At walks to the descendant at path (child indices from this node).
func (n *Node) At(path []int) *Node {
	cur := n
	for _, i := range path {
		if cur == nil || i < 0 || i >= len(cur.Children) {
			return nil
		}
		cur = cur.Children[i]
	}
	return cur
}

// walk visits the node and its descendants in document order.
func (n *Node) walk(fn func(*Node)) {
	if n == nil {
		return
	}
	fn(n)
	for _, c := range n.Children {
		c.walk(fn)
	}
}

// rawNode mirrors the tree JSON: {"t":type,"p":{…},"y":"types","c":[…]}.
type rawNode struct {
	T string            `json:"t"`
	P json.RawMessage   `json:"p"`
	Y string            `json:"y"`
	C []json.RawMessage `json:"c"`
}

// parseNode builds a node (and its subtree) from the raw JSON at path. The
// property kinds come from the JSON itself; the "y" letters sharpen a few
// (an integer written without a point still reads as a number either way).
func parseNode(raw json.RawMessage, path []int) (*Node, error) {
	var rn rawNode
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&rn); err != nil {
		return nil, err
	}
	n := &Node{Type: rn.T, Props: map[string]Value{}, Path: path}
	if len(rn.P) > 0 {
		var props map[string]interface{}
		pd := json.NewDecoder(bytes.NewReader(rn.P))
		pd.UseNumber()
		if err := pd.Decode(&props); err != nil {
			return nil, fmt.Errorf("node %s props: %w", rn.T, err)
		}
		for name, x := range props {
			n.Props[name] = fromJSON(x, 0)
		}
	}
	for i, c := range rn.C {
		cp := make([]int, len(path)+1)
		copy(cp, path)
		cp[len(path)] = i
		child, err := parseNode(c, cp)
		if err != nil {
			return nil, err
		}
		n.Children = append(n.Children, child)
	}
	return n, nil
}

// treeDoc is the shape of GET api/tree.
type treeDoc struct {
	Epoch uint64          `json:"epoch"`
	Seq   uint64          `json:"seq"`
	Stale bool            `json:"stale"`
	Meta  meta            `json:"meta"`
	Root  json.RawMessage `json:"root"`
}

// meta is the part of the tree document that describes the model.
type meta struct {
	Layout struct {
		Model          string `json:"model"`
		ModelName      string `json:"modelName"`
		Firmware       string `json:"firmware"`
		PhysicalFaders int    `json:"physicalFaders"`
		PadsPerBank    int    `json:"padsPerBank"`
		PadBanks       int    `json:"padBanks"`
		MixPerSource   int    `json:"mixPerSource"`
		MicInputs      int    `json:"micInputs"`
	} `json:"layout"`
	Faders     []Name `json:"faders"`
	Sources    []Name `json:"sources"`
	Outputs    []Name `json:"outputs"`
	UnityFader int64  `json:"unityFader"`
}

// Name is an id and a human label (a fader, a source or an output).
type Name struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// parseTree turns a tree document into a root node and its meta.
func parseTree(doc []byte) (*Node, *treeDoc, error) {
	var td treeDoc
	dec := json.NewDecoder(bytes.NewReader(doc))
	dec.UseNumber()
	if err := dec.Decode(&td); err != nil {
		return nil, nil, err
	}
	root, err := parseNode(td.Root, nil)
	if err != nil {
		return nil, nil, err
	}
	return root, &td, nil
}

// runs indexes the tree: every node of a given type in document order, so a
// strip, a fader or a meter can be found by its number.
type runs struct {
	byType map[string][]*Node
}

func indexRuns(root *Node) *runs {
	r := &runs{byType: map[string][]*Node{}}
	root.walk(func(n *Node) {
		r.byType[n.Type] = append(r.byType[n.Type], n)
	})
	return r
}

func (r *runs) nth(typ string, i int) *Node {
	ns := r.byType[typ]
	if i < 0 || i >= len(ns) {
		return nil
	}
	return ns[i]
}

func (r *runs) count(typ string) int { return len(r.byType[typ]) }
