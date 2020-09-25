package grison

import (
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
