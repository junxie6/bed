package buffer

import (
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestBufferEmpty(t *testing.T) {
	b := NewBuffer(strings.NewReader(""))

	p := make([]byte, 10)
	n, err := b.Read(p)
	if err != io.EOF {
		t.Errorf("err should be EOF but got: %v", err)
	}
	if n != 0 {
		t.Errorf("n should be 0 but got: %d", n)
	}

	l, err := b.Len()
	if err != nil {
		t.Errorf("err should be nil but got: %v", err)
	}
	if l != 0 {
		t.Errorf("l should be 0 but got: %d", l)
	}
}

func TestBuffer(t *testing.T) {
	b := NewBuffer(strings.NewReader("0123456789abcdef"))

	p := make([]byte, 8)
	n, err := b.Read(p)
	if err != nil {
		t.Errorf("err should be nil but got: %v", err)
	}
	if n != 8 {
		t.Errorf("n should be 8 but got: %d", n)
	}
	if string(p) != "01234567" {
		t.Errorf("p should be 01234567 but got: %s", string(p))
	}

	l, err := b.Len()
	if err != nil {
		t.Errorf("err should be nil but got: %v", err)
	}
	if l != 16 {
		t.Errorf("l should be 16 but got: %d", l)
	}

	_, err = b.Seek(4, io.SeekStart)
	if err != nil {
		t.Errorf("err should be nil but got: %v", err)
	}

	n, err = b.Read(p)
	if err != nil {
		t.Errorf("err should be EOF but got: %v", err)
	}
	if n != 8 {
		t.Errorf("n should be 8 but got: %d", n)
	}
	if string(p) != "456789ab" {
		t.Errorf("p should be 456789ab but got: %s", string(p))
	}

	_, err = b.Seek(-4, io.SeekEnd)
	if err != nil {
		t.Errorf("err should be nil but got: %v", err)
	}

	n, err = b.Read(p)
	if err != nil {
		t.Errorf("err should be EOF but got: %v", err)
	}
	if n != 4 {
		t.Errorf("n should be 4 but got: %d", n)
	}
	if string(p) != "cdef89ab" {
		t.Errorf("p should be cdef89ab but got: %s", string(p))
	}
}

func TestBufferInsert(t *testing.T) {
	b := NewBuffer(strings.NewReader("0123456789abcdef"))

	tests := []struct {
		index    int64
		b        byte
		offset   int64
		expected string
		len      int64
	}{
		{0, 0x39, 0, "90123456", 17},
		{0, 0x38, 0, "89012345", 18},
		{4, 0x37, 0, "89017234", 19},
		{8, 0x30, 3, "17234056", 20},
		{9, 0x31, 3, "17234015", 21},
		{9, 0x32, 4, "72340215", 22},
		{23, 0x39, 19, "def9\x00\x00\x00\x00", 23},
		{23, 0x38, 19, "def89\x00\x00\x00", 24},
	}

	for _, test := range tests {
		b.Insert(test.index, test.b)
		p := make([]byte, 8)

		_, err := b.Seek(test.offset, io.SeekStart)
		if err != nil {
			t.Errorf("err should be nil but got: %v", err)
		}

		n, err := b.Read(p)
		if err != nil && err != io.EOF {
			t.Errorf("err should be nil or io.EOF but got: %v", err)
		}
		if n != len(strings.TrimRight(test.expected, "\x00")) {
			t.Errorf("n should be %d but got: %d", len(strings.TrimRight(test.expected, "\x00")), n)
		}
		if string(p) != test.expected {
			t.Errorf("p should be %s but got: %s", test.expected, string(p))
		}

		l, err := b.Len()
		if err != nil {
			t.Errorf("err should be nil but got: %v", err)
		}
		if l != test.len {
			t.Errorf("l should be %d but got: %d", test.len, l)
		}
	}

	eis := b.EditedIndices()
	expected := []int64{0, 2, 4, 5, 8, 11, 23, 25}
	if !reflect.DeepEqual(eis, expected) {
		t.Errorf("edited indices should be %v but got: %v", expected, eis)
	}

	if len(b.rrs) != 8 {
		t.Errorf("len(b.rrs) should be 8 but got: %d", len(b.rrs))
	}
}

func TestBufferReplace(t *testing.T) {
	b := NewBuffer(strings.NewReader("0123456789abcdef"))

	tests := []struct {
		index    int64
		b        byte
		offset   int64
		expected string
		len      int64
	}{
		{0, 0x39, 0, "91234567", 16},
		{0, 0x38, 0, "81234567", 16},
		{1, 0x37, 0, "87234567", 16},
		{5, 0x30, 0, "87234067", 16},
		{4, 0x31, 0, "87231067", 16},
		{3, 0x30, 0, "87201067", 16},
		{2, 0x31, 0, "87101067", 16},
		{16, 0x31, 9, "9abcdef1", 16},
		{15, 0x30, 9, "9abcde01", 16},
	}

	for _, test := range tests {
		b.Replace(test.index, test.b)
		p := make([]byte, 8)

		_, err := b.Seek(test.offset, io.SeekStart)
		if err != nil {
			t.Errorf("err should be nil but got: %v", err)
		}

		n, err := b.Read(p)
		if err != nil && err != io.EOF {
			t.Errorf("err should be nil or io.EOF but got: %v", err)
		}
		if n != 8 {
			t.Errorf("n should be 8 but got: %d", n)
		}
		if string(p) != test.expected {
			t.Errorf("p should be %s but got: %s", test.expected, string(p))
		}

		l, err := b.Len()
		if err != nil {
			t.Errorf("err should be nil but got: %v", err)
		}
		if l != test.len {
			t.Errorf("l should be %d but got: %d", test.len, l)
		}
	}

	eis := b.EditedIndices()
	expected := []int64{0, 6, 15, 17}
	if !reflect.DeepEqual(eis, expected) {
		t.Errorf("edited indices should be %v but got: %v", expected, eis)
	}

	if len(b.rrs) != 4 {
		t.Errorf("len(b.rrs) should be 4 but got: %d", len(b.rrs))
	}
}

func TestBufferDelete(t *testing.T) {
	b := NewBuffer(strings.NewReader("0123456789abcdef"))

	tests := []struct {
		index    int64
		b        byte
		offset   int64
		expected string
		len      int64
	}{
		{4, 0x00, 0, "01235678", 15},
		{3, 0x00, 0, "01256789", 14},
		{6, 0x00, 0, "0125679a", 13},
		{0, 0x00, 0, "125679ab", 12},
		{4, 0x39, 0, "1256979a", 13},
		{5, 0x38, 0, "12569879", 14},
		{3, 0x00, 0, "1259879a", 13},
		{4, 0x00, 0, "125979ab", 12},
		{3, 0x00, 0, "12579abc", 11},
		{8, 0x39, 4, "9abc9def", 12},
		{8, 0x38, 4, "9abc89de", 13},
		{8, 0x00, 4, "9abc9def", 12},
		{8, 0x00, 4, "9abcdef\x00", 11},
	}

	for _, test := range tests {
		if test.b == 0x00 {
			b.Delete(test.index)
		} else {
			b.Insert(test.index, test.b)
		}
		p := make([]byte, 8)

		_, err := b.Seek(test.offset, io.SeekStart)
		if err != nil {
			t.Errorf("err should be nil but got: %v", err)
		}

		n, err := b.Read(p)
		if err != nil && err != io.EOF {
			t.Errorf("err should be nil or io.EOF but got: %v", err)
		}
		if n != len(strings.TrimRight(test.expected, "\x00")) {
			t.Errorf("n should be %d but got: %d", len(strings.TrimRight(test.expected, "\x00")), n)
		}
		if string(p) != test.expected {
			t.Errorf("p should be %s but got: %s", test.expected, string(p))
		}

		l, err := b.Len()
		if err != nil {
			t.Errorf("err should be nil but got: %v", err)
		}
		if l != test.len {
			t.Errorf("l should be %d but got: %d", test.len, l)
		}
	}

	eis := b.EditedIndices()
	expected := []int64{}
	if !reflect.DeepEqual(eis, expected) {
		t.Errorf("edited indices should be %v but got: %v", expected, eis)
	}

	if len(b.rrs) != 4 {
		t.Errorf("len(b.rrs) should be 4 but got: %d", len(b.rrs))
	}
}