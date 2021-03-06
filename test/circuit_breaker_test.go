package test

import (
	"errors"
	"fmt"
	"github/lenghan1991/circuitB"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

func NewFakeServer() {
	times := 0

	http.HandleFunc("/ping", func(responseWriter http.ResponseWriter, request *http.Request) {
		times++
		if times < 100 {
			if times%8 != 0 {
				http.Error(responseWriter, "Unavailable", 503)
			} else {
				fmt.Fprintf(responseWriter, "PONG")
			}
		} else if times > 200 && times < 300 {
			if times%7 != 0 {
				http.Error(responseWriter, "Unavailable", 503)
			} else {
				fmt.Fprintf(responseWriter, "PONG")
			}
		} else {
			fmt.Fprintf(responseWriter, "PONG")
		}

	})

	http.ListenAndServe(":8787", nil)
}

func Test_circuit_breaker(T *testing.T) {
	go NewFakeServer()
	time.Sleep(1 * time.Second)

	cb := circuitB.NewCircuitBreaker(
		&circuitB.CBConf{
			FailureInterval:  5,
			RecoveryInterval: 2,
			MaximumFailure:   20,
			FailureRatio:     0.8,
		},
	)
	for i := 0; i < 400; i++ {
		now := time.Now().Format("2006-01-02 15:04:05.99")

		if body, err := cb.Through(func() (response interface{}, err error) {
			resp, err := http.Get("http://127.0.0.1:8787/ping")
			if err != nil {
				return nil, err
			}

			if resp.StatusCode != 200 {
				return nil, errors.New("Service Error")
			} else {
				defer resp.Body.Close()
				return ioutil.ReadAll(resp.Body)
			}

		}); err != nil {
			fmt.Printf("%s Request(%d): %s\n", now, i, err.Error())
		} else {
			fmt.Printf("%s Request(%d): %s\n", now, i, string(body.([]byte)))
		}
		time.Sleep(50 * time.Millisecond)
	}
}
