package main

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {
	Describe("Array difference", func() {
		It("returns all unique elements", func() {
			arr1 := []int{1, 2, 3, 4, 5}
			arr2 := []int{3, 4, 5, 6, 7}
			missing := []int{1, 2}
			additional := []int{6, 7}

			intEql := func(a, b *int) bool {
				return *a == *b
			}
			actualMissing, actualAdditional := difference(arr1, arr2, intEql)

			Expect(actualMissing).To(Equal(missing))
			Expect(actualAdditional).To(Equal(additional))
		})
	})
})
