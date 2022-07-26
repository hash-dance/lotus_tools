package util

import "fmt"

// GetValueFromFields parse json value from string
// fields is []string(a.b.c),
// source is `{
//    "a": {
//        "b": {
//            "c": "hello"
//        }
//    }
// }`
// it will return `hello`, if no this field, return error
func GetValueFromFields(fields []string, source map[string]interface{}) (string, error) {
	if len(fields) == 1 {
		if v, ok := source[fields[0]]; ok {
			return fmt.Sprintf("%v", v), nil
		}
		return "", fmt.Errorf("no this field: %s", fields[0])
	}
	if next, ok := source[fields[0]].(map[string]interface{}); ok {
		return GetValueFromFields(fields[1:], next)
	}
	return "", fmt.Errorf("error parse field %s", fields[0])
}
