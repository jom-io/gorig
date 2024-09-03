package geo

import "math"

// Haversine 计算两个经纬度点之间的距离
func Haversine(lonFrom, latFrom, lonTo, latTo float64) float64 {
	var r float64 = 6371 // 地球半径，单位为公里
	φ1 := latFrom * math.Pi / 180
	φ2 := latTo * math.Pi / 180
	Δφ := (latTo - latFrom) * math.Pi / 180
	Δλ := (lonTo - lonFrom) * math.Pi / 180

	a := math.Sin(Δφ/2)*math.Sin(Δφ/2) +
		math.Cos(φ1)*math.Cos(φ2)*
			math.Sin(Δλ/2)*math.Sin(Δλ/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := r * c
	return distance
}
