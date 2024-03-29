package dbr

import (
	"context"
	"database/sql"
	"strconv"
)

// UpdateStmt builds `UPDATE ...`.
type UpdateStmt struct {
	runner
	EventReceiver
	Dialect

	raw

	Table      string
	Value      map[string]interface{}
	WhereCond  []Builder
	LimitCount int64
}

type UpdateBuilder = UpdateStmt

func (b *UpdateStmt) Build(d Dialect, buf Buffer) error {
	if b.raw.Query != "" {
		return b.raw.Build(d, buf)
	}

	if b.Table == "" {
		return ErrTableNotSpecified
	}

	if len(b.Value) == 0 {
		return ErrColumnNotSpecified
	}

	buf.WriteString("UPDATE ")
	buf.WriteString(d.QuoteIdent(b.Table))
	buf.WriteString(" SET ")

	i := 0
	for col, v := range b.Value {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(d.QuoteIdent(col))
		buf.WriteString(" = ")
		switch v := v.(type) {
		case raw:
			v.Build(d, buf)
		default:
			buf.WriteString(placeholder)
			buf.WriteValue(v)
		}
		i++
	}

	if len(b.WhereCond) > 0 {
		buf.WriteString(" WHERE ")
		err := And(b.WhereCond...).Build(d, buf)
		if err != nil {
			return err
		}
	}else{
		panic("没有条件")
	}

	if b.LimitCount >= 0 {
		buf.WriteString(" LIMIT ")
		buf.WriteString(strconv.FormatInt(b.LimitCount, 10))
	}

	return nil
}

// Update creates an UpdateStmt.
func Update(table string) *UpdateStmt {
	return &UpdateStmt{
		Table:      table,
		Value:      make(map[string]interface{}),
		LimitCount: -1,
	}
}

// Update creates an UpdateStmt.
func (sess *Session) Update(table string) *UpdateStmt {
	b := Update(table)
	b.runner = sess
	b.EventReceiver = sess.EventReceiver
	b.Dialect = sess.Dialect
	return b
}

// Update creates an UpdateStmt.
func (tx *Tx) Update(table string) *UpdateStmt {
	b := Update(table)
	b.runner = tx
	b.EventReceiver = tx.EventReceiver
	b.Dialect = tx.Dialect
	return b
}

// UpdateBySql creates an UpdateStmt with raw query.
func UpdateBySql(query string, value ...interface{}) *UpdateStmt {
	return &UpdateStmt{
		raw: raw{
			Query: query,
			Value: value,
		},
		Value:      make(map[string]interface{}),
		LimitCount: -1,
	}
}

// UpdateBySql creates an UpdateStmt with raw query.
func (sess *Session) UpdateBySql(query string, value ...interface{}) *UpdateStmt {
	b := UpdateBySql(query, value...)
	b.runner = sess
	b.EventReceiver = sess.EventReceiver
	b.Dialect = sess.Dialect
	return b
}

// UpdateBySql creates an UpdateStmt with raw query.
func (tx *Tx) UpdateBySql(query string, value ...interface{}) *UpdateStmt {
	b := UpdateBySql(query, value...)
	b.runner = tx
	b.EventReceiver = tx.EventReceiver
	b.Dialect = tx.Dialect
	return b
}

// Where adds a where condition.
// query can be Builder or string. value is used only if query type is string.
func (b *UpdateStmt) Where(query interface{}, value ...interface{}) *UpdateStmt {
	switch query := query.(type) {
	case string:
		b.WhereCond = append(b.WhereCond, Expr(query, value...))
	case Builder:
		b.WhereCond = append(b.WhereCond, query)
	}
	return b
}

// Set updates column with value.
func (b *UpdateStmt) Set(column string, value interface{}) *UpdateStmt {
	b.Value[column] = value
	return b
}

// SetMap specifies a map of (column, value) to update in bulk.
func (b *UpdateStmt) SetMap(m map[string]interface{}) *UpdateStmt {
	for col, val := range m {
		b.Set(col, val)
	}
	return b
}

func (b *UpdateStmt) Limit(n uint64) *UpdateStmt {
	b.LimitCount = int64(n)
	return b
}

//获取SQL
func (b *UpdateStmt) GetSQL() (string, error) {
	b1 := *b
	b2 := &b1
	return getSQL(b2, b2.Dialect)
}

func (b *UpdateStmt) Exec() (sql.Result, error) {
	return b.ExecContext(context.Background())
}

func (b *UpdateStmt) ExecContext(ctx context.Context) (sql.Result, error) {
	return exec(ctx, b.runner, b.EventReceiver, b, b.Dialect)
}
