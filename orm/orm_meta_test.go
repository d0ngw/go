package orm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type SecondID struct {
	AutoID
	ID2   int64  `column:"id2"`
	Name3 string `column:"name3"`
}

func (p *SecondID) TableName() string {
	return "second_test"
}

type ThirdID struct {
	ID3 int64 `column:"id3"`
	SecondID
	Name4 string `column:"name4"`
}

type FourthID struct {
	ID4 int64 `column:"id4"`
	ThirdID
	Name5  string `column:"name5"`
	Name6  string `column:"name6"`
	Name7  string `column:"name7"`
	Name8  string `column:"name8"`
	Name9  string `column:"name9"`
	Name10 string `column:"name10"`
	Name11 string `column:"name11"`
	Conf   *Conf  `column:"conf"`
}

func TestPaseMeta(t *testing.T) {
	meta, err := parseMeta(&FourthID{})
	assert.NoError(t, err)
	assert.NotNil(t, meta)
	assert.EqualValues(t, 15, len(meta.fields))
	expectIndexs := map[string][]int{
		"id4":    {0},
		"id3":    {1, 0},
		"id":     {1, 1, 0, 0},
		"name2":  {1, 1, 0, 1},
		"id2":    {1, 1, 1},
		"name3":  {1, 1, 2},
		"name4":  {1, 2},
		"name5":  {2},
		"name6":  {3},
		"name7":  {4},
		"name8":  {5},
		"name9":  {6},
		"name10": {7},
		"name11": {8},
		"conf":   {9},
	}
	for _, field := range meta.fields {
		t.Logf("field:%v,%v", field, expectIndexs[field.column])
		assert.EqualValues(t, field.index, expectIndexs[field.column])
	}
}
