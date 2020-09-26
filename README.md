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

type Master struct {
    Parents  []*Parent
    Children []*Child
}
```

The data structures above produce the following JSON output:

```json
{
    "Children": {
        "#3": {
            "Age": 10,
            "Father": {"$ref": "Parents:#2"},
            "Mother": {"$ref": "Parents:#1"},
            "Name": "Carol"
        },
        "#4": {
            "Age": 8,
            "Father": {"$ref": "Parents:#2"},
            "Mother": {"$ref": "Parents:#1"},
            "Name": "Dan"
        }
    },
    "Parents": {
        "#1": {
            "Children": [
                {"$ref": "Children:#3"},
                {"$ref": "Children:#4"}
            ],
            "Name": "Alice",
            "Sex": "Female",
            "Spouse": {"$ref": "Parents:#2"}
        },
        "#2": {
            "Children": [
                {"$ref": "Children:#3"},
                {"$ref": "Children:#4"}
            ],
            "Name": "Bob",
            "Sex": "Male",
            "Spouse": {"$ref": "Parents:#1"}
        }
    }
}
```