# Grison - Graph JSON

**Work in progress - do not use!**

Like `encoding/json` but stores graphs instead of trees.

Solves the following problems:

* Loops in the graph.
* Encoding and decoding of interface types.

Given that there's no obvious "starting point" in a graph and, in fact,
there may not be a single node from which all the other nodes are reachable,
grison requires that you create a "master" structure, i.e. a structure
with pointers to all the nodes:

```go
type Master struct{
    Foo []*Foo
    Bar []*Bar
}
```

You can then marshal/unmarshal this master structure:

```go
var m1 Master
...
b, err := grison.Marshal(&m1)
...
var m2 Master
err = grison.Unmarshal(b, &m2)
```

### Example

```go
type Parent struct {
	Name     string   `grison:"name"`
	Sex      string   `grison:"sex"`
	Spouse   *Parent  `grison:"spouse"`
	Children []*Child `grison:"children"`
}

type Child struct {
	Name   string  `grison:"name"`
	Age    int     `grison:"Age"`
	Father *Parent `grison:"father"`
	Mother *Parent `grison:"mother"`
}

type Master struct {
    Parents  []*Parent `grison:"parents"`
    Children []*Child  `grison:"children"`
}
```

The data structures above produce the following JSON output:

```json
{
    "children": {
        "#3": {
            "Age": 10,
            "father": {"$ref": "parents:#2"},
            "mother": {"$ref": "parents:#1"},
            "name": "Carol"
        },
        "#4": {
            "Age": 8,
            "father": {"$ref": "parents:#2"},
            "mother": {"$ref": "parents:#1"},
            "name": "Dan"
        }
    },
    "parents": {
        "#1": {
            "children": [
                {"$ref": "children:#3"},
                {"$ref": "children:#4"}
            ],
            "name": "Alice",
            "sex": "Female",
            "spouse": {"$ref": "parents:#2"}
        },
        "#2": {
            "children": [
                {"$ref": "children:#3"},
                {"$ref": "children:#4"}
            ],
            "name": "Bob",
            "sex": "Male",
            "spouse": {"$ref": "parents:#1"}
        }
    }
}
```