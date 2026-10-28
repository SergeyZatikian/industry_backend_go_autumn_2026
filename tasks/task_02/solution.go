package main

func rotateRunes(s string, shift int) string {
	runes := []rune(s)

	n := len(runes)

	if n == 0 {
		return ""
	}

	shift = shift % n

	if shift < 0 {
		shift += n
	}

	result := make([]rune, 0, n)

	result = append(result, runes[shift:]...)
	result = append(result, runes[:shift]...)

	return string(result)
}
