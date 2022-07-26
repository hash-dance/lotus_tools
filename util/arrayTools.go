/*Package util array tools
 */
package util

import (
	"fmt"
	"strconv"
	"strings"
)

// IntArray2String []int to string, sep with delim
func IntArray2String(arr []int, delim string) string {
	return strings.Trim(strings.Join(strings.Fields(fmt.Sprint(arr)), delim), "[]")
}

// String2IntArray string to []int
func String2IntArray(arr string, delim string) []int {
	if arr == "" {
		return []int{}
	}
	a := strings.Split(arr, delim)
	b := make([]int, len(a))
	for i, v := range a {
		val, err := strconv.Atoi(v)
		if err == nil {
			b[i] = val
		}
	}
	return b
}

func ArrInclude(arr []string, sub string) bool {
	for _, a := range arr {
		if a == sub {
			return true
		}
	}
	return false
}