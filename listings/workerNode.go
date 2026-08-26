func worker(
	done <-chan struct{},
	servers <-chan string,
	c chan<- result,
	targetIp string,
	fetchLatency func(serverIp, targetIp string) (Result, error),
) {
	for serverIp := range servers {
		latencyResult, err := fetchLatency(serverIp, targetIp)
		select {
		case c <- result{latencyResult.Address, latencyResult.Value, err}:
		case <-done:
			return
		}
	}
}