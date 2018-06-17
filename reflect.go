// Copyright 2018 Denis Bernard <db047h@gmail.com>
// Licensed under the MIT license. See license text in the LICENSE file.

package hwsim

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Updater is the interface that custom components built using reflection must implement.
// See MakePart.
//
type Updater interface {
	Update(*Circuit)
}

var updaterType = reflect.TypeOf((*Updater)(nil)).Elem()

// MakePart wraps an Updater into a custom component.
// Input/output pins are identified by field tags.
//
// The field tag must be `hw:"in"`` or `hw:"out"` to identify input and output
// pins. By default, the pin name is the field name in lowercase. A specific
// field name can be forced by adding it in the tag: `hw:"in,pin_name"`.
//
// Buses must be arrays of int.
//
func MakePart(t Updater) *PartSpec {
	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if k := typ.Kind(); k != reflect.Struct {
		panic(errors.Errorf("unsupported type %q for %q", k, typ.Name()))
	}

	sp := &PartSpec{
		Name: typ.Name(),
	}

	n := typ.NumField()
	for i := 0; i < n; i++ {
		var isInput bool
		f := typ.Field(i)
		pin := strings.ToLower(f.Name)
		tag, ok := f.Tag.Lookup("hw")
		if !ok {
			continue
		}
		tv := strings.Split(tag, ",")
		switch len(tv) {
		case 0:
			continue
		case 2:
			if tv[1] != "" {
				pin = tv[1]
			}
			fallthrough
		case 1:
			switch tv[0] {
			case "in":
				isInput = true
			case "out":
			default:
				panic(errors.Errorf("unsupported tag %q for field %q in %q", tag, f.Name, typ.Name()))
			}
		}

		ft := f.Type
		if k := ft.Kind(); k == reflect.Array && ft.Elem().Kind() == reflect.Int {
			// bus
			for i := 0; i < ft.Len(); i++ {
				if isInput {
					sp.Inputs = append(sp.Inputs, pin+"["+strconv.Itoa(i)+"]")
				} else {
					sp.Outputs = append(sp.Outputs, pin+"["+strconv.Itoa(i)+"]")
				}
			}
		} else if k == reflect.Int {
			// pin
			if isInput {
				sp.Inputs = append(sp.Inputs, pin)
			} else {
				sp.Outputs = append(sp.Outputs, pin)
			}
		} else {
			panic(errors.Errorf("unsupported type %q for field %q in %q", k, f.Name, typ.Name()))
		}
	}
	sp.Mount = mountPart(typ)
	return sp
}

func mountPart(typ reflect.Type) func(s *Socket) []Updater {
	return func(s *Socket) []Updater {
		v := reflect.New(typ)
		e := v.Elem()
		n := typ.NumField()
		for i := 0; i < n; i++ {
			f := typ.Field(i)
			pin := strings.ToLower(f.Name)
			tag, ok := f.Tag.Lookup("hw")
			if !ok {
				continue
			}
			tv := strings.Split(tag, ",")
			switch len(tv) {
			case 0:
				continue
			case 2:
				if tv[1] != "" {
					pin = tv[1]
				}
			}
			fv := e.FieldByName(f.Name)
			if !fv.IsValid() {
				continue
			}
			ft := f.Type
			if k := ft.Kind(); k == reflect.Array && ft.Elem().Kind() == reflect.Int {
				// bus
				for i := 0; i < fv.Len(); i++ {
					fv.Index(i).SetInt(int64(s.Pin(pin + "[" + strconv.Itoa(i) + "]")))
				}
			} else if k == reflect.Int {
				// pin
				fv.SetInt(int64(s.Pin(pin)))
			} else {
				panic(errors.Errorf("unsupported type %q for field %q in %q", k, f.Name, typ.Name()))
			}
		}

		comp := v.Interface().(Updater)
		return []Updater{comp}
	}
}
