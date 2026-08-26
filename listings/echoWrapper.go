
func (con *EchoConnection) SendEchoWithTTL(content string, ttl int) (*EchoResponse, error) {
	// save request id to filter it later
	requestID := rand.Intn(RANDOM_ID_MAX)

	// Build ICMP echo request
	msg := icmp.Message{
		Type: con.ipVersionStrategy.GetIcmpEchoType(),
		Body: &icmp.Echo{
			ID:   requestID,
			Seq:  con.sequenceNumber,
			Data: []byte(content),
		},
	}
	con.ipVersionStrategy.SetTTL(ttl)
	b, err := msg.Marshal(nil)

	// sends the message and waits its answer
	start := time.Now()
	err := con.connection.WriteTo(b, con.address)
	// Wait for a reply
	con.connection.SetReadDeadline(time.Now().Add(timeout))
	for {
		n, peer, err := con.connection.ReadFrom(reply)
		rtt := time.Since(start)

		if err != nil {
			// ... Timeout - no response
		}

		// Parse the ICMP reply
		parsedReply, err := icmp.ParseMessage(reply)

		responseType := con.ipVersionStrategy.CheckResponseType(parsedReply.Type)
		// If its not the package sent before, continue listening
		switch responseType {
		case networkstrategy.ICMPEchoReply:
			// handle echo reply (check package id)
		case networkstrategy.ICMPTimeExceeded:
			// handle time exceeded error
		default:
			// handle unexpected package (different id)
		}

		response := NewEchoResponse( /* response data */ )
		return response, nil
	}
}