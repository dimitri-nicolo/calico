package model

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

const (
	NestedLoopJoinStrategy = "NestedLoop"
	LeftPrefix             = "left"
	RightPrefix            = "right"
)

// Collection of views. Views should be APPEND-ONLY.
type Views struct {
	views map[string]View
}

// Representation of our data.
// Elements in fieldNames, rows, and viewMap are assumed to be in the same order.
type View struct {
	fieldNames []string
	rows       []Row
	viewMap    ViewMap
}

func CreateViews(existingViews *Views) *Views {
	if existingViews == nil {
		return &Views{
			views: make(map[string]View),
		}
	} else {
		return &Views{
			views: existingViews.views,
		}
	}
}

func CreateView(fieldNames []string, vm ViewMap) (*View, error) {
	return &View{
		fieldNames: fieldNames,
		rows:       make([]Row, 0),
		viewMap:    vm,
	}, nil
}

// Make a views copy.
func (vs *Views) Copy() *Views {
	vs2 := make(map[string]View)
	for k, v := range vs.views {
		vs2[k] = v
	}
	return &Views{
		views: vs2,
	}
}

func (vs *Views) AddView(viewName string, v *View) error {
	// Check if view dimensions match.
	// Only check first row to reduce computation time,
	// because the number of rows can be large.
	if len(v.rows) > 0 && v.rows[0].GetLen() != len(v.fieldNames) {
		return fmt.Errorf("Row does not have same dimension as field names")
	}

	// check for duplicate view names
	if _, ok := vs.views[viewName]; ok {
		log.WithField("viewName", viewName).Error("Cannot have duplicate view names")
		return fmt.Errorf("Duplicate view name " + viewName)
	}
	vs.views[viewName] = *v
	return nil
}

func (vs *Views) GetView(viewName string) (*View, error) {
	if v, ok := vs.views[viewName]; ok {
		return &v, nil
	}
	return nil, fmt.Errorf("View %s does not exist", viewName)
}

// Deep copy of view.
func (v *View) Copy() *View {
	fieldNames := make([]string, len(v.fieldNames))
	copy(fieldNames, v.fieldNames)

	rows := make([]Row, len(v.rows))
	copy(rows, v.rows)

	v2 := View{
		fieldNames: fieldNames,
		rows:       rows,
		viewMap:    v.viewMap,
	}
	return &v2
}

// Copy a view, deciding which rows to keep based on a filter.
// If filter is true, remove row. Otherwise keep row.
func (v *View) CopyWithFilter(filter []bool) *View {
	fieldNames := make([]string, len(v.fieldNames))
	copy(fieldNames, v.fieldNames)

	rows := make([]Row, 0)
	for i := range v.rows {
		if !filter[i] {
			rows = append(rows, v.rows[i])
		}
	}

	v2 := View{
		fieldNames: fieldNames,
		rows:       rows,
		viewMap:    v.viewMap,
	}
	return &v2
}

func (v *View) ReplaceColumn(idx int, columnValues []interface{}) error {
	if len(columnValues) != len(v.rows) {
		return fmt.Errorf("Size of new values (%i) doesn't match the number of rows (%i)", len(columnValues), len(v.rows))
	}
	if len(v.rows) > 0 && idx > v.rows[0].GetLen() {
		return fmt.Errorf("Index (%i) to insert new value is too large, row size %i", idx, v.rows[0].GetLen())
	}

	for i, _ := range v.rows {
		// Check if we are appending a new column or replacing an existing column.
		if idx == v.rows[i].GetLen() {
			v.rows[i].Append(columnValues[i])
		} else {
			v.rows[i].Replace(idx, columnValues[i])
		}
	}
	return nil
}

func (v *View) ReplaceFieldNames(fieldNames []string) {
	v.fieldNames = fieldNames
}

func (v *View) AddRow(r *Row) {
	v.rows = append(v.rows, *r)
}

// For each row in the view, get the values of the fields given.
// Return a channel where each element is a slice of the values for one row.
// Each value slice has the same ordering as the fields parameter.
func (v *View) GetFields(fields []string) <-chan []interface{} {
	c := make(chan []interface{})

	go func() {
		// get indices represented by fields
		indices := make([]int, 0)
		for _, f1 := range fields {
			for i, f2 := range v.fieldNames {
				if f1 == f2 {
					indices = append(indices, i)
				}
			}
		}

		for _, r := range v.rows {
			values := make([]interface{}, 0)
			for _, i := range indices {
				if val, err := r.Get(i); err == nil {
					values = append(values, val)
				} else {
					log.Errorf("%v", err)
				}
			}
			c <- values
		}
		close(c)
	}()

	return c
}

func (v *View) GetRowSize() int {
	return len(v.rows)
}

func (v *View) GetColumnSize() int {
	// don't use v.rows[0], since rows can be empty
	return len(v.fieldNames)
}

// Create a new view which is the join of two views.
// All columns for both the left and right views are kept after a join.
// Left columns are prefixed with a "left" and right columns with a "right".
// E.g. left  = [resource, namespace]
//      right = [useragent, namespace]
//      left join right on left.namespace=right.namespace produces:
//      [left.resource, left.namespace, right.useragent, right.namespace]
func (vLeft *View) Join(vRight *View, leftKey, rightKey, joinStrategy string) (*View, error) {
	// get indices of join keys
	leftKeyIdx, err := vLeft.getFieldIndex(leftKey)
	if err != nil {
		return nil, fmt.Errorf("Cannot find index of left key %s: %v", leftKey, err)
	}
	rightKeyIdx, err := vRight.getFieldIndex(rightKey)
	if err != nil {
		return nil, fmt.Errorf("Cannot find index of right key %s: %v", rightKey, err)
	}

	switch joinStrategy {
	case NestedLoopJoinStrategy:
		return vLeft.nestedLoopJoin(vRight, leftKeyIdx, rightKeyIdx)
	default:
		return nil, fmt.Errorf("Unrecognized join strategy %s", joinStrategy)
	}
}

func (vLeft *View) nestedLoopJoin(vRight *View, leftKeyIdx, rightKeyIdx int) (*View, error) {
	// set field names of new view
	fieldNames := make([]string, len(vLeft.fieldNames)+len(vRight.fieldNames))
	for i, n := range vLeft.fieldNames {
		fieldNames[i] = fmt.Sprintf("%s.%s", LeftPrefix, n)
	}
	for i, n := range vRight.fieldNames {
		fieldNames[i+len(vLeft.fieldNames)] = fmt.Sprintf("%s.%s", RightPrefix, n)
	}

	v, err := CreateView(fieldNames, nil)
	if err != nil {
		return nil, err
	}

	// join
	for _, ri := range vLeft.rows {
		for _, rj := range vRight.rows {
			leftVal, err := ri.Get(leftKeyIdx)
			if err != nil {
				return nil, err
			}
			rightVal, err := rj.Get(rightKeyIdx)
			if err != nil {
				return nil, err
			}
			if leftVal == rightVal {
				// two rows join successfully, make new row
				r := ri.Copy().AppendRow(&rj)
				v.AddRow(r)
			}
		}
	}

	return v, nil
}

// Flatten the view based on a given from-field.
// The field is assumed to be nested in some way, e.g. slice.
// The new view contains the same fields, except for the
// from-field being flattened. The from-field is replaced by a new to-field.
func (v *View) Flatten(fromField, toField string) (*View, error) {
	var idx int
	var fieldType string
	foundField := false
	for i, m := range v.viewMap {
		if m.TargetField == fromField {
			idx = i
			fieldType = m.Type
			foundField = true
			break
		}
	}
	if !foundField {
		return nil, fmt.Errorf("Field %s not found", fromField)
	}

	// Create new viewmap
	vm2, err := v.viewMap.createNewFromFlatten(fromField, toField)
	if err != nil {
		return nil, err
	}

	v2, err := CreateView(v.fieldNames, *vm2)
	if err != nil {
		return nil, err
	}

	for _, row := range v.rows {
		newRows, err := row.Flatten(idx, fieldType)
		if err != nil {
			return nil, err
		}

		for _, newRow := range newRows {
			v2.AddRow(newRow)
		}
	}
	return v2, nil
}

func (v *View) getFieldIndex(field string) (int, error) {
	for i, fn := range v.fieldNames {
		if field == fn {
			return i, nil
		}
	}
	return -1, fmt.Errorf("Cannot find field %s", field)
}

func (v *View) GetFieldNames() []string {
	return v.fieldNames
}

func (v View) String() (s string) {
	s = fmt.Sprintf("\t%v", v.fieldNames)
	for _, r := range v.rows {
		s = fmt.Sprintf("%s\n\t%v", s, r.row)
	}
	return
}

// Returns true if the view has no rows, false otherwise.
func (v *View) IsEmpty() bool {
	return v.rows == nil || len(v.rows) == 0
}
