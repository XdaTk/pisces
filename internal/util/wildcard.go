package util

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

func LongestCommonPrefix(a, b string) int {
	i := 0
	max := min(len(a), len(b))
	for i < max && a[i] == b[i] {
		i++
	}
	return i
}
