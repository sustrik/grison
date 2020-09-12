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

func (g *grison) marshalAny(obj interface{}) ([]byte, error) {
	v := reflect.ValueOf(obj)
	switch v.Kind() {
	case reflect.Ptr:
		if g.isNodeType(v.Elem().Type()) {
			return g.marshalNode(v.Interface())
		}
		return g.marshalAny(v.Elem().Interface())
	case reflect.Interface:
		panic("interface")
	case reflect.Struct:
		return g.marshalStruct(obj)
	case reflect.Slice:
		return g.marshalSlice(obj)
	case reflect.Map:
		return g.marshalMap(obj)
	default:
		r, err := jsonMarshal(obj)
		if err != nil {
			return []byte{}, err
		}
		if len(r) > 0 && string(r[0:2]) == "\"&" {
			r = append([]byte("\"&"), r[1:]...)
		}
		return r, nil
	}
}

func (g *grison) marshalNode(obj interface{}) ([]byte, error) {
	id, exists := g.allocate(obj)
	eobj := reflect.ValueOf(obj).Elem().Interface()
	if !exists {
		rm, err := g.marshalStruct(eobj)
		if err != nil {
			return nil, err
		}
		g.insert(reflect.TypeOf(eobj), id, rm)
	}
	ref := fmt.Sprintf("&%s:%s", g.types[reflect.TypeOf(eobj)], id)
	return jsonMarshal(ref)
}

func (g *grison) marshalStruct(obj interface{}) ([]byte, error) {
	v := reflect.ValueOf(obj)
	//if g.isNodeType(v.Type()) {
	//	return nil, errors.New("Node object used as embedded structure.")
	//}
	b := []byte("{")
	tp := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field, err := json.Marshal(tp.Field(i).Name)
		if err != nil {
			return []byte{}, err
		}
		value, err := g.marshalAny(v.Field(i).Interface())
		if err != nil {
			return []byte{}, err
		}
		b = append(b, field...)
		b = append(b, []byte(":")...)
		b = append(b, value...)
		if i < v.NumField()-1 {
			b = append(b, []byte(",")...)
		}
	}
	b = append(b, []byte("}")...)
	return b, nil
}

func (g *grison) marshalSlice(obj interface{}) ([]byte, error) {
	v := reflect.ValueOf(obj)
	if v.Type() == reflect.TypeOf([]byte{}) {
		return jsonMarshal(obj)
	}
	b := []byte("[")
	for i := 0; i < v.Len(); i++ {
		elem, err := g.marshalAny(v.Index(i).Interface())
		if err != nil {
			return []byte{}, err
		}
		b = append(b, elem...)
		if i < v.Len()-1 {
			b = append(b, []byte(",")...)
		}
	}
	b = append(b, []byte("]")...)
	return b, nil
}

func (g *grison) marshalMap(obj interface{}) ([]byte, error) {
	v := reflect.ValueOf(obj)
	b := []byte("{")
	keys := v.MapKeys()
	for i, k := range keys {
		key, err := g.marshalAny(k.Interface())
		if err != nil {
			return []byte{}, err
		}
		b = append(b, key...)
		b = append(b, []byte(":")...)
		elem, err := g.marshalAny(v.MapIndex(k).Interface())
		if err != nil {
			return []byte{}, err
		}
		b = append(b, elem...)
		if i < len(keys)-1 {
			b = append(b, []byte(",")...)
		}
	}
	b = append(b, []byte("}")...)
	return b, nil
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
			_, err = g.marshalAny(fld.Index(j).Interface())
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
