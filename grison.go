package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

func jsonMarshal(v interface{}) ([]byte, error) {
	b := bytes.Buffer{}
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	err := enc.Encode(v)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

type grison struct {
	// Node types (the structs, not the pointers).
	types map[reflect.Type]string
	// Objects marshalled so far.
	objects map[string]map[string]json.RawMessage
	// Map of object pointers to IDs of the objects.
	ids map[interface{}]string
	// Last generated object ID.
	id uint64
}

func (g *grison) extractSchema(m interface{}) error {
	g.types = make(map[reflect.Type]string)
	g.objects = make(map[string]map[string]json.RawMessage)
	g.ids = make(map[interface{}]string)
	tp := reflect.TypeOf(m)
	if tp.Kind() != reflect.Ptr {
		return errors.New(fmt.Sprintf("Master structure must be passed as a pointer, is %T", m))
	}
	tp = tp.Elem()
	if tp.Kind() != reflect.Struct {
		return errors.New(fmt.Sprintf("Master structure is not a structure, is %T.", m))
	}
	for i := 0; i < tp.NumField(); i++ {
		fldtp := tp.Field(i).Type
		fldname := tp.Field(i).Name
		if fldtp.Kind() != reflect.Slice && fldtp.Kind() != reflect.Map {
			return errors.New(fmt.Sprintf("Master field %s, in not a map or a slice.", fldname))
		}
		fldtp = fldtp.Elem()
		if fldtp.Kind() != reflect.Ptr {
			return errors.New(fmt.Sprintf("Master field %s doesn't contain pointers.", fldname))
		}
		fldtp = fldtp.Elem()
		if fldtp.Kind() != reflect.Struct {
			return errors.New(fmt.Sprintf("Master field %s doesn't contain pointers to structs.", fldname))
		}
		// TODO: Check for duplicate types.
		g.types[fldtp] = fldname
		g.objects[fldname] = make(map[string]json.RawMessage)
	}
	// TODO: There should be no embedded node instances.
	return nil
}

func (g *grison) isNodeType(tp reflect.Type) bool {
	_, ok := g.types[tp]
	return ok
}

func (g *grison) allocate(obj interface{}) (string, bool) {
	// Use the pointer as a hash key.
	id, ok := g.ids[obj]
	if ok {
		return id, true
	}
	// Generate new ID.
	g.id++
	id = fmt.Sprintf("#%d", g.id)
	g.ids[obj] = id
	return id, false
}

func (g *grison) insert(tp reflect.Type, id string, rm json.RawMessage) {
	g.objects[g.types[tp]][id] = rm
}

func (g *grison) getJSON() ([]byte, error) {
	return jsonMarshal(g.objects)
}

func (g *grison) marshalAny(obj reflect.Value) ([]byte, error) {
	switch obj.Kind() {
	case reflect.Ptr:
		if g.isNodeType(obj.Elem().Type()) {
			return g.marshalNode(obj)
		}
		return g.marshalAny(obj.Elem())
	case reflect.Interface:
		panic("interface")
	case reflect.Struct:
		return g.marshalStruct(obj)
	case reflect.Slice:
		return g.marshalSlice(obj)
	case reflect.Map:
		return g.marshalMap(obj)
	default:
		r, err := jsonMarshal(obj.Interface())
		if err != nil {
			return []byte{}, err
		}
		if len(r) > 0 && string(r[0:2]) == "\"&" {
			r = append([]byte("\"&"), r[1:]...)
		}
		return r, nil
	}
}

func (g *grison) marshalNode(obj reflect.Value) ([]byte, error) {
	id, exists := g.allocate(obj.Interface())
	eobj := obj.Elem().Interface()
	if !exists {
		rm, err := g.marshalStruct(reflect.ValueOf(eobj))
		if err != nil {
			return nil, err
		}
		g.insert(reflect.TypeOf(eobj), id, rm)
	}
	ref := fmt.Sprintf("&%s:%s", g.types[reflect.TypeOf(eobj)], id)
	return jsonMarshal(ref)
}

func (g *grison) marshalStruct(obj reflect.Value) ([]byte, error) {
	m := make(map[string]json.RawMessage)
	tp := obj.Type()
	for i := 0; i < obj.NumField(); i++ {
		key := tp.Field(i).Name
		elem, err := g.marshalAny(obj.Field(i))
		if err != nil {
			return []byte{}, err
		}
		m[key] = elem
	}
	return jsonMarshal(m)
}

func (g *grison) marshalSlice(obj reflect.Value) ([]byte, error) {
	if obj.Type() == reflect.TypeOf([]byte{}) {
		return jsonMarshal(obj.Interface())
	}
	var s []json.RawMessage
	for i := 0; i < obj.Len(); i++ {
		elem, err := g.marshalAny(obj.Index(i))
		if err != nil {
			return []byte{}, err
		}
		s = append(s, elem)
	}
	return jsonMarshal(s)
}

func (g *grison) marshalMap(obj reflect.Value) ([]byte, error) {
	m := make(map[string]json.RawMessage)
	keys := obj.MapKeys()
	for _, k := range keys {
		key := fmt.Sprintf("%v", k.Interface())
		elem, err := g.marshalAny(obj.MapIndex(k))
		if err != nil {
			return []byte{}, err
		}
		m[key] = elem
	}
	return jsonMarshal(m)
}

func Marshal(m interface{}) ([]byte, error) {
	var g grison
	err := g.extractSchema(m)
	if err != nil {
		return nil, err
	}
	ms := reflect.ValueOf(m).Elem()
	for i := 0; i < ms.NumField(); i++ {
		fld := ms.Field(i)
		for j := 0; j < fld.Len(); j++ {
			_, err = g.marshalAny(fld.Index(j))
			if err != nil {
				return nil, err
			}
		}
	}
	return g.getJSON()
}

///////////////////////////////////////////

type Foo struct {
	B *Bar
}

type Bar struct {
	F *Foo
}

// master structure
type MyGraph struct {
	Foos []*Foo
	Bars []*Bar
}

func main() {
	mg := &MyGraph{
		Foos: []*Foo{
			&Foo{},
		},
		Bars: []*Bar{
			&Bar{},
		},
	}
	mg.Foos[0].B = mg.Bars[0]
	mg.Bars[0].F = mg.Foos[0]

	gs, err := Marshal(mg)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v\n", string(gs))
}
