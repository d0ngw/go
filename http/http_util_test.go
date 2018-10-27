package http

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

//PageParam 分野参数
type PageParam struct {
	Page     int
	PageSize int
}

type params struct {
	PageParam
	ID             int64
	Name           string
	Weight         float32
	Height         float32 `pname:"h"`
	Ok             bool
	Children       []string `pname:"_"`
	Ages           []int32  `psep:","`
	FriendsNames   []string
	FriendsBooks   []int64   `psep:","`
	FriendsWeights []float32 `psep:","`
}

func TestParseParams(t *testing.T) {
	form := url.Values{}
	form.Set("id", "10")
	form.Set("name", "golang")
	form.Set("weight", "1.230")
	form.Set("h", "1.01")
	form.Set("ok", "true")
	form.Set("ages", "1,2,3")
	form["friends_names"] = []string{"tom", "jerry"}
	form.Set("friends_books", "1,2")
	form.Set("friends_weights", "0.1,0,-0.3")
	form.Set("page", "1")
	form.Set("page_size", "5")

	p := &params{}
	err := ParseParams(form, p)
	assert.Nil(t, err)
	fmt.Printf("%#v\n", p)
	assert.EqualValues(t, 10, p.ID)
	assert.EqualValues(t, "golang", p.Name)
	assert.EqualValues(t, 1.23, p.Weight)
	assert.EqualValues(t, 1.01, p.Height)
	assert.EqualValues(t, []int32{1, 2, 3}, p.Ages)
	assert.EqualValues(t, []string{"tom", "jerry"}, p.FriendsNames)
	assert.EqualValues(t, []float32{0.1, 0.0, -0.3}, p.FriendsWeights)
	assert.EqualValues(t, 1, p.Page)
	assert.EqualValues(t, 5, p.PageSize)
}
