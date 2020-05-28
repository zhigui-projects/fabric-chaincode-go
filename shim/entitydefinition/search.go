package entitydefinition

import (
	"bytes"
	"encoding/gob"
	"errors"
	"reflect"

	"github.com/golang/protobuf/proto"
)

var (
	PrimitiveFlag   = byte('p')
	DataTypeFlag    = byte('d')
	SliceTypeFlag   = byte('s')
	NoSliceTypeFlag = byte('n')
)

type Search struct {
	WhereConditions []map[string][][]byte
	NotConditions   []map[string][][]byte
	OrConditions    []map[string][][]byte
	OrderConditions []string
	OffsetCondition int
	LimitCondition  int
	Entity          string
}

func (s *Search) Where(query string, values ...interface{}) error {
	valueBytes, err := encodeValues(values)
	if err != nil {
		return err
	}

	s.WhereConditions = append(s.WhereConditions, map[string][][]byte{"query": {[]byte(query)}, "args": valueBytes})
	return nil
}

func (s *Search) Not(query string, values ...interface{}) error {
	valueBytes, err := encodeValues(values)
	if err != nil {
		return err
	}

	s.NotConditions = append(s.NotConditions, map[string][][]byte{"query": {[]byte(query)}, "args": valueBytes})
	return nil
}

func (s *Search) Or(query string, values ...interface{}) error {
	valueBytes, err := encodeValues(values)
	if err != nil {
		return err
	}

	s.OrConditions = append(s.OrConditions, map[string][][]byte{"query": {[]byte(query)}, "args": valueBytes})
	return nil
}

func (s *Search) Order(value string) *Search {
	if value != "" {
		s.OrderConditions = append(s.OrderConditions, value)
	}
	return s
}

func (s *Search) Limit(limit int) *Search {
	s.LimitCondition = limit
	return s
}

func (s *Search) Offset(offset int) *Search {
	s.OffsetCondition = offset
	return s
}

func encodeValues(values []interface{}) ([][]byte, error) {
	valueBytes := make([][]byte, 0)
	var err error
	for _, value := range values {
		t := reflect.TypeOf(value)
		k := t.Kind()
		var valueBuf bytes.Buffer
		e := gob.NewEncoder(&valueBuf)
		err = e.Encode(value)
		if err != nil {
			return nil, err
		}
		b := make([]byte, 0)
		if k == reflect.Uint || k == reflect.Uint8 || k == reflect.Uint16 || k == reflect.Uint32 || k == reflect.Uint64 || k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 || k == reflect.Int32 || k == reflect.Int64 || k == reflect.String || k == reflect.Bool || k == reflect.Float32 || k == reflect.Float64 {
			kindBytes := proto.EncodeVarint(uint64(k))
			b = append(b, NoSliceTypeFlag)
			b = append(b, PrimitiveFlag)
			b = append(b, kindBytes...)
			b = append(b, valueBuf.Bytes()...)
		} else if k == reflect.Struct {
			key := nameKey[t.PkgPath()+ORMDB_SEPERATOR+t.Name()]
			b = append(b, NoSliceTypeFlag)
			b = append(b, DataTypeFlag, key)
			b = append(b, valueBuf.Bytes()...)
		} else if k == reflect.Slice {
			ek := t.Elem().Kind()
			if ek == reflect.Uint || ek == reflect.Uint8 || ek == reflect.Uint16 || ek == reflect.Uint32 || ek == reflect.Uint64 || ek == reflect.Int || ek == reflect.Int8 || ek == reflect.Int16 || ek == reflect.Int32 || ek == reflect.Int64 || ek == reflect.String || ek == reflect.Bool || ek == reflect.Float32 || ek == reflect.Float64 {
				ekindBytes := proto.EncodeVarint(uint64(ek))
				b = append(b, SliceTypeFlag)
				b = append(b, PrimitiveFlag)
				b = append(b, ekindBytes...)
				b = append(b, valueBuf.Bytes()...)
			} else if ek == reflect.Struct {
				ekey := nameKey[t.Elem().PkgPath()+ORMDB_SEPERATOR+t.Elem().Name()]
				b = append(b, SliceTypeFlag)
				b = append(b, DataTypeFlag, ekey)
				b = append(b, valueBuf.Bytes()...)
			}
		} else {
			return nil, errors.New("not supported type for search value")
		}
		valueBytes = append(valueBytes, b)
	}
	return valueBytes, nil
}

func DecodeSearchValues(values [][]byte) ([]interface{}, error) {
	args := make([]interface{}, 0)
	var err error
	for _, value := range values {
		isSliceFlag := value[0]
		isPrimitiveFlag := value[1]
		if NoSliceTypeFlag == isSliceFlag {
			if PrimitiveFlag == isPrimitiveFlag {
				kind, n := proto.DecodeVarint(value[2:])
				var elemType reflect.Type
				switch reflect.Kind(kind) {
				case reflect.Uint:
					elemType = reflect.TypeOf(SampleUint)
				case reflect.Uint8:
					elemType = reflect.TypeOf(SampleUint8)
				case reflect.Uint16:
					elemType = reflect.TypeOf(SampleUint16)
				case reflect.Uint32:
					elemType = reflect.TypeOf(SampleUint32)
				case reflect.Uint64:
					elemType = reflect.TypeOf(SampleUint64)
				case reflect.Int:
					elemType = reflect.TypeOf(SampleInt)
				case reflect.Int8:
					elemType = reflect.TypeOf(SampleInt8)
				case reflect.Int16:
					elemType = reflect.TypeOf(SampleInt16)
				case reflect.Int32:
					elemType = reflect.TypeOf(SampleInt32)
				case reflect.Int64:
					elemType = reflect.TypeOf(SampleInt64)
				case reflect.String:
					elemType = reflect.TypeOf(SampleString)
				case reflect.Bool:
					elemType = reflect.TypeOf(SampleBool)
				case reflect.Float32:
					elemType = reflect.TypeOf(SampleFloat32)
				case reflect.Float64:
					elemType = reflect.TypeOf(SampleFloat64)
				default:
					return nil, errors.New("not supported data type")
				}
				elem := reflect.New(elemType).Interface()
				d := gob.NewDecoder(bytes.NewBuffer(value[2+n:]))
				err = d.Decode(elem)
				args = append(args, reflect.ValueOf(elem).Elem().Interface())
			} else if DataTypeFlag == isPrimitiveFlag {
				typeKey := value[2]
				t := datatype[typeKey]
				elem := reflect.New(t).Interface()
				d := gob.NewDecoder(bytes.NewBuffer(value[3:]))
				err = d.Decode(elem)
				args = append(args, reflect.ValueOf(elem).Elem().Interface())
			} else {
				return nil, errors.New("not supported data type")
			}
		} else if SliceTypeFlag == isSliceFlag {
			if PrimitiveFlag == isPrimitiveFlag {
				kind, n := proto.DecodeVarint(value[2:])
				var elemType reflect.Type
				switch reflect.Kind(kind) {
				case reflect.Uint:
					elemType = reflect.SliceOf(reflect.TypeOf(SampleUint))
				case reflect.Uint8:
					elemType = reflect.SliceOf(reflect.TypeOf(SampleUint8))
				case reflect.Uint16:
					elemType = reflect.SliceOf(reflect.TypeOf(SampleUint16))
				case reflect.Uint32:
					elemType = reflect.SliceOf(reflect.TypeOf(SampleUint32))
				case reflect.Uint64:
					elemType = reflect.SliceOf(reflect.TypeOf(SampleUint64))
				case reflect.Int:
					elemType = reflect.SliceOf(reflect.TypeOf(SampleInt))
				case reflect.Int8:
					elemType = reflect.SliceOf(reflect.TypeOf(SampleInt8))
				case reflect.Int16:
					elemType = reflect.SliceOf(reflect.TypeOf(SampleInt16))
				case reflect.Int32:
					elemType = reflect.SliceOf(reflect.TypeOf(SampleInt32))
				case reflect.Int64:
					elemType = reflect.SliceOf(reflect.TypeOf(SampleInt64))
				case reflect.String:
					elemType = reflect.SliceOf(reflect.TypeOf(SampleString))
				case reflect.Bool:
					elemType = reflect.SliceOf(reflect.TypeOf(SampleBool))
				case reflect.Float32:
					elemType = reflect.SliceOf(reflect.TypeOf(SampleFloat32))
				case reflect.Float64:
					elemType = reflect.SliceOf(reflect.TypeOf(SampleFloat64))
				default:
					return nil, errors.New("not supported data type")
				}
				elem := reflect.New(elemType).Interface()
				d := gob.NewDecoder(bytes.NewBuffer(value[2+n:]))
				err = d.Decode(elem)
				args = append(args, reflect.ValueOf(elem).Elem().Interface())
			} else if DataTypeFlag == isPrimitiveFlag {
				typeKey := value[2]
				t := datatype[typeKey]
				elem := reflect.New(reflect.SliceOf(t)).Interface()
				d := gob.NewDecoder(bytes.NewBuffer(value[3:]))
				err = d.Decode(elem)
				args = append(args, reflect.ValueOf(elem).Elem().Interface())
			} else {
				return nil, errors.New("not supported data type")
			}
		}
	}
	return args, err
}
