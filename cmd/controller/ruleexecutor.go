package rule

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	model "github.com/tigera/honeypod-recommendation/pkg/model"
	"github.com/tigera/lma/pkg/elastic"
)

type RuleExecutor struct {
	rule          Rule
	ctx           context.Context
	views         *model.Views
	conditionsMet map[string]bool
}

type Processor interface {
	Process() (*model.View, error)
	getFieldNames() ([]string, error)
	Name() string
}

func NewRuleExecutor(r Rule, globalRe *RuleExecutor) *RuleExecutor {
	var vs *model.Views
	if globalRe == nil {
		vs = nil
	} else {
		// Since the global views can be used in conjunction with multiple
		// rules, it's best we make a copy for each rule that requires it.
		// Otherwise, the views for each rule is going to be added into
		// the global views and carried through to future rules, and
		// this can be quite costly in memory and complexity.
		vs = globalRe.views.Copy()

		// Append global parameters so they can be used in the rule.
		// If parameter is already defined in the rule, do NOT overwrite it.
		// Rule parameters take precedence over global parameters.
		if r.Parameters == nil {
			r.Parameters = make(map[string]string)
		}
		for k, v := range globalRe.rule.Parameters {
			if _, ok := r.Parameters[k]; !ok {
				r.Parameters[k] = v
			}
		}
	}
	re := RuleExecutor{
		rule:          r,
		ctx:           context.Background(),
		views:         model.CreateViews(vs),
		conditionsMet: make(map[string]bool, 0),
	}
	return &re
}

func (re *RuleExecutor) Run() ([]Recommendation, error) {
	// Init elastic.
	elasticClient := elastic.MustGetElasticClient()

	// process elastic queries
	for _, cfg := range re.rule.ElasticProcessorCfgs {
		p, err := NewElasticProcessor(cfg, elasticClient, re.ctx, &re.rule)
		if err != nil {
			return nil, err
		}
		if err := re.executeProcessor(p); err != nil {
			return nil, err
		}
	}

	// data processing
	for _, cfg := range re.rule.DataProcessorCfgs {
		p, err := NewProcessor(cfg, re.views)
		if err != nil {
			return nil, err
		}

		if err = re.executeProcessor(p); err != nil {
			return nil, err
		}
	}

	// generate recommendations
	var suggestedRecs []Recommendation
	for _, cfg := range re.rule.RecommendationGeneratorCfgs {
		g, err := NewRecommendationGenerator(cfg, re.views)
		if err != nil {
			return nil, fmt.Errorf("Error encountered creating recommendation generator: %v", err)
		}

		recs, err := g.GenerateRecommendations()
		if err != nil {
			return nil, fmt.Errorf("Error encountered generating recommendations: %v", err)
		}
		suggestedRecs = append(suggestedRecs, recs...)

		log.WithFields(log.Fields{
			"recommendation generator": cfg.Type,
			"recommendations":          suggestedRecs,
		}).Debug("Generator completed")
	}

	return suggestedRecs, nil
}

func (re *RuleExecutor) executeProcessor(p Processor) error {
	v, err := p.Process()
	if err != nil {
		return err
	}
	err = re.addView(p.Name(), v)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"processor": p.Name(),
		"view":      v,
	}).Debug("Processor completed")
	return nil
}

func (re *RuleExecutor) addView(viewName string, v *model.View) error {
	if err := re.views.AddView(viewName, v); err != nil {
		return fmt.Errorf("Error getting view %s: %v", viewName, err)
	}
	return nil
}
