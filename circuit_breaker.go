package circuitB

import (
	"errors"
	"fmt"
	"time"
)

const (
	CLOSED    = "closed"
	OPEN      = "open"
	HALF_OPEN = "half_open"

	MAXIUM_FAILURE_REACHED  = "maxium_failure_reached"
	SERVICE_UNREACHABLE     = "service_unreachable"
	RECOVERY_TIMEOUT        = "recovery_timeout"
	MINIMUM_SUCCESS_REACHED = "minimum_success_reached"

	DEFAULT_FAILURE_INTERVAL  = 20 * time.Second
	DEFAULT_RECOVERY_INTERVAL = 40 * time.Second
	DEFAULT_FAILURE_RATIO     = 0.8
)

var (
	ZERO_TIME          = time.Time{}
	ErrOpenState       = errors.New("Circuit Breaker is Open")
	ErrTooManyRequests = errors.New("Too Many Requests")
)

// TODO: Exception Handling (Client through CircuitBreaker)
// TODO: Exception Judgement Strategy (e.g. Overload => larger timeout)
// TODO: Logging
// TODO: Recoverability (e.g. Open State for too long)
// TODO: HeartBeating on Backend Service
// TODO: Manual Override
// TODO: Concurrency
// TODO: Resource Differentiation
// TODO: Accelerated Circuit Breaking (e.g. HTTP 503 Service Unavailable, should switch to Open immediately)
// TODO: Replaying Failed Requests
// TODO: Inappropriate Timeouts on External Services

type CircuitBreaker struct {
	fsm *FSM

	failureCount   int
	failureRatio   float64
	maximumFailure int

	totalRequestsCount  int
	successRequestCount int

	failureInterval  time.Duration
	recoveryInterval time.Duration

	expire   time.Time
	recovery time.Time
}

type CBConf struct {
	FailureInterval  time.Duration
	RecoveryInterval time.Duration

	MaximumFailure int
	FailureRatio   float64
}

func NewCircuitBreaker(conf *CBConf) *CircuitBreaker {
	failureInterval := DEFAULT_FAILURE_INTERVAL
	recoveryInterval := DEFAULT_RECOVERY_INTERVAL
	failureRatio := DEFAULT_FAILURE_RATIO

	if conf.FailureInterval > 0 {
		failureInterval = conf.FailureInterval
	}

	if conf.RecoveryInterval > 0 {
		recoveryInterval = conf.RecoveryInterval
	}

	if conf.FailureRatio > 0 {
		failureRatio = conf.FailureRatio
	}

	cb := &CircuitBreaker{
		failureCount:        0,
		totalRequestsCount:  0,
		successRequestCount: 0,
		failureInterval:     failureInterval,
		recoveryInterval:    recoveryInterval,
		maximumFailure:      conf.MaximumFailure,
		failureRatio:        failureRatio,
	}

	cb.InitFSM()

	return cb
}

func (self *CircuitBreaker) InitFSM() {
	fsm := NewFSM(CLOSED)
	fsm.AddHandler(CLOSED, MAXIUM_FAILURE_REACHED, func() FSMState {
		return OPEN
	})

	fsm.AddHandler(CLOSED, SERVICE_UNREACHABLE, func() FSMState {
		return OPEN
	})

	fsm.AddHandler(OPEN, RECOVERY_TIMEOUT, func() FSMState {
		return HALF_OPEN
	})

	fsm.AddHandler(HALF_OPEN, MINIMUM_SUCCESS_REACHED, func() FSMState {
		self.expire = time.Now().Add(self.failureInterval)

		return CLOSED
	})
	fsm.AddHandler(HALF_OPEN, MAXIUM_FAILURE_REACHED, func() FSMState {
		return OPEN
	})

	self.fsm = fsm
}

func (self *CircuitBreaker) ResetCloseState() {
	self.failureCount = 0
	self.totalRequestsCount = 0
	self.expire = time.Now().Add(self.failureInterval * time.Second)
	self.recovery = ZERO_TIME
}

func (self *CircuitBreaker) ResetOpenState() {
	self.failureCount = 0
	self.totalRequestsCount = 0
	self.expire = ZERO_TIME
	self.recovery = time.Now().Add(self.recoveryInterval * time.Second)
}

func (self *CircuitBreaker) ResetHalfOpenState() {
	self.failureCount = 0
	self.totalRequestsCount = 0
	self.expire = ZERO_TIME
	self.recovery = ZERO_TIME
}

func (self *CircuitBreaker) IsOverFailureRatio() bool {
	return float64(self.failureCount)/float64(self.totalRequestsCount) >= self.failureRatio
}

func (self *CircuitBreaker) wrapReq(request func() (response interface{}, err error)) (response interface{}, err error) {
	resp, err := request()
	self.totalRequestsCount++
	if err != nil {
		self.failureCount++
	}

	return resp, err
}

func (self *CircuitBreaker) Through(request func() (response interface{}, err error)) (response interface{}, err error) {
	switch self.fsm.GetState() {
	case CLOSED:
		resp, err := self.wrapReq(request)
		if time.Now().After(self.expire) {
			self.ResetCloseState()
		} else if self.IsOverFailureRatio() && self.failureCount >= self.maximumFailure {
			state := self.fsm.Trigger(MAXIUM_FAILURE_REACHED)
			if state == OPEN {
				self.ResetOpenState()
			}
		}

		return resp, err
	case OPEN:
		if time.Now().After(self.recovery) {
			state := self.fsm.Trigger(RECOVERY_TIMEOUT)
			if state == HALF_OPEN {
				self.ResetHalfOpenState()

				return self.wrapReq(request)
			}
		} else {
			return nil, ErrOpenState
		}
	case HALF_OPEN:
		if self.IsOverFailureRatio() && self.failureCount >= self.maximumFailure {
			state := self.fsm.Trigger(MAXIUM_FAILURE_REACHED)
			if state == OPEN {
				self.ResetOpenState()
			}
			return nil, ErrTooManyRequests
		}

		return self.wrapReq(request)
	default:
	}

	return nil, ErrUnknownState(string(self.fsm.GetState()))
}

func ErrUnknownState(state string) error {
	return errors.New(fmt.Sprintf("Circuit Breaker is on Unknown State %s", state))
}
