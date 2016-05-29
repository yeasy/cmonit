package util

/*It's pity for go having no such useful methods.*/
//Avg return the minimum element of a float64 array
func Avg(a []float64) (float64) {
	return Sum(a)/float64(len(a))
}

//Min return the minimum element of a float64 array
func Min(a []float64) (min float64) {
	min = a[0]
	for _, v := range a {
		if min > v {
			min = v
		}
	}
	return
}

//Max return the maximum element of a float64 array
func Max(a []float64) (max float64) {
	max = a[0]
	for _, v := range a {
		if max < v {
			max = v
		}
	}
	return
}

//Sum return the sum result of a float64 array
func Sum(a []float64) (sum float64) {
	for _, v := range a {
		sum += v
	}
	return
}
