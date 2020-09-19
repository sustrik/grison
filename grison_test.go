package main

import (
	"strings"
	"testing"
)

func MarshalTest(t *testing.T, m interface{}, expected string) {
	b, err := Marshal(m)
	if err != nil {
		t.Errorf("error encountered: %v", err)
	}
	actual := strings.TrimSpace(string(b))
	if actual != expected {
		t.Errorf("marshal error.\nexpect=%s\nactual=%v", expected, actual)
	}
}

func TestBasicTypes(t *testing.T) {
	type Node struct {
		A int
		B uint32
		C float32
		D bool
		E string
		F []byte
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
			},
		},
	}
	MarshalTest(t, m, `{"Node":{"#1":{"A":-42,"B":42,"C":1.1,"D":true,"E":"foo","F":"AQID"}}}`)
}

func TestRefEscape(t *testing.T) {
	type Node struct {
		A string
		B string
		C string
	}
	type Master struct {
		Node []*Node
	}
	m := &Master{
		Node: []*Node{
			&Node{
				A: "&foo",
				B: "&&foo",
				C: "foo&bar",
			},
		},
	}
	MarshalTest(t, m, `{"Node":{"#1":{"A":"&&foo","B":"&&&foo","C":"foo&bar"}}}`)
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
				},
			},
		},
	}
	MarshalTest(t, m, `{"Node":{"#1":{"A":["a","b","c"],"B":[[4,5],[6],null]}}}`)
}

func TestMaps(t *testing.T) {
	type Node struct {
		A map[string]int
		B map[int]string
		C map[int]map[int]int
	}
	type Master struct {
		Node []*Node
	}
	m := &Master{
		Node: []*Node{
			&Node{
				A: map[string]int{"a": 1, "b": 2, "c": 3},
				B: map[int]string{1: "a", 2: "b", 3: "c"},
				C: map[int]map[int]int{
					0: map[int]int{1: 2, 3: 4},
					5: map[int]int{6: 7},
				},
			},
		},
	}
	MarshalTest(t, m, `{"Node":{"#1":{"A":{"a":1,"b":2,"c":3},"B":{"1":"a","2":"b","3":"c"},"C":{"0":{"1":2,"3":4},"5":{"6":7}}}}}`)
}
