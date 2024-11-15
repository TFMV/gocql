//go:build all || unit
// +build all unit

package serialization_test

import (
	"gopkg.in/inf.v0"
	"math"
	"testing"

	"github.com/gocql/gocql"
	"github.com/gocql/gocql/internal/tests/serialization"
	"github.com/gocql/gocql/internal/tests/serialization/mod"
)

func TestMarshalDecimal(t *testing.T) {
	tType := gocql.NewNativeType(4, gocql.TypeDecimal, "")

	marshal := func(i interface{}) ([]byte, error) { return gocql.Marshal(tType, i) }
	unmarshal := func(bytes []byte, i interface{}) error {
		return gocql.Unmarshal(tType, bytes, i)
	}

	// Unmarshal does not support deserialization of `decimal` with `nil` and `zero` `value len` 'into `inf.Dec`.
	brokenUnmarshalTypes := serialization.GetTypes(inf.Dec{}, (*inf.Dec)(nil))

	serialization.PositiveSet{
		Data:   nil,
		Values: mod.Values{(*inf.Dec)(nil)},
	}.Run("[nil]nullable", t, marshal, unmarshal)

	serialization.PositiveSet{
		Data:                 nil,
		Values:               mod.Values{inf.Dec{}},
		BrokenUnmarshalTypes: brokenUnmarshalTypes,
	}.Run("[nil]unmarshal", t, nil, unmarshal)

	serialization.PositiveSet{
		Data:                 make([]byte, 0),
		Values:               mod.Values{*inf.NewDec(0, 0)}.AddVariants(mod.Reference),
		BrokenUnmarshalTypes: brokenUnmarshalTypes,
	}.Run("[]unmarshal", t, nil, unmarshal)

	serialization.PositiveSet{
		Data:   []byte("\x00\x00\x00\x00\x00"),
		Values: mod.Values{*inf.NewDec(0, 0)}.AddVariants(mod.Reference),
	}.Run("zeros", t, marshal, unmarshal)

	serialization.PositiveSet{
		Data:   []byte("\x7f\xff\xff\xff\x7f\xff\xff\xff\xff\xff\xff\xff"),
		Values: mod.Values{*inf.NewDec(int64(math.MaxInt64), inf.Scale(int32(math.MaxInt32)))}.AddVariants(mod.Reference),
	}.Run("max_ints", t, marshal, unmarshal)

	serialization.PositiveSet{
		Data:   []byte("\x80\x00\x00\x00\x80\x00\x00\x00\x00\x00\x00\x00"),
		Values: mod.Values{*inf.NewDec(int64(math.MinInt64), inf.Scale(int32(math.MinInt32)))}.AddVariants(mod.Reference),
	}.Run("min_ints", t, marshal, unmarshal)
}
