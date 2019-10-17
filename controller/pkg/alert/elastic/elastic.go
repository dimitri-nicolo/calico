// Copyright 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"github.com/projectcalico/libcalico-go/lib/validator/v3/query"
)

type JsonObject map[string]interface{}

type ElasticQueryConverter interface {
	Convert(*query.Query) interface{}
}

type converter struct {
	atomToElastic func(atom *query.Atom) interface{}
}

func comparatorToElastic(c query.Comparator, key string, value interface{}) interface{} {
	switch c {
	case query.CmpEqual:
		return JsonObject{
			"term": JsonObject{
				key: JsonObject{
					"value": value,
				},
			},
		}
	case query.CmpNotEqual:
		return JsonObject{
			"bool": JsonObject{
				"must_not": JsonObject{
					"term": JsonObject{
						key: JsonObject{
							"value": value,
						},
					},
				},
			},
		}
	case query.CmpLt, query.CmpLte, query.CmpGt, query.CmpGte:
		return JsonObject{
			"range": JsonObject{
				key: JsonObject{
					c.ToElasticFunc(): value,
				},
			},
		}
	}
	panic("unknown operator")
}

func basicAtomToElastic(k *query.Atom) interface{} {
	return comparatorToElastic(k.Comparator, k.Key, k.Value)
}

func (c converter) valueToElastic(v *query.Value) interface{} {
	if v.Atom != nil {
		return c.atomToElastic(v.Atom)
	}
	if v.Subquery != nil {
		return c.Convert(v.Subquery)
	}
	panic("empty value")
}

func (c converter) unaryOpTermToElastic(v *query.UnaryOpTerm) interface{} {
	if v.Negator != nil {
		return JsonObject{
			"bool": JsonObject{
				"must_not": c.valueToElastic(v.Value),
			},
		}
	}
	return c.valueToElastic(v.Value)
}

func (c converter) opValueToElastic(o *query.OpValue) interface{} {
	return c.unaryOpTermToElastic(o.Value)
}

func (c converter) termToElastic(t *query.Term) interface{} {
	terms := []interface{}{c.unaryOpTermToElastic(t.Left)}
	for _, r := range t.Right {
		terms = append(terms, c.opValueToElastic(r))
	}

	if len(terms) == 1 {
		return terms[0]
	}

	return JsonObject{
		"bool": JsonObject{
			"must": terms,
		},
	}
}

func (c converter) opTermToElastic(o *query.OpTerm) interface{} {
	return c.termToElastic(o.Term)
}

func (c converter) Convert(q *query.Query) interface{} {
	if q.Left == nil {
		return JsonObject{
			"match_all": JsonObject{},
		}
	}
	terms := []interface{}{c.termToElastic(q.Left)}

	for _, r := range q.Right {
		terms = append(terms, c.opTermToElastic(r))
	}

	if len(terms) == 1 {
		return terms[0]
	}

	return JsonObject{
		"bool": JsonObject{
			"should": terms,
		},
	}
}
