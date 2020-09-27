package grison

import (
	"reflect"
	"strings"
	"testing"
	//"github.com/go-test/deep"
)

func MarshalTestWithOpts(t *testing.T, m interface{}, expected string, mopts MarshalOpts, uopts UnmarshalOpts) {
	b, err := MarshalWithOpts(m, mopts)
	if err != nil {
		t.Errorf("encoding error encountered: %v", err)
	}
	actual := strings.TrimSpace(string(b))
	if actual != expected {
		t.Errorf("unexpected marshal result.\nexpect=%s\nactual=%s", expected, actual)
	}
	var m2 interface{}
	m2 = reflect.New(reflect.TypeOf(m).Elem()).Interface()
	err = UnmarshalWithOpts([]byte(actual), m2, uopts)
	if err != nil {
		t.Errorf("decoding error encountered: %v", err)
	}
	if !reflect.DeepEqual(m, m2) {
		t.Errorf("unexpected unmarshal result")
	}
	//if diff := deep.Equal(m, m2); diff != nil {
	//	t.Errorf("unexpected unmarshal result.\n%v", diff)
	//}
}

func MarshalTest(t *testing.T, m interface{}, expected string) {
	MarshalTestWithOpts(t, m, expected, MarshalOpts{}, UnmarshalOpts{})
}

func TestMimimal(t *testing.T) {
	type Node struct {
		A int
	}
	type Master struct {
		Node []*Node
	}
	m := &Master{
		Node: []*Node{
			&Node{
				A: 2,
			},
		},
	}
	MarshalTest(t, m, `{"Node":{"#1":{"A":2}}}`)
}

func TestNoNodes(t *testing.T) {
	type Node struct{}
	type Master struct {
		Node []*Node
	}
	m := &Master{
		Node: []*Node{},
	}
	MarshalTest(t, m, `{"Node":{}}`)
}

func TestBasicTypes(t *testing.T) {
	type Node struct {
		A int
		B uint32
		C float32
		D bool
		E string
		F []byte
		G [3]byte
	}
	type Master struct {
		Node []*Node
	}
	m := &Master{
		Node: []*Node{
			&Node{
				A: -42,
				B: 42,
				C: 1.1,
				D: true,
				E: "foo",
				F: []byte{1, 2, 3},
				G: [3]byte{4, 5, 6},
			},
		},
	}
	MarshalTest(t, m, `{"Node":{"#1":{"A":-42,"B":42,"C":1.1,"D":true,"E":"foo","F":"AQID","G":[4,5,6]}}}`)
}

func TestSlices(t *testing.T) {
	type Node struct {
		A []string
		B [][]int
	}
	type Master struct {
		Node []*Node
	}
	m := &Master{
		Node: []*Node{
			&Node{
				A: []string{"a", "b", "c"},
				B: [][]int{
					[]int{4, 5},
					[]int{6},
					[]int{},
					nil,
				},
			},
		},
	}
	MarshalTest(t, m, `{"Node":{"#1":{"A":["a","b","c"],"B":[[4,5],[6],[],null]}}}`)
}

func TestArrays(t *testing.T) {
	type Node struct {
		A [2]string
		B [2][2]int
	}
	type Master struct {
		Node []*Node
	}
	m := &Master{
		Node: []*Node{
			&Node{
				A: [2]string{"a", "b"},
				B: [2][2]int{
					[2]int{4, 5},
					[2]int{6, 7},
				},
			},
		},
	}
	MarshalTest(t, m, `{"Node":{"#1":{"A":["a","b"],"B":[[4,5],[6,7]]}}}`)
}

func TestMaps(t *testing.T) {
	type Node struct {
		A map[string]int
		B map[string]string
		C map[string]map[string]int
	}
	type Master struct {
		Node []*Node
	}
	m := &Master{
		Node: []*Node{
			&Node{
				A: map[string]int{"a": 1, "b": 2, "c": 3},
				B: map[string]string{"1": "a", "2": "b", "3": "c"},
				C: map[string]map[string]int{
					"0": map[string]int{"1": 2, "3": 4},
					"5": map[string]int{"6": 7},
					"3": map[string]int{},
					"9": nil,
				},
			},
		},
	}
	MarshalTest(t, m, `{"Node":{"#1":{"A":{"a":1,"b":2,"c":3},"B":{"1":"a","2":"b","3":"c"},"C":{"0":{"1":2,"3":4},"3":{},"5":{"6":7},"9":null}}}}`)
}

func TestInterface(t *testing.T) {
	type Node struct {
		I interface{}
	}
	type Bar struct {
		I int
	}
	type Master struct {
		Node []*Node
		Bar  []*Bar
	}
	m := &Master{
		Node: []*Node{
			&Node{},
			&Node{},
		},
		Bar: []*Bar{
			&Bar{I: 42},
		},
	}
	m.Node[0].I = m.Node[1]
	m.Node[1].I = m.Bar[0]
	MarshalTest(t, m, `{"Bar":{"#3":{"I":42}},"Node":{"#1":{"I":{"$ref":"Node:#2"}},"#2":{"I":{"$ref":"Bar:#3"}}}}`)
}

func TestNil(t *testing.T) {
	type Node struct {
		P *int
		I interface{}
	}
	type Master struct {
		Node []*Node
	}
	m := &Master{
		Node: []*Node{
			&Node{},
		},
	}
	MarshalTest(t, m, `{"Node":{"#1":{"I":null,"P":null}}}`)
}

func TestRef(t *testing.T) {
	type Node struct {
		N *Node
	}
	type Master struct {
		Node []*Node
	}
	m := &Master{
		Node: []*Node{
			&Node{},
			&Node{},
		},
	}
	m.Node[0].N = m.Node[1]
	MarshalTest(t, m, `{"Node":{"#1":{"N":{"$ref":"Node:#2"}},"#2":{"N":null}}}`)
}

func TestLoop(t *testing.T) {
	type Node struct {
		N *Node
	}
	type Master struct {
		Node []*Node
	}
	m := &Master{
		Node: []*Node{
			&Node{},
			&Node{},
		},
	}
	m.Node[0].N = m.Node[1]
	m.Node[1].N = m.Node[0]
	MarshalTest(t, m, `{"Node":{"#1":{"N":{"$ref":"Node:#2"}},"#2":{"N":{"$ref":"Node:#1"}}}}`)
}

func TestIgnore(t *testing.T) {
	type Node struct {
		A int `grison:"-"`
		B int
	}
	type Master struct {
		Foo  int `grison:"-"`
		Node []*Node
	}
	m := &Master{
		Node: []*Node{
			&Node{
				A: 0,
				B: 4,
			},
		},
	}
	MarshalTest(t, m, `{"Node":{"#1":{"B":4}}}`)
}

func TestTagNames(t *testing.T) {
	type Node struct {
		A int `grison:"foo"`
		B int `grison:"bar"`
	}
	type Master struct {
		Node []*Node `grison:"noodle"`
	}
	m := &Master{
		Node: []*Node{
			&Node{
				A: 2,
				B: 4,
			},
		},
	}
	MarshalTest(t, m, `{"noodle":{"#1":{"bar":4,"foo":2}}}`)
}

func TestOmitEmpty(t *testing.T) {
	type Node struct {
		A int    `grison:"foo,omitempty"`
		B string `grison:"bar,omitempty"`
		C int    `grison:",omitempty"`
	}
	type Noodle struct{}
	type Master struct {
		Node   []*Node
		Noodle []*Noodle `grison:",omitempty"`
	}
	m := &Master{
		Node: []*Node{
			&Node{
				C: 3,
			},
		},
	}
	MarshalTest(t, m, `{"Node":{"#1":{"C":3}}}`)
}

type Node1 struct {
	ID string
	I  int
}

func (n *Node1) GetID() string {
	return n.ID
}

func TestIDs(t *testing.T) {
	type Master struct {
		Node1 []*Node1
	}
	m := &Master{
		Node1: []*Node1{
			&Node1{
				ID: "foo",
				I:  33,
			},
		},
	}
	MarshalTestWithOpts(t, m, `{"Node1":{"foo":{"I":33,"ID":"foo"}}}`,
		MarshalOpts{GetIDs: true}, UnmarshalOpts{})
}

type Prop int

func (p *Prop) MarshalJSON() ([]byte, error) {
	return []byte(`"foo"`), nil
}

func (p *Prop) UnmarshalJSON(data []byte) error {
	*p = 33
	return nil
}

func TestCustomMarshaler(t *testing.T) {
	type Node struct {
		P  Prop
		PP *Prop
	}
	type Master struct {
		Node []*Node
	}
	var prop Prop = 33
	m := &Master{
		Node: []*Node{
			&Node{
				P:  33,
				PP: &prop,
			},
		},
	}
	MarshalTest(t, m, `{"Node":{"#1":{"P":"foo","PP":"foo"}}}`)
}
