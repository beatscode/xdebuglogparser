package main

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestXxx(t *testing.T) {

	num := fmt.Sprintf("%f", float64(1.225788)-float64(1.225784)) //  4e-06
	f, _ := strconv.ParseFloat(num, 64)
	if f == 0.000004 {
		t.Log("Success")
	} else {
		//How come I don't get 0.000004
		t.Error("Not Equal", num)
	}

	if getFloat(f) == 0.000004 {
		t.Log("Success")
	} else {
		t.Error("Fail", getFloat(f))
	}
}

func TestParseIntFromStringWithNewline(t *testing.T) {
	value, _ := strconv.ParseInt(strings.Trim(`3375160
		`, ""), 10, 64)
	if value == 3375160 {
		t.Log("Success")
	} else {
		t.Error("Failed to Convert string to integer", value)
	}
}

func TestParseFloatFromStringWithPadding(t *testing.T) {
	str := "1.225800"
	num, _ := strconv.ParseFloat(str, 64)

	if num == 1.225800 {
		t.Log("Success")
	} else {
		t.Error("Invalid COnversion", num)
	}
}

func TestSliceCut(t *testing.T) {
	a := []string{"Alex", "Samson", "Leon", "Hamida"}

	//a = append(a[:3], a[2:]...)
	fmt.Println(a[0:2])

}
func getFloat(f float64) float64 {
	//fmt.Println("My Float:", f)
	return f
}
