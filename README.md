# Grison - Graph JSON

**Work in progress - do not use!**

Like `encoding/json` but stores graphs instead of trees.

Solves the following problems:

* Loops in the graph.
* Nodes in the graph referenced via interface rather than a pointer
  to a concrete type.

Given that there's no obvious "starting point" in a graph and, in fact,
there may not be a single node from which all other nodes are reachable,
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