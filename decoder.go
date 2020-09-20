package grison

import (
	"encoding/json"
	"fmt"
)

type Decoder struct {
	objects map[string]map[string]interface{}
}

func NewDecoder(m interface{}) (*Decoder, error) {
	return &Decoder{
		objects: make(map[string]map[string]interface{}),
	}, nil
}

func (dec *Decoder) unmarshalNode(tp string, id string, b []byte) (interface{}, error) {
	dec.objects[tp][id] = nil
	return nil, nil
}

func Unmarshal(data []byte, m interface{}) error {
	dec, err := NewDecoder(m)
	if err != nil {
		return err
	}
	var objs map[string]map[string]json.RawMessage
	if err := json.Unmarshal(data, &objs); err != nil {
		return err
	}
	fmt.Printf("%v\n", objs)
	for tp, nodes := range objs {
		for id, node := range nodes {
			_, err := dec.unmarshalNode(tp, id, node)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
