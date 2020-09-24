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
	"errors"
	"fmt"
	"reflect"
)

type Decoder struct {
	// Node types (the structs, not the pointers).
	types  map[reflect.Type]string
	names  map[string]reflect.Type
	master reflect.Value
	refmap map[string]reflect.Value
}

func NewDecoder(m interface{}) (*Decoder, error) {
	tps, nms, err := scrapeMasterStruct(m)
	if err != nil {
		return nil, err
	}
	return &Decoder{
		types:  tps,
		names:  nms,
		master: reflect.ValueOf(m).Elem(),
		refmap: make(map[string]reflect.Value),
	}, nil
}

func (dec *Decoder) unmarshalPtr(b []byte, v reflect.Value) error {
	if string(b) == "null" {
		return nil
	}
	_, ok := dec.types[v.Type().Elem().Elem()]
	if ok {
		return dec.unmarshalRef(b, v)
	}
	p := reflect.New(v.Type().Elem().Elem())
	err := dec.unmarshalAny(b, p)
	if err != nil {
		return err
	}
	v.Elem().Set(p)
	return nil
}

func (dec *Decoder) unmarshalInterface(b []byte, v reflect.Value) error {
	if string(b) == "null" {
		return nil
	}
	return dec.unmarshalRef(b, v)
}

func (dec *Decoder) unmarshalRef(b []byte, v reflect.Value) error {
	var ref string
	err := json.Unmarshal(b, &ref)
	if err != nil {
		return err
	}
	obj, ok := dec.refmap[ref]
	if !ok {
		return fmt.Errorf("invalid reference %s", ref)
	}
	v.Elem().Set(obj)
	return nil
}

func (dec *Decoder) unmarshalStruct(b []byte, v reflect.Value) error {
	var rmm map[string]json.RawMessage
	err := json.Unmarshal(b, &rmm)
	if err != nil {
		return err
	}
	for k, rm := range rmm {
		fld := v.Elem().FieldByName(k)
		if !fld.IsValid() {
			return fmt.Errorf("unknown field %s", k)
		}
		v := reflect.New(fld.Type())
		err = dec.unmarshalAny(rm, v)
		if err != nil {
			return err
		}
		fld.Set(v.Elem())
	}
	return nil
}

func (dec *Decoder) unmarshalMap(b []byte, v reflect.Value) error {
	var rmm map[string]json.RawMessage
	err := json.Unmarshal(b, &rmm)
	if err != nil {
		return err
	}
	m := reflect.MakeMap(v.Type().Elem())
	for k, rm := range rmm {
		v := reflect.New(m.Type().Elem())
		err = dec.unmarshalAny(rm, v)
		if err != nil {
			return err
		}
		m.SetMapIndex(reflect.ValueOf(k), v.Elem())
	}
	v.Elem().Set(m)
	return nil
}

func (dec *Decoder) unmarshalSlice(b []byte, v reflect.Value) error {
	if v.Type().Elem().Elem() == reflect.TypeOf(byte(0)) {
		return json.Unmarshal(b, v.Interface())
	}
	var rms []json.RawMessage
	err := json.Unmarshal(b, &rms)
	if err != nil {
		return err
	}
	s := reflect.MakeSlice(v.Type().Elem(), len(rms), len(rms))
	for i, rm := range rms {
		err = dec.unmarshalAny(rm, s.Index(i).Addr())
		if err != nil {
			return err
		}
	}
	v.Elem().Set(s)
	return nil
}

func (dec *Decoder) unmarshalArray(b []byte, v reflect.Value) error {
	var rms []json.RawMessage
	err := json.Unmarshal(b, &rms)
	if err != nil {
		return err
	}
	for i, rm := range rms {
		err = dec.unmarshalAny(rm, v.Elem().Index(i).Addr())
		if err != nil {
			return err
		}
	}
	return nil
}

func (dec *Decoder) unmarshalString(b []byte, v reflect.Value) error {
	err := json.Unmarshal(b, v.Interface())
	if err != nil {
		return err
	}
	s := v.Elem().String()
	if s[0:1] == "^" {
		if s[1:2] == "^" {
			v.Elem().Set(reflect.ValueOf(s[1:len(s)]))
		} else {
			// TODO: panic?
			return errors.New("reference used as a string")
		}
	}
	return nil
}

func (dec *Decoder) unmarshalAny(b []byte, v reflect.Value) error {
	switch v.Elem().Kind() {
	// TODO: Array
	case reflect.Ptr:
		return dec.unmarshalPtr(b, v)
	case reflect.Interface:
		return dec.unmarshalInterface(b, v)
	case reflect.Struct:
		return dec.unmarshalStruct(b, v)
	case reflect.Map:
		return dec.unmarshalMap(b, v)
	case reflect.Slice:
		return dec.unmarshalSlice(b, v)
	case reflect.Array:
		return dec.unmarshalArray(b, v)
	case reflect.String:
		return dec.unmarshalString(b, v)
	default:
		return json.Unmarshal(b, v.Interface())
	}
}

func Unmarshal(b []byte, m interface{}) error {
	dec, err := NewDecoder(m)
	if err != nil {
		return err
	}
	var rmm map[string]map[string]json.RawMessage
	err = json.Unmarshal(b, &rmm)
	if err != nil {
		return err
	}
	// Create empty shells of individual objects so that we
	// can create pointers to them.
	mv := reflect.ValueOf(m).Elem()
	for tp, rms := range rmm {
		fld := mv.FieldByName(tp)
		if !fld.IsValid() {
			return fmt.Errorf("unknown node type %s", tp)
		}
		// TODO: Deterministic order!
		for id, _ := range rms {
			v := reflect.New(fld.Type().Elem().Elem())
			a := reflect.Append(fld, v)
			fld.Set(a)
			ref := fmt.Sprintf("^%s:%s", tp, id)
			dec.refmap[ref] = v
		}
	}
	// Now we can unmarshal individual nodes.
	for tp, rms := range rmm {
		for id, rm := range rms {
			ref := fmt.Sprintf("^%s:%s", tp, id)
			err = dec.unmarshalAny(rm, dec.refmap[ref])
			if err != nil {
				return err
			}
		}
	}
	return nil
}
