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

package engine

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/dolthub/go-mysql-server/sql"
	"github.com/dolthub/go-mysql-server/sql/types"
	"github.com/stretchr/testify/require"
)

func TestSecondsSince(t *testing.T) {
	t.Run("1 second passes", func(t *testing.T) {
		start := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
		stop := time.Date(2022, 1, 1, 0, 0, 1, 0, time.UTC)
		require.Equal(t, 1.0, secondsSince(start, stop))
	})
	t.Run("1 second and 1 millisecond passes", func(t *testing.T) {
		start := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
		stop := time.Date(2022, 1, 1, 0, 0, 1, int(1*time.Millisecond), time.UTC)
		require.Equal(t, 1.001, secondsSince(start, stop))
	})
	t.Run("1 second and 0.5 millisecond passes", func(t *testing.T) {
		start := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
		stop := time.Date(2022, 1, 1, 0, 0, 1, int(1*time.Millisecond/2), time.UTC)
		require.Equal(t, 1.000, secondsSince(start, stop))
	})
}

// lazyOkResultIter models the row iterators returned for statements that are evaluated lazily, such as
// EXECUTE of a prepared INSERT/UPDATE/DELETE: the statement's work only happens when a row is pulled.
type lazyOkResultIter struct {
	err      error
	consumed bool
	closed   bool
	depleted bool
}

var _ sql.RowIter = (*lazyOkResultIter)(nil)

func (i *lazyOkResultIter) Next(*sql.Context) (sql.Row, error) {
	if i.err != nil {
		return nil, i.err
	}
	if i.depleted {
		return nil, io.EOF
	}
	i.depleted = true
	i.consumed = true
	return sql.Row{types.NewOkResult(1)}, nil
}

func (i *lazyOkResultIter) Close(*sql.Context) error {
	i.closed = true
	return nil
}

// TestPrettyPrintResultsDrainsOkResult pins the fix for dolthub/dolt#11345. When an OK result isn't
// printed (non-TTY `dolt sql -q`/`-f`), the iterator must still be drained — otherwise a lazily
// evaluated write such as EXECUTE of a prepared UPDATE reports success and applies nothing.
func TestPrettyPrintResultsDrainsOkResult(t *testing.T) {
	t.Run("drains the iterator when the OK result is not printed", func(t *testing.T) {
		iter := &lazyOkResultIter{}

		err := PrettyPrintResults(sql.NewEmptyContext(), FormatTabular, types.OkResultSchema, iter, false, false, false, false)

		require.NoError(t, err)
		require.True(t, iter.consumed, "iterator was not drained, so the statement's writes never happened")
		require.True(t, iter.closed)
	})

	t.Run("surfaces an error raised while draining", func(t *testing.T) {
		wantErr := errors.New("duplicate primary key given: [a]")
		iter := &lazyOkResultIter{err: wantErr}

		err := PrettyPrintResults(sql.NewEmptyContext(), FormatTabular, types.OkResultSchema, iter, false, false, false, false)

		require.ErrorIs(t, err, wantErr)
		require.True(t, iter.closed)
	})
}
