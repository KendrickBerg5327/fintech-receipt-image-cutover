package main

import "testing"

func TestReceiptDecision(t *testing.T) {
	for _, tc := range []struct {
		name  string
		event PaymentEvent
		want  string
	}{
		{"approved payment", PaymentEvent{Kind: "payment", Risk: "low"}, "publish"},
		{"high risk payment", PaymentEvent{Kind: "payment", Risk: "high"}, "review"},
		{"chargeback", PaymentEvent{Kind: "chargeback", Risk: "low"}, "review"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := decide(tc.event).Action; got != tc.want {
				t.Fatalf("action = %q, want %q", got, tc.want)
			}
		})
	}
}
