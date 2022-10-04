package util;

func ArrayContains(arr []interface{}, str string) bool {
	for _, a := range arr {
		if str == a {
			return true
		}
	}

	return false
}

