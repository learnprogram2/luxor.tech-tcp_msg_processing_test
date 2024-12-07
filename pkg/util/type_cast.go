package util

func IntValue(value interface{}) (int, bool) {
	switch value.(type) {
	case int:
		return value.(int), true
	case int64:
		return int(value.(int64)), true
	case float64: // JSON numbers are decoded as float64
		return int(value.(float64)), true
	}
	return 0, false
}

func StringValue(value interface{}) (string, bool) {
	val, ok := value.(string)
	return val, ok
}
