package math

func Ceil(x float64) int {
	i := int(x)
	if float64(i) < x {
		return i + 1
	}
	return i
}
