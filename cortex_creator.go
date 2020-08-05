package neocortex

import (
	"context"
	"log"
	"time"

	"os"
	"os/signal"

)

func newDefaultEngine(cognitive CognitiveService, channels ...CommunicationChannel) *Engine {
	engine := &Engine{}
	engine.channels = channels
	engine.cognitive = cognitive
	engine.registeredResolvers = map[CommunicationChannel]map[*Matcher]*HandleResolver{}
	engine.generalResolver = map[CommunicationChannel]*HandleResolver{}
	engine.registeredInjection = map[CommunicationChannel]map[*Matcher]*InInjection{}
	engine.generalInjection = map[CommunicationChannel]*InInjection{}
	engine.done = make(chan error, 1)
	engine.Register = map[string]string{}
	// engine.logger = logrus.StandardLogger() // In the future
	engine.ActiveDialogs = map[*Context]*Dialog{}
	engine.dialogPerformanceFunc = defaultPerformance
	engine.secret = "neocortex2019"

	return engine
}

// Default ...
func Default(repository Repository, cognitive CognitiveService, channels ...CommunicationChannel) (*Engine, error) {
	var err error
	engine := newDefaultEngine(cognitive, channels...)
	engine.Repository = repository
	engine.Analytics, err = newDefaultAnalytics(engine.Repository, defaultPerformance)
	if err != nil {
		return nil, err
	}
	engine.api = newCortexAPI(repository, engine.Analytics, "/api", ":4200")

	fabric := func(ctx context.Context, info PersonInfo) *Context {
		newContext := cognitive.CreateNewContext(&ctx, info)
		return newContext
	}

	cognitive.OnContextIsDone(func(c *Context) {
		engine.onContextIsDone(c)
	})

	for _, ch := range channels {
		engine.registeredResolvers[ch] = map[*Matcher]*HandleResolver{}
		ch.SetContextFabric(fabric)
		err := ch.RegisterMessageEndpoint(func(c *Context, message *Input, response OutputResponse) error {
			return engine.onMessage(ch, c, message, response)
		})

		if err != nil {
			return nil, err
		}

		ch.OnNewContextCreated(func(c *Context) {
			engine.onNewContextCreated(c)
		})

		ch.OnContextIsDone(func(c *Context) {
			engine.onContextIsDone(c)
		})

		go func(ch *CommunicationChannel) {
			err := (*ch).ToHear()
			if err != nil {
				engine.done <- err
			}
		}(&ch)
	}

	return engine, nil
}

func (engine *Engine) Run() error {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	gc := defaultGarbageCollector(10 * time.Minute)

	go func() {
		<-signalChan
		log.Println("Closing all dialogs, total: ", len(engine.ActiveDialogs))
		for c := range engine.ActiveDialogs {
			engine.onContextIsDone(c)
		}
		engine.done <- nil
	}()
	go func() {
		if engine.api.repository != nil {
			engine.done <- engine.api.Launch(engine)
		}
	}()

	engine.runGarbageCollector(gc)

	return <-engine.done
}
