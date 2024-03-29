package dbr

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
)

// InsertStmt builds `INSERT INTO ...`.
type InsertStmt struct {
	runner
	EventReceiver
	Dialect
	raw
	Table        string
	Column       []string
	Value        [][]interface{}
	RunLen       int
	ReturnColumn []string
	RecordID     *int64
}

type InsertBuilder = InsertStmt

func (b *InsertStmt) Build(d Dialect, buf Buffer) error {
	//赋予批量插入默认最大上限
	if b.RunLen==0{
		b.RunLen=1000
	}
	if b.raw.Query != "" {
		return b.raw.Build(d, buf)
	}

	if b.Table == "" {
		return ErrTableNotSpecified
	}

	if len(b.Column) == 0 {
		return ErrColumnNotSpecified
	}

	buf.WriteString("INSERT INTO ")
	buf.WriteString(d.QuoteIdent(b.Table))

	var placeholderBuf strings.Builder
	placeholderBuf.WriteString("(")
	buf.WriteString(" (")
	for i, col := range b.Column {
		if i > 0 {
			buf.WriteString(",")
			placeholderBuf.WriteString(",")
		}
		buf.WriteString(d.QuoteIdent(col))
		placeholderBuf.WriteString(placeholder)
	}
	buf.WriteString(") VALUES ")
	placeholderBuf.WriteString(")")
	placeholderStr := placeholderBuf.String()
	var runnum int
	for i, tuple := range b.Value {

		//超过更新行数
		if i>=b.RunLen{
			break
		}
		//更新数量
		runnum++
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(placeholderStr)
		buf.WriteValue(tuple...)
	}
	//进行截取
	b.Value=b.Value[runnum:]
	if len(b.ReturnColumn) > 0 {
		buf.WriteString(" RETURNING ")
		for i, col := range b.ReturnColumn {
			if i > 0 {
				buf.WriteString(",")
			}
			buf.WriteString(d.QuoteIdent(col))
		}
	}

	return nil
}

// InsertInto creates an InsertStmt.
func InsertInto(table string) *InsertStmt {
	return &InsertStmt{
		Table: table,
	}
}

// InsertInto creates an InsertStmt.
func (sess *Session) InsertInto(table string) *InsertStmt {
	b := InsertInto(table)
	b.runner = sess
	b.EventReceiver = sess.EventReceiver
	b.Dialect = sess.Dialect
	return b
}

// InsertInto creates an InsertStmt.
func (tx *Tx) InsertInto(table string) *InsertStmt {
	b := InsertInto(table)
	b.runner = tx
	b.EventReceiver = tx.EventReceiver
	b.Dialect = tx.Dialect
	return b
}

// InsertBySql creates an InsertStmt from raw query.
func InsertBySql(query string, value ...interface{}) *InsertStmt {
	return &InsertStmt{
		raw: raw{
			Query: query,
			Value: value,
		},
	}
}

// InsertBySql creates an InsertStmt from raw query.
func (sess *Session) InsertBySql(query string, value ...interface{}) *InsertStmt {
	b := InsertBySql(query, value...)
	b.runner = sess
	b.EventReceiver = sess.EventReceiver
	b.Dialect = sess.Dialect
	return b
}

// InsertBySql creates an InsertStmt from raw query.
func (tx *Tx) InsertBySql(query string, value ...interface{}) *InsertStmt {
	b := InsertBySql(query, value...)
	b.runner = tx
	b.EventReceiver = tx.EventReceiver
	b.Dialect = tx.Dialect
	return b
}

func (b *InsertStmt) Columns(column ...string) *InsertStmt {
	b.Column = column
	return b
}

// Values adds a tuple to be inserted.
// The order of the tuple should match Columns.
func (b *InsertStmt) Values(value ...interface{}) *InsertStmt {
	b.Value = append(b.Value, value)
	return b
}

// Record adds a tuple for columns from a struct.
//
// If there is a field called "Id" or "ID" in the struct,
// it will be set to LastInsertId.
func (b *InsertStmt) Record(structValue interface{}) *InsertStmt {
	v := reflect.Indirect(reflect.ValueOf(structValue))

	if v.Kind() == reflect.Struct {
		found := make([]interface{}, len(b.Column)+1)
		// ID is recommended by golint here
		s := newTagStore()
		s.findValueByName(v, append(b.Column, "id"), found, false)

		value := found[:len(found)-1]
		for i, v := range value {
			if v != nil {
				value[i] = v.(reflect.Value).Interface()
			}
		}

		if v.CanSet() {
			switch idField := found[len(found)-1].(type) {
			case reflect.Value:
				if idField.Kind() == reflect.Int64 {
					b.RecordID = idField.Addr().Interface().(*int64)
				}
			}
		}
		b.Values(value...)
	}
	return b
}

//插入map，key为column，value为value
func (b *InsertStmt) Map(kv map[string]interface{}) *InsertStmt {
	value := []interface{}{}
	if len(b.Column) == 0 {
		for k, v := range kv {
			b.Column = append(b.Column, k)
			value = append(value, v)
		}
	} else {
		for _, col := range b.Column {
			v := kv[col]
			value = append(value, v)
		}
	}
	b.Value = append(b.Value, value)
	return b
}

// Returning specifies the returning columns for postgres.
func (b *InsertStmt) Returning(column ...string) *InsertStmt {
	b.ReturnColumn = column
	return b
}

// Pair adds (column, value) to be inserted.
// It is an error to mix Pair with Values and Record.
func (b *InsertStmt) Pair(column string, value interface{}) *InsertStmt {
	b.Column = append(b.Column, column)
	switch len(b.Value) {
	case 0:
		b.Values(value)
	case 1:
		b.Value[0] = append(b.Value[0], value)
	default:
		panic("pair only allows one record to insert")
	}
	return b
}

//获取SQL
func (b *InsertStmt) GetSQL() (string, error) {
	b1 := *b
	b2 := &b1
	return getSQL(b2, b2.Dialect)
}

func (b *InsertStmt) Exec() (sql.Result, error) {
	return b.ExecContext(context.Background())
}
// Returning specifies the returning columns for postgres.
func (b *InsertStmt) SetRunLen(i int) *InsertStmt {
	//b.runnum
	b.RunLen = i
	return b
}

func (b *InsertStmt) ExecContext(ctx context.Context) (sql.Result, error) {
	var err error
	var result sql.Result
	for len(b.Value) > 0 && err == nil {
		//_, err = b.ExecContext(context.Background())
		result, err = exec(ctx, b.runner, b.EventReceiver, b, b.Dialect)
		if err != nil {
			return nil, err
		}
		if b.RecordID != nil {
			if id, err := result.LastInsertId(); err == nil {
				*b.RecordID = id
			}
			b.RecordID = nil
		}
	}
	return result, nil
}

func (b *InsertStmt) LoadContext(ctx context.Context, value interface{}) error {
	_, err := query(ctx, b.runner, b.EventReceiver, b, b.Dialect, value)
	return err
}

func (b *InsertStmt) Load(value interface{}) error {
	return b.LoadContext(context.Background(), value)
}
