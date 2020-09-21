package grison

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

type Decoder struct {
	// Node types (the structs, not the pointers).
	types map[reflect.Type]string
}

func NewDecoder(m interface{}) (*Decoder, error) {
	tps, _, err := scrapeMasterStruct(m)
	if err != nil {
		return nil, err
	}
	return &Decoder{
		types: tps,
	}, nil
}

func (dec *Decoder) unmarshalPtr(b []byte, v reflect.Value) error {
	// TODO: Handle nodes
	if string(b) != "null" {
		p := reflect.New(v.Type().Elem().Elem())
		err := dec.unmarshalAny(b, p)
		if err != nil {
			return err
		}
		v.Elem().Set(p)
	}
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
	if v.Type().Elem() == reflect.TypeOf([]byte{}) {
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
	case reflect.Ptr:
		return dec.unmarshalPtr(b, v)
	case reflect.Struct:
		return dec.unmarshalStruct(b, v)
	case reflect.Map:
		return dec.unmarshalMap(b, v)
	case reflect.Slice:
		return dec.unmarshalSlice(b, v)
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
	mv := reflect.ValueOf(m).Elem()
	for tp, rms := range rmm {
		fld := mv.FieldByName(tp)
		if !fld.IsValid() {
			return fmt.Errorf("unknown node type %s", tp)
		}
		for id, rm := range rms {
			fmt.Printf("%s:%s\n", tp, id)
			v := reflect.New(fld.Type().Elem())
			err = dec.unmarshalAny(rm, v)
			if err != nil {
				return err
			}
			a := reflect.Append(fld, v.Elem())
			fld.Set(a)
		}
	}
	return nil
}
