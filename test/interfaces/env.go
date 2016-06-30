package interfaces

// Env is used to create a generated env mock using mockery
type Env interface {
	Get(key string) string
}
