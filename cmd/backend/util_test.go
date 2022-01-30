package main

import "testing"

func TestIsInteger(t *testing.T) {
	assertCorrectMessage := func(t *testing.T, got bool, want bool) {
		t.Helper()
		if got != want {
			t.Errorf("got %t want %t", got, want)
		}
	}

	t.Run(`testing "6000"`, func(t *testing.T) {
		got := isInteger("6000")
		want := true
		assertCorrectMessage(t, got, want)
	})

	t.Run(`testing ""`, func(t *testing.T) {
		got := isInteger("")
		want := false
		assertCorrectMessage(t, got, want)
	})

	t.Run(`testing "test"`, func(t *testing.T) {
		got := isInteger("test")
		want := false
		assertCorrectMessage(t, got, want)
	})

	t.Run(`testing " 6000"`, func(t *testing.T) {
		got := isInteger(" 6000")
		want := false
		assertCorrectMessage(t, got, want)
	})

}
