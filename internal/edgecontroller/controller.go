package edgecontroller

type Client interface {
	// IsConnected return true if the client is connected to the server.
	IsConnected() bool
}
