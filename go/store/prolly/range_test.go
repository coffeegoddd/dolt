// Copyright 2022 Dolthub, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prolly

import (
	"context"
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dolthub/dolt/go/store/val"
)

var tuples = []val.Tuple{
	intTuple(1, 1),           // 0
	intTuple(1, 2),           // 1
	intTuple(1, 3),           // 2
	intTuple(2, 1),           // 3
	intTuple(2, 2),           // 4
	intTuple(2, 3),           // 5
	intTuple(3, 1),           // 6
	intTuple(3, 2),           // 7
	intTuple(3, 3),           // 8
	intTuple(4, 1),           // 9
	intTuple(4, 2),           // 10
	intTuple(4, 3),           // 11
	intNullTuple(&nine, nil), // 12
	intNullTuple(nil, &nine), // 13
}

var nine = int32(9)

func TestRangeBounds(t *testing.T) {
	intType := val.Type{Enc: val.Int32Enc}
	twoCol := val.NewTupleDescriptor(
		intType, // c0
		intType, // c1
	)

	tests := []struct {
		name      string
		testRange Range
		inside    []val.Tuple
	}{
		{
			name: "unbound range",
			testRange: Range{
				Start: nil,
				Stop:  nil,
				Desc:  twoCol,
			},
			inside: tuples[:],
		},

		// first column ranges
		{
			name: "c0 > 1",
			testRange: Range{
				Start: []RangeCut{
					{Value: intVal(1), Type: intType, Inclusive: false},
				},
				Stop: nil,
				Desc: twoCol,
			},
			inside: tuples[3:13],
		},
		{
			name: "c0 < 1",
			testRange: Range{
				Start: nil,
				Stop: []RangeCut{
					{Value: intVal(1), Type: intType, Inclusive: false},
				},
				Desc: twoCol,
			},
			inside: nil,
		},
		{
			name: "2 <= c0 <= 3",
			testRange: Range{
				Start: []RangeCut{
					{Value: intVal(2), Type: intType, Inclusive: true},
				},
				Stop: []RangeCut{
					{Value: intVal(3), Type: intType, Inclusive: true},
				},
				Desc: twoCol,
			},
			inside: tuples[3:9],
		},
		{
			name: "c0 = NULL",
			testRange: Range{
				Start: []RangeCut{
					{Null: true, Type: intType},
				},
				Stop: []RangeCut{
					{Null: true, Type: intType},
				},
				Desc: twoCol,
			},
			inside: tuples[13:],
		},

		// second column ranges
		{
			name: "c1 > 1",
			testRange: Range{
				Start: []RangeCut{
					{Value: nil, Type: intType},
					{Value: intVal(1), Type: intType, Inclusive: false},
				},
				Stop: nil,
				Desc: twoCol,
			},
			inside: concat(tuples[1:3], tuples[4:6], tuples[7:9], tuples[10:12], tuples[13:]),
		},
		{
			name: "c1 < 1",
			testRange: Range{
				Start: nil,
				Stop: []RangeCut{
					{Value: nil, Type: intType},
					{Value: intVal(1), Type: intType, Inclusive: false},
				},
				Desc: twoCol,
			},
			inside: nil,
		},
		{
			name: "2 <= c1 <= 3",
			testRange: Range{
				Start: []RangeCut{
					{Value: nil, Type: intType},
					{Value: intVal(2), Type: intType, Inclusive: true},
				},
				Stop: []RangeCut{
					{Value: nil, Type: intType},
					{Value: intVal(3), Type: intType, Inclusive: true},
				},
				Desc: twoCol,
			},
			inside: concat(tuples[1:3], tuples[4:6], tuples[7:9], tuples[10:12]),
		},
		{
			name: "c1 = NULL",
			testRange: Range{
				Start: []RangeCut{
					{Value: nil, Type: intType},
					{Null: true, Type: intType},
				},
				Stop: []RangeCut{
					{Value: nil, Type: intType},
					{Null: true, Type: intType},
				},
				Desc: twoCol,
			},
			inside: tuples[12:13],
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rng := test.testRange
			for _, tup := range test.inside {
				inStart, inStop := rng.AboveStart(tup), rng.BelowStop(tup)
				assert.True(t, inStart && inStop,
					"%s should be in range %s \n",
					rng.Desc.Format(tup), rng.format())
			}

			count := 0
			for _, tup := range tuples {
				if rng.AboveStart(tup) && rng.BelowStop(tup) {
					count++
				}
			}
			assert.Equal(t, len(test.inside), count)
		})
	}
}

func TestRangeSearch(t *testing.T) {
	intType := val.Type{Enc: val.Int32Enc}
	twoCol := val.NewTupleDescriptor(
		intType, // c0
		intType, // c1
	)

	tests := []struct {
		name      string
		testRange Range
		hi, lo    int
	}{
		{
			name: "unbound range",
			testRange: Range{
				Start: nil,
				Stop:  nil,
				Desc:  twoCol,
			},
			lo: 0,
			hi: 14,
		},

		// first column ranges
		{
			name: "c0 > 1",
			testRange: Range{
				Start: []RangeCut{
					{Value: intVal(1), Type: intType, Inclusive: false},
				},
				Stop: nil,
				Desc: twoCol,
			},
			lo: 3,
			hi: 14,
		},
		{
			name: "c0 < 1",
			testRange: Range{
				Start: nil,
				Stop: []RangeCut{
					{Value: intVal(1), Type: intType, Inclusive: false},
				},
				Desc: twoCol,
			},
			lo: 0,
			hi: 0,
		},
		{
			name: "2 <= c0 <= 3",
			testRange: Range{
				Start: []RangeCut{
					{Value: intVal(2), Type: intType, Inclusive: true},
				},
				Stop: []RangeCut{
					{Value: intVal(3), Type: intType, Inclusive: true},
				},
				Desc: twoCol,
			},
			lo: 3,
			hi: 9,
		},
		{
			name: "c0 = NULL",
			testRange: Range{
				Start: []RangeCut{
					{Null: true, Type: intType},
				},
				Stop: []RangeCut{
					{Null: true, Type: intType},
				},
				Desc: twoCol,
			},
			lo: 13,
			hi: 14,
		},

		// second column ranges
		{
			name: "c1 > 1",
			testRange: Range{
				Start: []RangeCut{
					{Value: nil, Type: intType},
					{Value: intVal(1), Type: intType, Inclusive: false},
				},
				Stop: nil,
				Desc: twoCol,
			},
			lo: 1,
			hi: 14,
		},
		{
			name: "c1 < 1",
			testRange: Range{
				Start: nil,
				Stop: []RangeCut{
					{Value: nil, Type: intType},
					{Value: intVal(1), Type: intType, Inclusive: false},
				},
				Desc: twoCol,
			},
			lo: 0,
			hi: 0,
		},
		{
			name: "2 <= c1 <= 3",
			testRange: Range{
				Start: []RangeCut{
					{Value: nil, Type: intType},
					{Value: intVal(2), Type: intType, Inclusive: true},
				},
				Stop: []RangeCut{
					{Value: nil, Type: intType},
					{Value: intVal(3), Type: intType, Inclusive: true},
				},
				Desc: twoCol,
			},
			lo: 1,
			hi: 12,
		},
		{
			name: "c1 = NULL",
			testRange: Range{
				Start: []RangeCut{
					{Value: nil, Type: intType},
					{Null: true, Type: intType},
				},
				Stop: []RangeCut{
					{Value: nil, Type: intType},
					{Null: true, Type: intType},
				},
				Desc: twoCol,
			},
			lo: 12,
			hi: 13,
		},
	}

	values := make([]val.Tuple, len(tuples))
	for i := range values {
		values[i] = make(val.Tuple, 0)
	}
	testNode := newTupleLeafNode(tuples, values)
	testMap := Map{root: testNode, keyDesc: twoCol}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			rng := test.testRange

			startSearch := rangeStartSearchFn(rng)
			idx := startSearch(testNode)
			assert.Equal(t, test.lo, idx, "range should start at index %d", test.lo)

			stopSearch := rangeStopSearchFn(rng)
			idx = stopSearch(testNode)
			assert.Equal(t, test.hi, idx, "range should stop before index %d", test.hi)

			iter, err := testMap.IterRange(ctx, rng)
			require.NoError(t, err)
			expected := tuples[test.lo:test.hi]

			i := 0
			for {
				tup, _, err := iter.Next(ctx)
				if err == io.EOF {
					break
				}
				require.NoError(t, err)
				require.True(t, i < len(expected))
				assert.Equal(t, expected[i], tup)
				i++
			}
		})
	}
}

func intVal(i int32) (buf []byte) {
	buf = make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(i))
	return
}

func intNullTuple(ints ...*int32) val.Tuple {
	types := make([]val.Type, len(ints))
	for i := range types {
		types[i] = val.Type{
			Enc:      val.Int32Enc,
			Nullable: true,
		}
	}

	desc := val.NewTupleDescriptor(types...)
	tb := val.NewTupleBuilder(desc)
	for i, val := range ints {
		if val != nil {
			tb.PutInt32(i, *val)
		}
	}
	return tb.Build(sharedPool)
}
