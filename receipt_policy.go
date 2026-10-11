package main

import "strings"

type PaymentEvent struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Risk  string `json:"risk"`
	Image string `json:"image"`
}

type Decision struct {
	Action string `json:"action"`
	Notice string `json:"notice"`
}

func decide(e PaymentEvent) Decision {
	if e.Kind == "chargeback" || e.Risk == "high" {
		return Decision{Action: "review", Notice: "asset held for manual review"}
	}
	return Decision{Action: "publish", Notice: "optimized receipt available"}
}

func validEvent(e PaymentEvent) bool {
	return e.ID != "" && !strings.ContainsAny(e.ID, "/\\") && (e.Kind == "payment" || e.Kind == "chargeback") && (e.Risk == "low" || e.Risk == "high") && e.Image != ""
}
