package entitydefinition

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type TestSubModel struct {
	ID        uint       `gorm:"primary_key"`
	CreatedAt time.Time  `ormdb:"datatype"`
	UpdatedAt time.Time  `ormdb:"datatype"`
	DeletedAt *time.Time `sql:"index" ormdb:"datatype"`
	Name      string
}

type TestModel struct {
	Name           string
	Age            sql.NullInt64   `ormdb:"datatype"`
	Birthday       *time.Time      `ormdb:"datatype"`
	Email          string          `gorm:"type:varchar(100);unique_index"`
	Role           string          `gorm:"size:255"`        // set field size to 255
	MemberNumber   *string         `gorm:"unique;not null"` // set member number to unique and not null
	Num            int             `gorm:"AUTO_INCREMENT"`  // set num to auto incrementable
	Address        string          `gorm:"index:addr"`      // create index with name `addr` for address
	IgnoreMe       int             `gorm:"-"`               // ignore this field
	TestSubModels  []TestSubModel  `ormdb:"entity"`
	TestSubModels1 []*TestSubModel `ormdb:"entity"`
	IgnoreMe1      *int            `gorm:"-"` // ignore this field
	Attachment     []byte
}

func TestRegisterEntityAndDynamicStruct(t *testing.T) {
	// Transform model to entityFieldDefinitions
	subKey, subEntityFieldDefinitions, err := RegisterEntity(&TestSubModel{})
	assert.NoError(t, err)
	key, entityFieldDefinitions, err := RegisterEntity(&TestModel{})
	assert.NoError(t, err)

	// Marshal test for entityFieldDefinitions
	subEntityFieldDefinitionsBytes, err := json.Marshal(subEntityFieldDefinitions)
	assert.NoError(t, err)
	entityFieldDefinitionsBytes, err := json.Marshal(entityFieldDefinitions)
	assert.NoError(t, err)

	// Unmarshal test for entityFieldDefinitions
	var subEntityFieldDefinitions1 []EntityFieldDefinition
	err = json.Unmarshal(subEntityFieldDefinitionsBytes, &subEntityFieldDefinitions1)
	assert.NoError(t, err)
	for _, e := range subEntityFieldDefinitions1 {
		if e.Name == "ID" {
			assert.Equal(t, reflect.Uint, e.Kind)
		}
	}
	var entityFieldDefinitions1 []EntityFieldDefinition
	err = json.Unmarshal(entityFieldDefinitionsBytes, &entityFieldDefinitions1)
	assert.NoError(t, err)

	// Test create DynamicStruct from entityFieldDefinition
	registry := make(map[string]DynamicStruct)
	subEntityDS := NewBuilder().AddEntityFieldDefinition(subEntityFieldDefinitions1, registry).Build()
	for i := 0; i < subEntityDS.NumField(); i++ {
		field := subEntityDS.StructType().Field(i)
		if field.Name == "ID" {
			assert.Equal(t, `gorm:"primary_key"`, string(field.Tag))
		}
	}

	err = json.Unmarshal(subEntityFieldDefinitionsBytes, &subEntityFieldDefinitions1)
	assert.NoError(t, err)
	registry[subKey] = subEntityDS
	entityDS := NewBuilder().AddEntityFieldDefinition(entityFieldDefinitions1, registry).Build()
	registry[key] = entityDS

	for i := 0; i < entityDS.NumField(); i++ {
		field := entityDS.StructType().Field(i)
		if field.Name == "Age" {
			assert.Equal(t, reflect.TypeOf(sql.NullInt64{}), field.Type)
		}
	}

	now := time.Now()
	testSubModel := &TestSubModel{ID: uint(1), Name: "Test", CreatedAt: now}
	testSubModelBytes, err := json.Marshal(testSubModel)
	assert.NoError(t, err)

	testSubModel1 := subEntityDS.Interface()
	err = json.Unmarshal(testSubModelBytes, testSubModel1)
	assert.NoError(t, err)
	assert.Equal(t, "Test", reflect.ValueOf(testSubModel1).Elem().FieldByName("Name").String())
	assert.Equal(t, now.Local(), reflect.ValueOf(testSubModel1).Elem().FieldByName("CreatedAt").Interface())
	assert.Equal(t, uint64(1), reflect.ValueOf(testSubModel1).Elem().FieldByName("ID").Uint())

	testModel := &TestModel{Name: "Test", Role: "SS", Age: sql.NullInt64{Int64: int64(1)}, TestSubModels: []TestSubModel{*testSubModel}}
	testModelBytes, err := json.Marshal(testModel)
	assert.NoError(t, err)

	testModel1 := entityDS.Interface()
	err = json.Unmarshal(testModelBytes, testModel1)
	assert.NoError(t, err)
	assert.Equal(t, "Test", reflect.ValueOf(testModel1).Elem().FieldByName("Name").String())
	assert.Equal(t, "SS", reflect.ValueOf(testModel1).Elem().FieldByName("Role").String())
	assert.Equal(t, sql.NullInt64{Int64: int64(1)}, reflect.ValueOf(testModel1).Elem().FieldByName("Age").Interface())
}
