package grison

import (
	"fmt"
	"testing"
)

func TestMarshalIndent(t *testing.T) {
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
	b, err := MarshalIndent(m, ">", "  ")
	if err != nil {
		t.Errorf("encoding error encountered: %v", err)
	}
	expect := `{
>  "Node": {
>    "#1": {
>      "A": [
>        "a",
>        "b"
>      ],
>      "B": [
>        [
>          4,
>          5
>        ],
>        [
>          6,
>          7
>        ]
>      ]
>    }
>  }
>}`
	if string(b) != expect {
		t.Errorf("unexpected marshal result\n%s", string(b))
	}
}

type Parent struct {
	Name     string
	Sex      string
	Spouse   *Parent
	Children []*Child
}
type Child struct {
	Name   string
	Age    int
	Father *Parent
	Mother *Parent
}

func TestExample(t *testing.T) {
	type Master struct {
		Parents  []*Parent
		Children []*Child
	}
	m := &Master{
		Parents: []*Parent{
			&Parent{
				Name: "Alice",
				Sex:  "Female",
			},
			&Parent{
				Name: "Bob",
				Sex:  "Male",
			},
		},
		Children: []*Child{
			&Child{
				Name: "Carol",
				Age:  10,
			},
			&Child{
				Name: "Dan",
				Age:  8,
			},
		},
	}
	m.Parents[0].Spouse = m.Parents[1]
	m.Parents[0].Children = []*Child{m.Children[0], m.Children[1]}
	m.Parents[1].Children = []*Child{m.Children[0], m.Children[1]}
	m.Parents[1].Spouse = m.Parents[0]
	m.Children[0].Father = m.Parents[1]
	m.Children[0].Mother = m.Parents[0]
	m.Children[1].Father = m.Parents[1]
	m.Children[1].Mother = m.Parents[0]
	b, err := MarshalIndent(m, "", "    ")
	if err != nil {
		t.Errorf("encoding error encountered: %v", err)
	}
	fmt.Println(string(b))
}
