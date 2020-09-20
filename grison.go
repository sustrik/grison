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
		types:   make(map[reflect.Type]string),
		objects: make(map[string]map[string]json.RawMessage),
		ids:     make(map[interface{}]string),
	}
	tp := reflect.TypeOf(m)
	if tp.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("master structure must be passed as a pointer, is %T", m)
	}
	tp = tp.Elem()
	if tp.Kind() != reflect.Struct {
		return nil, fmt.Errorf("master structure is not a structure, is %T", m)
	}
	for i := 0; i < tp.NumField(); i++ {
		fldtp := tp.Field(i).Type
		fldname := tp.Field(i).Name
		if fldtp.Kind() != reflect.Slice && fldtp.Kind() != reflect.Map {
			return nil, fmt.Errorf("master field %s, in not a map or a slice", fldname)
		}
		fldtp = fldtp.Elem()
		if fldtp.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("master field %s doesn't contain pointers", fldname)
		}
		fldtp = fldtp.Elem()
		if fldtp.Kind() != reflect.Struct {
			return nil, fmt.Errorf("master field %s doesn't contain pointers to structs", fldname)
		}
		// TODO: Check for duplicate types.
		enc.types[fldtp] = fldname
		enc.objects[fldname] = make(map[string]json.RawMessage)
	}
	// TODO: There should be no embedded node instances.
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
		if enc.isNodeType(obj.Elem().Type()) {
			return enc.marshalNode(obj)
		}
		return enc.marshalAny(obj.Elem())
	case reflect.Interface:
		panic("interface")
	case reflect.Struct:
		return enc.marshalStruct(obj)
	case reflect.Slice:
		return enc.marshalSlice(obj)
	case reflect.Map:
		return enc.marshalMap(obj)
	default:
		r, err := json.Marshal(obj.Interface())
		if err != nil {
			return []byte{}, err
		}
		if len(r) > 0 && string(r[0:2]) == "\"^" {
			r = append([]byte("\"^"), r[1:]...)
		}
		return r, nil
	}
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
	ref := fmt.Sprintf("^%s:%s", enc.types[reflect.TypeOf(eobj)], id)
	return json.Marshal(ref)
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

func (enc *Encoder) marshalSlice(obj reflect.Value) ([]byte, error) {
	if obj.Type() == reflect.TypeOf([]byte{}) {
		return json.Marshal(obj.Interface())
	}
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
