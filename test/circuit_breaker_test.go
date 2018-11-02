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
		} else if times > 400 && times < 800 {
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
			FailureInterval:  10,
			RecoveryInterval: 5,
			MaximumFailure:   20,
			FailureRatio:     0.8,
		},
	)

	for i := 0; i < 1000; i++ {
		now := time.Now().Format("2006-01-02 15:04:05.99")
		if body, err := cb.Through(func() (response interface{}, err error) {
			resp, err := http.Get("http://127.0.0.1:8787/ping")
			if err != nil {
				return nil, err
			}

			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}

			if resp.StatusCode != 200 {
				return body, errors.New("Service Error")
			} else {

				return body, nil
			}

		}); err != nil {
			fmt.Printf("%s Request(%d): %s\n", now, i, err.Error())
		} else {
			fmt.Printf("%s Request(%d): %s\n", now, i, string(body.([]byte)))
		}
		time.Sleep(20 * time.Millisecond)
	}

}
