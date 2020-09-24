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
	// TODO: Chan, Func, UnsafePointer is invalid
	return tps, nms, nil
}