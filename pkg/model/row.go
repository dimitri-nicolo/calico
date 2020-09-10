package model

import "fmt"

type Row struct {
	// Internal implementation of the row.
	row []interface{}
}

func NewRow() *Row {
	return &Row{
		row: make([]interface{}, 0),
	}
}

func (r *Row) Copy() *Row {
	r2 := make([]interface{}, len(r.row))
	copy(r2, r.row)
	return &Row{
		row: r2,
	}
}

func (r *Row) Append(elem ...interface{}) *Row {
	r.row = append(r.row, elem...)
	return r
}

func (r1 *Row) AppendRow(r2 *Row) *Row {
	return &Row{
		row: append(r1.row, r2.row...),
	}
}

func (r *Row) Replace(idx int, elem interface{}) (*Row, error) {
	if idx >= r.GetLen() {
		return nil, fmt.Errorf("Index %i out of bounds", idx)
	}
	r.row[idx] = elem
	return r, nil
}

func (r *Row) Get(idx int) (interface{}, error) {
	if idx >= r.GetLen() {
		return nil, fmt.Errorf("Index %i out of bounds", idx)
	}
	return r.row[idx], nil
}

func (r *Row) GetLen() int {
	return len(r.row)
}

func (r *Row) Flatten(idx int, fieldType string) ([]*Row, error) {
	val, err := r.Get(idx)
	if err != nil {
		return nil, err
	}

	// By converting the value at the index to a generic slice,
	// we make sure it can actually be flattened.
	arr, err := convertToGenericSlice(val, fieldType)
	if err != nil {
		return nil, err
	}

	// For each value in the slice, create a new row.
	rows := make([]*Row, len(arr))
	for i, flattenedVal := range arr {
		r2inner := make([]interface{}, r.GetLen())
		// Copy all fields of the original row into the new row, except
		// at the index to be flattened. In this index use the flattened value.
		for j := range r.row {
			if j == idx {
				r2inner[j] = flattenedVal
			} else {
				r2inner[j] = r.row[j]
			}
		}
		r2 := &Row{
			row: r2inner,
		}
		rows[i] = r2
	}
	return rows, nil
}
