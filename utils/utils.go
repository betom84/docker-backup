package utils

func Contains[C comparable](array []C, item C) bool {
	for _, i := range array {
		if i == item {
			return true
		}
	}

	return false
}

func IsEmpty(things ...string) bool {
	for _, t := range things {
		if len(t) == 0 {
			return true
		}
	}

	return false
}
