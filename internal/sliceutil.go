package internal

// FilterMap applies a filter and a map function to a slice of type T and returns a new slice of type U.
func FilterMap[T any, U any](input []T, filter func(T) bool, mapper func(T) U) []U {
	result := make([]U, 0, len(input))
	for _, v := range input {
		if filter(v) {
			result = append(result, mapper(v))
		}
	}
	return result
}

// Filter returns a new slice containing only elements that pass the filter.
func Filter[T any](input []T, filter func(T) bool) []T {
	result := make([]T, 0, len(input))
	for _, v := range input {
		if filter(v) {
			result = append(result, v)
		}
	}
	return result
}

// Map returns a new slice with the result of applying mapper to each element.
func Map[T any, U any](input []T, mapper func(T) U) []U {
	result := make([]U, 0, len(input))
	for _, v := range input {
		result = append(result, mapper(v))
	}
	return result
}
