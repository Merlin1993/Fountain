package log

// Ctx is a map of key/value pairs to pass as context to a log function
// Use this only if you really need greater safety around the arguments you pass
// to the logging functions.
type CtxMap map[string]interface{}

func (c CtxMap) toArray() []interface{} {
	arr := make([]interface{}, len(c) *2)

	i := 0
	for k, v := range c {
		arr[i] = k
		arr[i+1] = v
		i += 2
	}

	return arr
}
