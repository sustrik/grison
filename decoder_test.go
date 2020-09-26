package grison

import (
	"reflect"
	"testing"

	"github.com/go-test/deep"
)

func UnmarshalTestRaw(t *testing.T, b string, v interface{}) {
	type emptyMaster struct{}
	dec, err := newDecoder(&emptyMaster{})
	if err != nil {
		t.Errorf("can't create decoder: %v", err)
		return
	}
	v2 := reflect.New(reflect.TypeOf(v))
	err = dec.unmarshalAny([]byte(b), v2)
	if err != nil {
		t.Errorf("decoding error encountered: %v", err)
		return
	}
	if diff := deep.Equal(v, v2.Elem().Interface()); diff != nil {
		t.Errorf("unexpected unmarshal result.\n%v", diff)
	}
}

func UnmarshalTest(t *testing.T, b string, v interface{}) {
	v2 := reflect.New(reflect.TypeOf(v))
	err := Unmarshal([]byte(b), v2)
	if err != nil {
		t.Errorf("decoding error encountered: %v", err)
		return
	}
	if diff := deep.Equal(v, v2.Elem().Interface()); diff != nil {
		t.Errorf("unexpected unmarshal result.\n%v", diff)
	}
}

func TestDecodeBasicTypes(t *testing.T) {
	UnmarshalTestRaw(t, `543`, 543)
	UnmarshalTestRaw(t, `-32`, -32)
	UnmarshalTestRaw(t, `3.14`, 3.14)
	UnmarshalTestRaw(t, `"foo"`, "foo")
}

func TestDecodeSlice(t *testing.T) {
	UnmarshalTestRaw(t, `[1,2,3]`, []int{1, 2, 3})
	UnmarshalTestRaw(t, `["foo","bar"]`, []string{"foo", "bar"})
}

func TestDecodeMap(t *testing.T) {
	UnmarshalTestRaw(t, `{"a":1,"b":2}`, map[string]int{"a": 1, "b": 2})
}

func TestDecodeStruct(t *testing.T) {
	type Foo struct {
		A int
		B string
	}
	UnmarshalTestRaw(t, `{"A":1,"B":"bar"}`, Foo{A: 1, B: "bar"})
}

func TestDecodePtr(t *testing.T) {
	var i int = 10
	pi := &i
	UnmarshalTestRaw(t, `10`, pi)
	ppi := &pi
	UnmarshalTestRaw(t, `10`, ppi)
}
