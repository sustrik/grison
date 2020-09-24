/*
	Copyright (c) 2020 Martin Sustrik

	Permission is hereby granted, free of charge, to any person obtaining a copy
	of this software and associated documentation files (the "Software"),
	to deal in the Software without restriction, including without limitation
	the rights to use, copy, modify, merge, publish, distribute, sublicense,
	and/or sell copies of the Software, and to permit persons to whom
	the Software is furnished to do so, subject to the following conditions:
	The above copyright notice and this permission notice shall be included
	in all copies or substantial portions of the Software.
	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
	THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
	FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
	IN THE SOFTWARE.
*/

package grison

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// Encoder handles encoding of graphs into grison format.
type Encoder struct {
	// Node types (the structs, not the pointers).
	types map[reflect.Type]string
	// Objects marshalled so far.
	objects map[string]map[string]json.RawMessage
	// Map of object pointers to IDs of the objects.
	ids map[interface{}]string
	// Last generated object ID.
	id uint64
}

// NewEncoder creates new grison encoder, based on the supplied master structure.
func NewEncoder(m interface{}) (*Encoder, error) {
	enc := &Encoder{
		objects: make(map[string]map[string]json.RawMessage),
		ids:     make(map[interface{}]string),
	}
	tps, nms, err := scrapeMasterStruct(m)
	if err != nil {
		return nil, err
	}
	enc.types = tps
	for nm, _ := range nms {
		enc.objects[nm] = make(map[string]json.RawMessage)
	}
	return enc, nil
}

func (enc *Encoder) isNodeType(tp reflect.Type) bool {
	_, ok := enc.types[tp]
	return ok
}

func (enc *Encoder) allocate(obj interface{}) (string, bool) {
	// Use the pointer as a hash key.
	id, ok := enc.ids[obj]
	if ok {
		return id, true
	}
	// Generate new ID.
	enc.id++
	id = fmt.Sprintf("#%d", enc.id)
	enc.ids[obj] = id
	return id, false
}

func (enc *Encoder) insert(tp reflect.Type, id string, rm json.RawMessage) {
	enc.objects[enc.types[tp]][id] = rm
}

func (enc *Encoder) getJSON() ([]byte, error) {
	return json.Marshal(enc.objects)
}

func (enc *Encoder) marshalAny(obj reflect.Value) ([]byte, error) {
	switch obj.Kind() {
	case reflect.Ptr:
		return enc.marshalPtr(obj)
	case reflect.Interface:
		return enc.marshalInterface(obj)
	case reflect.Struct:
		return enc.marshalStruct(obj)
	case reflect.Slice:
		return enc.marshalSlice(obj)
	case reflect.Array:
		return enc.marshalArray(obj)
	case reflect.Map:
		return enc.marshalMap(obj)
	default:
		return json.Marshal(obj.Interface())
	}
}

func (enc *Encoder) marshalPtr(obj reflect.Value) ([]byte, error) {
	if obj.IsNil() {
		return json.Marshal(nil)
	}
	if enc.isNodeType(obj.Elem().Type()) {
		return enc.marshalNode(obj)
	}
	return enc.marshalAny(obj.Elem())
}

func (enc *Encoder) marshalInterface(obj reflect.Value) ([]byte, error) {
	if obj.IsNil() {
		return json.Marshal(nil)
	}
	tp := obj.Elem().Elem().Type()
	if !enc.isNodeType(tp) {
		return nil, fmt.Errorf("object behind an interface is not a node, it is %v", tp)
	}
	return enc.marshalNode(obj.Elem())
}

func (enc *Encoder) marshalNode(obj reflect.Value) ([]byte, error) {
	id, exists := enc.allocate(obj.Interface())
	eobj := obj.Elem().Interface()
	if !exists {
		rm, err := enc.marshalStruct(reflect.ValueOf(eobj))
		if err != nil {
			return nil, err
		}
		enc.insert(reflect.TypeOf(eobj), id, rm)
	}
	ref := fmt.Sprintf("%s:%s", enc.types[reflect.TypeOf(eobj)], id)
	return json.Marshal(map[string]string{"$ref": ref})
}

func (enc *Encoder) marshalStruct(obj reflect.Value) ([]byte, error) {
	m := make(map[string]json.RawMessage)
	tp := obj.Type()
	for i := 0; i < obj.NumField(); i++ {
		key := tp.Field(i).Name
		elem, err := enc.marshalAny(obj.Field(i))
		if err != nil {
			return []byte{}, err
		}
		m[key] = elem
	}
	return json.Marshal(m)
}

func (enc *Encoder) marshalArray(obj reflect.Value) ([]byte, error) {
	var s []json.RawMessage
	for i := 0; i < obj.Len(); i++ {
		elem, err := enc.marshalAny(obj.Index(i))
		if err != nil {
			return []byte{}, err
		}
		s = append(s, elem)
	}
	return json.Marshal(s)
}

func (enc *Encoder) marshalSlice(obj reflect.Value) ([]byte, error) {
	if obj.Type() == reflect.TypeOf([]byte{}) {
		return json.Marshal(obj.Interface())
	}
	return enc.marshalArray(obj)
}

func (enc *Encoder) marshalMap(obj reflect.Value) ([]byte, error) {
	m := make(map[string]json.RawMessage)
	keys := obj.MapKeys()
	for _, k := range keys {
		key := fmt.Sprintf("%v", k.Interface())
		elem, err := enc.marshalAny(obj.MapIndex(k))
		if err != nil {
			return []byte{}, err
		}
		m[key] = elem
	}
	return json.Marshal(m)
}

// Marshal encodes the supplied graph into grison format.
func Marshal(m interface{}) ([]byte, error) {
	enc, err := NewEncoder(m)
	if err != nil {
		return nil, err
	}
	ms := reflect.ValueOf(m).Elem()
	for i := 0; i < ms.NumField(); i++ {
		fld := ms.Field(i)
		for j := 0; j < fld.Len(); j++ {
			_, err = enc.marshalAny(fld.Index(j))
			if err != nil {
				return nil, err
			}
		}
	}
	return enc.getJSON()
}
