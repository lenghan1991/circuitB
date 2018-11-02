package circuitB

import (
	"fmt"
	"sync"
)

type FSMState string
type FSMEvent string
type FSMHandler func() FSMState

type FSM struct {
	mu       sync.Mutex
	state    FSMState
	handlers map[FSMState]map[FSMEvent]FSMHandler
}

func NewFSM(initState FSMState) *FSM {
	return &FSM{
		state:    initState,
		handlers: make(map[FSMState]map[FSMEvent]FSMHandler),
	}
}

func (self *FSM) GetState() FSMState {
	return self.state
}

func (self *FSM) AddHandler(state FSMState, event FSMEvent, handler FSMHandler) *FSM {
	if _, ok := self.handlers[state]; !ok {
		self.handlers[state] = make(map[FSMEvent]FSMHandler)
	}
	if _, ok := self.handlers[state][event]; ok {
		fmt.Printf("Event %s on State %s Overwrited!", event, state)
	}
	self.handlers[state][event] = handler

	return self
}

func (self *FSM) Trigger(event FSMEvent) FSMState {
	self.mu.Lock()
	defer self.mu.Unlock()
	if _, ok := self.handlers[self.GetState()]; ok {
		if handler, ok := self.handlers[self.GetState()][event]; ok {
			self.state = handler()
		}
	}

	return self.GetState()
}
