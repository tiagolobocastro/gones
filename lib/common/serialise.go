package common

import (
	"encoding/gob"
	"os"
	"reflect"
)

type Serialiser interface {
	Serialise(elem ...interface{}) error
	DeSerialise(elem ...interface{}) error
}

type Serialisable interface {
	Serialise(e Serialiser) error
	DeSerialise(e Serialiser) error
}

func NewSerialiser(writer *os.File) Serialiser {
	return &gobSerialiser{
		encoder: gob.NewEncoder(writer),
		decoder: gob.NewDecoder(writer),
	}
}

type gobSerialiser struct {
	encoder *gob.Encoder
	decoder *gob.Decoder
}

func (g *gobSerialiser) Serialise(elem ...interface{}) error {
	for _, elem := range elem {
		if err := g.encode(elem); err != nil {
			return err
		}
	}
	return nil
}
func (g *gobSerialiser) encode(elem interface{}) error {
	if reflect.ValueOf(elem).Kind() == reflect.Array {
		for i := 0; i < reflect.ValueOf(elem).Len(); i++ {
			v := reflect.ValueOf(elem).Index(i)
			if v.Type().Implements(reflect.TypeOf((*Serialisable)(nil)).Elem()) {
				return v.Interface().(Serialisable).Serialise(g)
			}
		}
	} else if reflect.ValueOf(elem).Type().Implements(reflect.TypeOf((*Serialisable)(nil)).Elem()) {
		return reflect.ValueOf(elem).Interface().(Serialisable).Serialise(g)
	}
	return g.encoder.Encode(elem)
}

func (g *gobSerialiser) DeSerialise(elem ...interface{}) error {
	for _, e := range elem {
		if err := g.decode(e); err != nil {
			return err
		}
	}
	return nil
}
func (g *gobSerialiser) decode(elem interface{}) error {
	if reflect.ValueOf(elem).Kind() == reflect.Array {
		for i := 0; i < reflect.ValueOf(elem).Len(); i++ {
			v := reflect.ValueOf(elem).Index(i)
			if v.Type().Implements(reflect.TypeOf((*Serialisable)(nil)).Elem()) {
				return v.Interface().(Serialisable).DeSerialise(g)
			}
		}
	} else if reflect.ValueOf(elem).Type().Implements(reflect.TypeOf((*Serialisable)(nil)).Elem()) {
		return reflect.ValueOf(elem).Interface().(Serialisable).DeSerialise(g)
	}
	return g.decoder.Decode(elem)
}
