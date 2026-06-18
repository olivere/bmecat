package bmecat2005_test

import (
	"strings"
	"testing"
)

func diffStrings(t testing.TB, want, have string) {
	t.Error("different output found")
	l1 := strings.Split(want, "\n")
	l2 := strings.Split(have, "\n")
	var r1, r2 []string
	if len(l1) > len(l2) {
		r1, r2 = l1, l2
	} else {
		r1, r2 = l2, l1
	}
	for i, a := range r1 {
		if i >= len(r2) {
			break
		} else if a != r2[i] {
			t.Logf("   %s\n", a)
			t.Logf("!= %s\n", r2[i])
		}
	}
	t.Fail()
}
