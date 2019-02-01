package gopcp_rpc

import (
	"fmt"
	"testing"
)

func assertEqual(t *testing.T, expect interface{}, actual interface{}, message string) {
	if expect == actual {
		return
	}
	if len(message) == 0 {
		message = fmt.Sprintf("expect %v !=  actual %v", expect, actual)
	}
	t.Fatal(message)
}

func assertEqualBytes(t *testing.T, bytes1 []byte, bytes2 []byte) {
	if len(bytes1) != len(bytes2) {
		t.Errorf("bytes are not equal. bytes1=%v, byte2=%v", bytes1, bytes2)
	}

	for idx, b := range bytes1 {
		if b != bytes2[idx] {
			t.Errorf("bytes are not equal. bytes1=%v, byte2=%v", bytes1, bytes2)
		}
	}
}

func TestBase(t *testing.T) {
	assertEqualBytes(t, TextToPkt("hello"), []byte{0, 5, 0, 0, 0, 104, 101, 108, 108, 111})
}

func TestGetPktText(t *testing.T) {
	text := "hello, world! Should be longerrrrrrrrrrrrrrrrr."
	p := GetPackageProtocol()

	for i := 1; i < 1000; i++ {
		pkt := TextToPkt(text)
		part1 := pkt[0:10]
		part2 := pkt[10:]

		r1 := p.GetPktText(part1)
		r2 := p.GetPktText(part2)

		assertEqual(t, len(r1), 0, "")
		assertEqual(t, len(r2), 1, "")
		assertEqual(t, r2[0], text, "")
	}
}
