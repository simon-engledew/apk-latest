//go:generate curl -LO http://dl-cdn.alpinelinux.org/alpine/v3.8/main/x86_64/APKINDEX.tar.gz

package main

import (
	"os"
	"testing"
)

func TestBasic(t *testing.T) {
	handle, err := os.Open("APKINDEX.tar.gz")
	if err != nil {
		t.Error(err)
	}

	scanIndex(handle, func(apk *apk) error {
		return nil
	})
}
