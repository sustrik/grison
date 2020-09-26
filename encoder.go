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

// encoder handles encoding of graphs into grison format.
type encoder struct {
	// Node types (the structs, not the pointers).
	types map[reflect.Type]string
	// Objects marshalled so far.
	objects map[string]map[string]json.RawMessage
	// Map of object pointers to IDs of the objects.
	ids map[interface{}]string
	// Last generated object ID.
	id uint64
	// Types marked with omitempty tag.
	omitEmpty []string
	opts      MarshalOpts
}

// newEncoder creates new grison encoder, based on the supplied master structure.
func newEncoder(m interface{}, opts MarshalOpts) (*encoder, error) {
	enc := &encoder{
		objects: make(map[string]map[string]json.RawMessage),
		ids:     make(map[interface{}]string),
		opts:    opts,
	}
	tps, nms, oe, err := scrapeMasterStruct(m, opts.IDField)
	if err != nil {
		return nil, err
	}
	enc.types = tps
	enc.omitEmpty = oe
	for nm := range nms {
		enc.objects[nm] = make(map[string]json.RawMessage)
	}
	return enc, nil
}

func (enc *encoder) isNodeType(tp reflect.Type) bool {
	_, ok := enc.types[tp]
	return ok
}

func (enc *encoder) allocate(obj interface{}, newid string) (string, bool) {
	// Use the pointer as a hash key.
	id, ok := enc.ids[obj]
	if ok {
		return id, true
	}
	if newid == "" {
		enc.id++
		id = fmt.Sprintf("#%d", enc.id)
	} else {
		id = newid
	}
	enc.ids[obj] = id
	return id, false
}

func (enc *encoder) insert(tp reflect.Type, id string, rm json.RawMessage) {
	enc.objects[enc.types[tp]][id] = rm
}

func (enc *encoder) getJSON() ([]byte, error) {
	enc.filterEmpty()
	return json.Marshal(enc.objects)
}

func (enc *encoder) getJSONIndent(prefix string, indent string) ([]byte, error) {
	enc.filterEmpty()
	return json.MarshalIndent(enc.objects, prefix, indent)
}

func (enc *encoder) filterEmpty() {
	for _, tp := range enc.omitEmpty {
		if len(enc.objects[tp]) == 0 {
			delete(enc.objects, tp)
		}
	}
}

func (enc *encoder) marshalAny(obj reflect.Value) ([]byte, error) {
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

func (enc *encoder) marshalPtr(obj reflect.Value) ([]byte, error) {
	if obj.IsNil() {
		return json.Marshal(nil)
	}
	if enc.isNodeType(obj.Elem().Type()) {
		return enc.marshalNode(obj)
	}
	return enc.marshalAny(obj.Elem())
}

func (enc *encoder) marshalInterface(obj reflect.Value) ([]byte, error) {
	if obj.IsNil() {
		return json.Marshal(nil)
	}
	tp := obj.Elem().Elem().Type()
	if !enc.isNodeType(tp) {
		return nil, fmt.Errorf("object behind an interface is not a node, it is %v", tp)
	}
	return enc.marshalNode(obj.Elem())
}

func (enc *encoder) marshalNode(obj reflect.Value) ([]byte, error) {
	var id string
	if enc.opts.IDField != "" {
		id = obj.Elem().FieldByName(enc.opts.IDField).String()
	}
	id, exists := enc.allocate(obj.Interface(), id)
	eobj := obj.Elem().Interface() // TODO: get rid of this back-and-forth
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

func (enc *encoder) marshalStruct(obj reflect.Value) ([]byte, error) {
	m := make(map[string]json.RawMessage)
	tp := obj.Type()
	for i := 0; i < obj.NumField(); i++ {
		ft := getFieldTags(tp.Field(i))
		if ft.ignore {
			continue
		}
		if ft.omitEmpty && obj.Field(i).IsZero() {
			continue
		}
		elem, err := enc.marshalAny(obj.Field(i))
		if err != nil {
			return []byte{}, err
		}
		m[ft.name] = elem
	}
	return json.Marshal(m)
}

func (enc *encoder) marshalArray(obj reflect.Value) ([]byte, error) {
	s := make([]json.RawMessage, 0, obj.Len())
	for i := 0; i < obj.Len(); i++ {
		elem, err := enc.marshalAny(obj.Index(i))
		if err != nil {
			return []byte{}, err
		}
		s = append(s, elem)
	}
	return json.Marshal(s)
}

func (enc *encoder) marshalSlice(obj reflect.Value) ([]byte, error) {
	if obj.IsNil() {
		return []byte("null"), nil
	}
	if obj.Type() == reflect.TypeOf([]byte{}) {
		return json.Marshal(obj.Interface())
	}
	return enc.marshalArray(obj)
}

func (enc *encoder) marshalMap(obj reflect.Value) ([]byte, error) {
	if obj.IsNil() {
		return []byte("null"), nil
	}
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

func marshalInternal(m interface{}, opts MarshalOpts) (*encoder, error) {
	enc, err := newEncoder(m, opts)
	if err != nil {
		return nil, err
	}
	ms := reflect.ValueOf(m).Elem()
	for i := 0; i < ms.NumField(); i++ {
		fldtp := ms.Type().Field(i)
		ft := getFieldTags(fldtp)
		if ft.ignore {
			continue
		}
		fld := ms.Field(i)
		for j := 0; j < fld.Len(); j++ {
			_, err = enc.marshalAny(fld.Index(j))
			if err != nil {
				return nil, err
			}
		}
	}
	return enc, nil
}

type MarshalOpts struct {
	Prefix  string
	Indent  string
	IDField string
}

func MarshalWithOpts(m interface{}, opts MarshalOpts) ([]byte, error) {
	enc, err := marshalInternal(m, opts)
	if err != nil {
		return nil, err
	}
	if opts.Prefix == "" && opts.Indent == "" {
		return enc.getJSON()
	}
	return enc.getJSONIndent(opts.Prefix, opts.Indent)
}

func Marshal(m interface{}) ([]byte, error) {
	return MarshalWithOpts(m, MarshalOpts{})
}
