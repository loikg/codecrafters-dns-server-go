package main

import (
	"testing"
)

func TestLiveResolver_SendRequest(t *testing.T) {
	req := DNSMessage{
		Header: DNSHeader{
			ID:      1234,
			Flags:   DNSHeaderFlags{},
			QDCOUNT: 1,
		},
		Questions: []DNSQuestion{
			{
				Name:  "google.com",
				Type:  1,
				Class: 1,
			},
		},
	}
	resolver, err := NewResolver("1.1.1.1:53")
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}
	defer resolver.Close()

	resp, err := resolver.SendRequest(&req)
	if err != nil {
		t.Fatalf("failed to send request with resolver: %v", err)
	}

	if resp.Header.Flags.RCODE != NoErrorResponseCode {
		t.Errorf("reques failed with error code (RCODE): %d", resp.Header.Flags.RCODE)
	}
	if len(resp.Answers) < 1 {
		t.Errorf(
			"expected at least one answer instead got %d answers %+v",
			len(resp.Answers),
			resp.Answers,
		)
	}
	for _, answer := range resp.Answers {
		if answer.Name != req.Questions[0].Name {
			t.Errorf(
				"expected answers for domain name %s but instead got %s",
				req.Questions[0].Name,
				answer.Name,
			)
		}
	}
}
