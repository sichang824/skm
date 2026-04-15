package requestctx

// Key is a typed key for storing values in context
type Key string

const (
	// UserKey stores user information for the current request
	UserKey Key = "user"
)
