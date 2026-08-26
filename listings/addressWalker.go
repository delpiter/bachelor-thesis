func serverListWalk(
	done <-chan struct{},
    serverIp []string
) (<-chan string, <-chan error) {
	paths := make(chan string)
	errc := make(chan error, 1)
	go func() {
		defer close(paths)

		for i := range serverIp {
			select {
			case paths <- serverIp[i]:
			case <-done:
				errc <- errors.New("Request cancelled")
				return
			}
		}
		errc <- nil
	}()
	return paths, errc
}