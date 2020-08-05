package watson

import (
	neo "github.com/minskylab/neocortex"
	"github.com/watson-developer-cloud/go-sdk/assistantv2"
)

func (watson *Cognitive) NewInput(opts *assistantv2.MessageOptions, data neo.InputData) *neo.Input {
	entities := make([]neo.Entity, 0)
	for _, e := range opts.Input.Entities {
		entities = append(entities, getNeocortexEntity(e))
	}

	intents := make([]neo.Intent, 0)
	for _, i := range opts.Input.Intents {
		intents = append(intents, getNeocortexIntent(i))
	}

	return &neo.Input{
		Data:     data,
		Intents:  intents,
		Entities: entities,
	}
}
