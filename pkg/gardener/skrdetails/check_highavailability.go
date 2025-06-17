package skrdetails

func IsOdd(zones[]string) bool {
	return len(zones)%2 != 0

}

func IsHighAvailability(zones []string) bool {
	return len(zones) >= 3 && IsOdd(zones)
}
