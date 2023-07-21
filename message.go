package main

const (
	ACTION_TOGGLE   = "toggle"
	ACTION_OPEN     = "open"
	ACTION_TEST     = "test"
	ACTION_PARTY    = "party"
	ACTION_VACATION = "vacation"
)

type Message struct {
	Action string `json:"action"`
}
