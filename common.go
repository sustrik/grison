package grison

import (
	"fmt"
	"reflect"
)

func scrapeMasterStruct(m interface{}) (map[reflect.Type]string, map[string]reflect.Type, error) {
	tps := make(map[reflect.Type]string)
	nms := make(map[string]reflect.Type)
	tp := reflect.TypeOf(m)
	if tp == nil {
		return nil, nil, fmt.Errorf("master structure is nil")
	}
	if tp.Kind() != reflect.Ptr {
		return nil, nil, fmt.Errorf("master structure must be passed as a pointer, it is %T", m)
	}
	tp = tp.Elem()
	if tp.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("master structure is not a structure, it is %T", m)
	}
	for i := 0; i < tp.NumField(); i++ {
		fldtp := tp.Field(i).Type
		fldname := tp.Field(i).Name
		if fldtp.Kind() != reflect.Slice && fldtp.Kind() != reflect.Map {
			return nil, nil, fmt.Errorf("master field %s is not a map or slice, it is %v", fldname, fldtp)
		}
		fldtp = fldtp.Elem()
		if fldtp.Kind() != reflect.Ptr {
			return nil, nil, fmt.Errorf("master field %s doesn't contain pointers", fldname)
		}
		fldtp = fldtp.Elem()
		if fldtp.Kind() != reflect.Struct {
			return nil, nil, fmt.Errorf("master field %s doesn't contain pointers to structs", fldname)
		}
		// TODO: Check for duplicate types.
		tps[fldtp] = fldname
		nms[fldname] = fldtp
	}
	// TODO: There should be no embedded node instances.
	return tps, nms, nil
}
