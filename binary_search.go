package main

import (
	"fmt"
	"sort"
)

// Iterative binary search implementation
func binarySearchIterative(arr []int, target int) int {
	low := 0
	high := len(arr) - 1

	for low <= high {
		mid := (low + high) / 2
		
		if arr[mid] == target {
			return mid
		} else if arr[mid] < target {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}
	
	return -1 // Element not found
}

// Recursive binary search implementation
func binarySearchRecursive(arr []int, target int, low, high int) int {
	if low > high {
		return -1 // Element not found
	}
	
	mid := (low + high) / 2
	
	if arr[mid] == target {
		return mid
	} else if arr[mid] < target {
		return binarySearchRecursive(arr, target, mid+1, high)
	} else {
		return binarySearchRecursive(arr, target, low, mid-1)
	}
}

// Wrapper function for recursive implementation
func binarySearchRecursiveWrapper(arr []int, target int) int {
	return binarySearchRecursive(arr, target, 0, len(arr)-1)
}

// Generic binary search for any comparable type
func binarySearchGeneric[T comparable](arr []T, target T) int {
	low := 0
	high := len(arr) - 1
	
	for low <= high {
		mid := (low + high) / 2
		
		if arr[mid] == target {
			return mid
		} else if arr[mid] < target {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}
	
	return -1
}

func main() {
	// Example with integers
	sortedArray := []int{1, 3, 5, 7, 9, 11, 13, 15, 17, 19}
	
	fmt.Println("Sorted array:", sortedArray)
	
	// Test iterative search
	target := 7
	result := binarySearchIterative(sortedArray, target)
	if result != -1 {
		fmt.Printf("Element %d found at index %d (iterative)\n", target, result)
	} else {
		fmt.Printf("Element %d not found (iterative)\n", target)
	}
	
	// Test recursive search
	result = binarySearchRecursiveWrapper(sortedArray, target)
	if result != -1 {
		fmt.Printf("Element %d found at index %d (recursive)\n", target, result)
	} else {
		fmt.Printf("Element %d not found (recursive)\n", target)
	}
	
	// Test with non-existent element
	target = 8
	result = binarySearchIterative(sortedArray, target)
	if result != -1 {
		fmt.Printf("Element %d found at index %d\n", target, result)
	} else {
		fmt.Printf("Element %d not found\n", target)
	}
	
	// Example with strings
	stringArray := []string{"apple", "banana", "cherry", "date", "elderberry"}
	
	fmt.Println("\nString array:", stringArray)
	
	targetString := "cherry"
	result = binarySearchGeneric(stringArray, targetString)
	if result != -1 {
		fmt.Printf("Element %s found at index %d (generic)\n", targetString, result)
	} else {
		fmt.Printf("Element %s not found (generic)\n", targetString)
	}
	
	// Using Go's built-in sort.Search for integers
	fmt.Println("\nUsing Go's built-in sort.Search:")
	index := sort.Search(len(sortedArray), func(i int) bool {
		return sortedArray[i] >= target
	})
	if index < len(sortedArray) && sortedArray[index] == target {
		fmt.Printf("Element %d found at index %d (using sort.Search)\n", target, index)
	} else {
		fmt.Printf("Element %d not found (using sort.Search)\n", target)
	}
}