package kerrsource

import (
	"bytes"
	"math/rand"
)

func GetRandomId() string {
	var result bytes.Buffer
	var temp string
	for i := 0; i < 10; {
		if string(randInt(65, 90)) != temp {
			temp = string(randInt(65, 90))
			result.WriteString(temp)
			i++
		}
	}
	return result.String()
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}

func IsId(s string) bool {
	if len(s) != 10 {
		return false
	}
	for _, r := range s {
		if int(r) > 90 || int(r) < 65 {
			return false
		}
	}
	return true
}
