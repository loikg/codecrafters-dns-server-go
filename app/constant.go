package main

const (
	// OPCODE
	// See [RFC1035 4.1.1]
	// [RFC1035]: https://datatracker.ietf.org/doc/html/rfc1035#section-4.1.1
	StandardQueryOpCode = 0
	InverseQueryOpCode  = 1
	ServerStatusOpCode  = 2

	// Response Codes (RCODE)
	// See [RFC1035 4.1.1]
	// [RFC1035]: https://datatracker.ietf.org/doc/html/rfc1035#section-4.1.1
	NoErrorResponseCode        = 0
	FormatErrorResponseCode    = 1
	ServerFailureResponseCode  = 2
	NameErrorResponseCode      = 3
	NotImplementedResponseCode = 4
	RefusedResponseCode        = 5
)
