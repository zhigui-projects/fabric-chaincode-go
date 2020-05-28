package entitydefinition

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestSearch(t *testing.T) {
	ti := time.Now().UTC()
	search := &Search{}
	search.Where("id = ? AND birthday = ? AND name IN (?) ", "123", ti, []string{"123", "234"})
	search.Where("id = ? AND birthday = ? AND name IN (?) ", "123", ti, []string{"123", "234"})

	search.Or("id = ? AND birthday = ? AND name IN (?) ", "123", ti, []string{"123", "234"})
	search.Not("id = ? AND birthday = ? AND name IN (?) ", "123", ti, []string{"123", "234"})

	search.Order("id desc, name")
	search.Limit(20)
	search.Offset(10)
	var searchBytes bytes.Buffer
	e := gob.NewEncoder(&searchBytes)
	err := e.Encode(search)
	assert.NoError(t, err)

	d := gob.NewDecoder(&searchBytes)
	search1 := &Search{}
	err = d.Decode(search1)

	arg1 := search1.WhereConditions[0]["args"][0]
	assert.Equal(t, NoSliceTypeFlag, arg1[0])
	assert.Equal(t, PrimitiveFlag, arg1[1])
	kind, n := proto.DecodeVarint(arg1[2:])
	assert.Equal(t, uint64(reflect.String), kind)

	var s1 string
	d = gob.NewDecoder(bytes.NewBuffer(arg1[2+n:]))
	err = d.Decode(&s1)
	assert.NoError(t, err)
	assert.Equal(t, "123", s1)

	arg2 := search1.OrConditions[0]["args"][1]
	assert.Equal(t, NoSliceTypeFlag, arg2[0])
	assert.Equal(t, DataTypeFlag, arg2[1])
	assert.Equal(t, TimeTimeTypeKey, arg2[2])

	var s2 time.Time
	d = gob.NewDecoder(bytes.NewBuffer(arg2[3:]))
	err = d.Decode(&s2)
	assert.NoError(t, err)
	assert.Equal(t, ti, s2)

	arg3 := search1.NotConditions[0]["args"][2]
	assert.Equal(t, SliceTypeFlag, arg3[0])
	assert.Equal(t, PrimitiveFlag, arg3[1])
	kind, n = proto.DecodeVarint(arg3[2:])
	assert.Equal(t, uint64(reflect.String), kind)

	var s3 []string
	d = gob.NewDecoder(bytes.NewBuffer(arg3[2+n:]))
	err = d.Decode(&s3)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(s3))

	assert.Equal(t, "id desc, name", search1.OrderConditions[0])
	assert.Equal(t, 10, search1.OffsetCondition)
	assert.Equal(t, 20, search1.LimitCondition)

	argsDecode, err := DecodeSearchValues(search1.WhereConditions[0]["args"])
	assert.Equal(t, 3, len(argsDecode))
}
