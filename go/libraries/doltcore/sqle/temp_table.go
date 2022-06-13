// Copyright 2019 Dolthub, Inc.
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

package sqle

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"

	"github.com/dolthub/go-mysql-server/sql"

	"github.com/dolthub/dolt/go/libraries/doltcore/doltdb"
	"github.com/dolthub/dolt/go/libraries/doltcore/doltdb/durable"
	"github.com/dolthub/dolt/go/libraries/doltcore/schema"
	"github.com/dolthub/dolt/go/libraries/doltcore/sqle/dsess"
	"github.com/dolthub/dolt/go/libraries/doltcore/sqle/globalstate"
	"github.com/dolthub/dolt/go/libraries/doltcore/sqle/index"
	"github.com/dolthub/dolt/go/libraries/doltcore/sqle/sqlutil"
	"github.com/dolthub/dolt/go/libraries/doltcore/sqle/writer"
	"github.com/dolthub/dolt/go/libraries/doltcore/table/editor"
	"github.com/dolthub/dolt/go/libraries/doltcore/table/editor/creation"
	"github.com/dolthub/dolt/go/store/types"
)

type TempTable struct {
	tableName string
	dbName    string
	pkSch     sql.PrimaryKeySchema

	table *doltdb.Table
	sch   schema.Schema

	lookup sql.IndexLookup

	ed   writer.TableWriter
	opts editor.Options
}

var _ sql.TemporaryTable = &TempTable{}
var _ sql.Table = &TempTable{}
var _ sql.PrimaryKeyTable = &TempTable{}
var _ sql.IndexedTable = &TempTable{}
var _ sql.IndexAlterableTable = &TempTable{}
var _ sql.ForeignKeyTable = &TempTable{}
var _ sql.CheckTable = &TempTable{}
var _ sql.CheckAlterableTable = &TempTable{}
var _ sql.StatisticsTable = &TempTable{}

func NewTempTable(
	ctx *sql.Context,
	ddb *doltdb.DoltDB,
	pkSch sql.PrimaryKeySchema,
	name, db string,
	opts editor.Options,
) (*TempTable, error) {
	sess := dsess.DSessFromSess(ctx.Session)

	dbState, ok, err := sess.LookupDbState(ctx, db)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, fmt.Errorf("database %s not found in session", db)
	}

	ws := dbState.WorkingSet

	sch, err := temporaryDoltSchema(ctx, pkSch)
	if err != nil {
		return nil, err
	}
	vrw := ddb.ValueReadWriter()

	idx, err := durable.NewEmptyIndex(ctx, vrw, sch)
	if err != nil {
		return nil, err
	}
	set := durable.NewIndexSet(ctx, vrw)

	tbl, err := doltdb.NewTable(ctx, ddb.ValueReadWriter(), sch, idx, set, nil)
	if err != nil {
		return nil, err
	}

	newRoot, err := ws.WorkingRoot().PutTable(ctx, name, tbl)
	if err != nil {
		return nil, err
	}

	newWs := ws.WithWorkingRoot(newRoot)

	gs := globalstate.NewGlobalStateStore()
	ait, err := gs.GetAutoIncrementTracker(ctx, newWs)
	if err != nil {
		return nil, err
	}

	writeSession := writer.NewWriteSession(tbl.Format(), newWs, ait, opts)

	tempTable := &TempTable{
		tableName: name,
		dbName:    db,
		pkSch:     pkSch,
		table:     tbl,
		sch:       sch,
		opts:      opts,
	}

	tempTable.ed, err = writeSession.GetTableWriter(ctx, name, db, setTempTableRoot(tempTable), false)
	if err != nil {
		return nil, err
	}

	return tempTable, nil
}

func setTempTableRoot(t *TempTable) func(ctx *sql.Context, dbName string, newRoot *doltdb.RootValue) error {
	return func(ctx *sql.Context, dbName string, newRoot *doltdb.RootValue) error {
		newTable, _, err := newRoot.GetTable(ctx, t.tableName)
		if err != nil {
			return err
		}

		t.table = newTable

		sess := dsess.DSessFromSess(ctx.Session)

		dbState, ok, err := sess.LookupDbState(ctx, t.dbName)
		if err != nil {
			return err
		}

		if !ok {
			return fmt.Errorf("database %s not found in session", t.dbName)
		}

		ws := dbState.WorkingSet
		newWs := ws.WithWorkingRoot(newRoot)

		gs := globalstate.NewGlobalStateStore()
		ait, err := gs.GetAutoIncrementTracker(ctx, newWs)
		if err != nil {
			return err
		}

		writeSession := writer.NewWriteSession(newTable.Format(), newWs, ait, t.opts)
		t.ed, err = writeSession.GetTableWriter(ctx, t.tableName, t.dbName, setTempTableRoot(t), false)
		if err != nil {
			return err
		}

		return nil
	}
}

func (t *TempTable) GetIndexes(ctx *sql.Context) ([]sql.Index, error) {
	return index.DoltIndexesFromTable(ctx, t.dbName, t.tableName, t.table)
}

func (t *TempTable) Name() string {
	return t.tableName
}

func (t *TempTable) String() string {
	return t.tableName
}

func (t *TempTable) NumRows(ctx *sql.Context) (uint64, error) {
	m, err := t.table.GetRowData(ctx)
	if err != nil {
		return 0, err
	}
	return m.Count(), nil
}

func (t *TempTable) Format() *types.NomsBinFormat {
	return t.table.Format()
}

func (t *TempTable) Schema() sql.Schema {
	return t.pkSch.Schema
}

func (t *TempTable) sqlSchema() sql.PrimaryKeySchema {
	return t.pkSch
}

func (t *TempTable) Partitions(ctx *sql.Context) (sql.PartitionIter, error) {
	rows, err := t.table.GetRowData(ctx)
	if err != nil {
		return nil, err
	}
	return newDoltTablePartitionIter(rows, partitionsFromRows(ctx, rows)...), nil
}

func (t *TempTable) IsTemporary() bool {
	return true
}

func (t *TempTable) DataLength(ctx *sql.Context) (uint64, error) {
	idx, err := t.table.GetRowData(ctx)
	if err != nil {
		return 0, err
	}
	return idx.Count(), nil
}

func (t *TempTable) CalculateStatistics(ctx *sql.Context) error {
	return nil
}

func (t *TempTable) PartitionRows(ctx *sql.Context, partition sql.Partition) (sql.RowIter, error) {
	if t.lookup != nil {
		return index.RowIterForIndexLookup(ctx, t.table, t.lookup, t.pkSch, nil)
	} else {
		return partitionRows(ctx, t.table, t.sqlSchema().Schema, nil, partition)
	}
}

func (t *TempTable) WithIndexLookup(lookup sql.IndexLookup) sql.Table {
	t.lookup = lookup
	return t
}

func (t *TempTable) CreateIndex(ctx *sql.Context, indexName string, using sql.IndexUsing, constraint sql.IndexConstraint, columns []sql.IndexColumn, comment string) error {
	if constraint != sql.IndexConstraint_None && constraint != sql.IndexConstraint_Unique {
		return fmt.Errorf("only the following types of index constraints are supported: none, unique")
	}
	cols := make([]string, len(columns))
	for i, c := range columns {
		cols[i] = c.Name
	}

	ret, err := creation.CreateIndex(
		ctx,
		t.table,
		indexName,
		cols,
		constraint == sql.IndexConstraint_Unique,
		true,
		comment,
		t.opts,
	)
	if err != nil {
		return err
	}

	t.table = ret.NewTable
	return nil
}

func (t *TempTable) DropIndex(ctx *sql.Context, indexName string) error {
	_, err := t.sch.Indexes().RemoveIndex(indexName)
	if err != nil {
		return err
	}

	newTable, err := t.table.UpdateSchema(ctx, t.sch)
	if err != nil {
		return err
	}
	newTable, err = newTable.DeleteIndexRowData(ctx, indexName)
	if err != nil {
		return err
	}
	t.table = newTable

	return nil
}

func (t *TempTable) RenameIndex(ctx *sql.Context, fromIndexName string, toIndexName string) error {
	_, err := t.sch.Indexes().RenameIndex(fromIndexName, toIndexName)
	if err != nil {
		return err
	}

	newTable, err := t.table.UpdateSchema(ctx, t.sch)
	if err != nil {
		return err
	}
	newTable, err = newTable.RenameIndexRowData(ctx, fromIndexName, toIndexName)
	if err != nil {
		return err
	}
	t.table = newTable

	return nil
}

func (t *TempTable) GetDeclaredForeignKeys(ctx *sql.Context) ([]sql.ForeignKeyConstraint, error) {
	return nil, nil
}

func (t *TempTable) GetReferencedForeignKeys(ctx *sql.Context) ([]sql.ForeignKeyConstraint, error) {
	return nil, nil
}

func (t *TempTable) CreateIndexForForeignKey(ctx *sql.Context, indexName string, using sql.IndexUsing, constraint sql.IndexConstraint, columns []sql.IndexColumn) error {
	return sql.ErrTemporaryTablesForeignKeySupport.New()
}

func (t *TempTable) AddForeignKey(ctx *sql.Context, fk sql.ForeignKeyConstraint) error {
	return sql.ErrTemporaryTablesForeignKeySupport.New()
}

func (t *TempTable) UpdateForeignKey(ctx *sql.Context, fkName string, fk sql.ForeignKeyConstraint) error {
	return sql.ErrTemporaryTablesForeignKeySupport.New()
}

func (t *TempTable) DropForeignKey(ctx *sql.Context, fkName string) error {
	return sql.ErrTemporaryTablesForeignKeySupport.New()
}

func (t *TempTable) GetForeignKeyUpdater(ctx *sql.Context) sql.ForeignKeyUpdater {
	return nil
}

func (t *TempTable) Inserter(*sql.Context) sql.RowInserter {
	return t
}

func (t *TempTable) Deleter(*sql.Context) sql.RowDeleter {
	return t
}

func (t *TempTable) Replacer(*sql.Context) sql.RowReplacer {
	return t
}

func (t *TempTable) Updater(*sql.Context) sql.RowUpdater {
	return t
}

func (t *TempTable) GetChecks(*sql.Context) ([]sql.CheckDefinition, error) {
	return checksInSchema(t.sch), nil
}

func (t *TempTable) PrimaryKeySchema() sql.PrimaryKeySchema {
	return t.pkSch
}

func (t *TempTable) CreateCheck(ctx *sql.Context, check *sql.CheckDefinition) error {
	sch, err := t.table.GetSchema(ctx)
	if err != nil {
		return err
	}

	check = &(*check)
	if check.Name == "" {
		check.Name = strconv.Itoa(rand.Int())
	}

	_, err = sch.Checks().AddCheck(check.Name, check.CheckExpression, check.Enforced)
	if err != nil {
		return err
	}
	t.table, err = t.table.UpdateSchema(ctx, sch)

	return err
}

func (t *TempTable) DropCheck(ctx *sql.Context, chName string) error {
	err := t.sch.Checks().DropCheck(chName)
	if err != nil {
		return err
	}
	t.table, err = t.table.UpdateSchema(ctx, t.sch)

	return err
}

func (t *TempTable) Insert(ctx *sql.Context, sqlRow sql.Row) error {
	return t.ed.Insert(ctx, sqlRow)
}

func (t *TempTable) Update(ctx *sql.Context, oldRow sql.Row, newRow sql.Row) error {
	return t.ed.Update(ctx, oldRow, newRow)
}

func (t *TempTable) Delete(ctx *sql.Context, sqlRow sql.Row) error {
	return t.ed.Delete(ctx, sqlRow)
}

func (t *TempTable) StatementBegin(ctx *sql.Context) {
	return
}

func (t *TempTable) DiscardChanges(ctx *sql.Context, errorEncountered error) error {
	t.lookup = nil
	return nil
}

func (t *TempTable) StatementComplete(ctx *sql.Context) error {
	t.lookup = nil
	return nil
}

func (t *TempTable) Close(ctx *sql.Context) error {
	err := t.ed.Close(ctx)

	t.lookup = nil
	return err
}

func temporaryDoltSchema(ctx context.Context, pkSch sql.PrimaryKeySchema) (sch schema.Schema, err error) {
	cols := make([]schema.Column, len(pkSch.Schema))
	for i, col := range pkSch.Schema {
		tag := uint64(i)
		cols[i], err = sqlutil.ToDoltCol(tag, col)
		if err != nil {
			return nil, err
		}
	}

	sch, err = schema.SchemaFromCols(schema.NewColCollection(cols...))
	if err != nil {
		return nil, err
	}

	err = sch.SetPkOrdinals(pkSch.PkOrdinals)
	if err != nil {
		return nil, err
	}

	return sch, nil
}
