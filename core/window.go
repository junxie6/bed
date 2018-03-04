package core

import (
	"io"
	"strconv"

	"github.com/itchyny/bed/buffer"
	"github.com/itchyny/bed/util"
)

// Window represents an editor window.
type Window struct {
	buffer      *buffer.Buffer
	name        string
	height      int64
	width       int64
	offset      int64
	cursor      int64
	length      int64
	stack       []position
	append      bool
	replaceByte bool
	extending   bool
	pending     bool
	pendingByte byte
}

type position struct {
	cursor int64
	offset int64
}

// NewWindow creates a new editor window.
func NewWindow(r io.ReadSeeker, name string, height, width int64) (*Window, error) {
	buffer := buffer.NewBuffer(r)
	length, err := buffer.Len()
	if err != nil {
		return nil, err
	}
	return &Window{
		buffer: buffer,
		name:   name,
		height: height,
		width:  width,
		length: length,
	}, nil
}

func (w *Window) readBytes(pos int64, len int) (int, []byte, error) {
	bytes := make([]byte, len)
	_, err := w.buffer.Seek(pos, io.SeekStart)
	if err != nil {
		return 0, bytes, err
	}
	n, err := w.buffer.Read(bytes)
	if err != nil && err != io.EOF {
		return 0, bytes, err
	}
	return n, bytes, nil
}

// State returns the current state of the buffer.
func (w *Window) State() (State, error) {
	n, bytes, err := w.readBytes(w.offset, int(w.height*w.width))
	if err != nil {
		return State{}, err
	}
	return State{
		Name:          w.name,
		Width:         int(w.width),
		Offset:        w.offset,
		Cursor:        w.cursor,
		Bytes:         bytes,
		Size:          n,
		Length:        w.length,
		Pending:       w.pending,
		PendingByte:   w.pendingByte,
		EditedIndices: w.buffer.EditedIndices(),
	}, nil
}

func (w *Window) cursorUp(count int64) {
	w.cursor -= util.MinInt64(util.MaxInt64(count, 1), w.cursor/w.width) * w.width
	if w.cursor < w.offset {
		w.offset = w.cursor / w.width * w.width
	}
}

func (w *Window) cursorDown(count int64) {
	w.cursor += util.MinInt64(
		util.MinInt64(util.MaxInt64(count, 1), (util.MaxInt64(w.length, 1)-1)/w.width-w.cursor/w.width)*w.width,
		util.MaxInt64(w.length, 1)-1-w.cursor)
	if w.cursor >= w.offset+w.height*w.width {
		w.offset = (w.cursor - w.height*w.width + w.width) / w.width * w.width
	}
}

func (w *Window) cursorLeft(count int64) {
	w.cursor -= util.MinInt64(util.MaxInt64(count, 1), w.cursor%w.width)
}

func (w *Window) cursorRight(count int64) {
	w.cursor += util.MinInt64(util.MinInt64(util.MaxInt64(count, 1), w.width-1-w.cursor%w.width), util.MaxInt64(w.length, 1)-1-w.cursor)
}

func (w *Window) cursorPrev(count int64) {
	w.cursor -= util.MinInt64(util.MaxInt64(count, 1), w.cursor)
	if w.cursor < w.offset {
		w.offset = w.cursor / w.width * w.width
	}
}

func (w *Window) cursorNext(count int64) {
	w.cursor += util.MinInt64(util.MaxInt64(count, 1), util.MaxInt64(w.length, 1)-1-w.cursor)
	if w.cursor >= w.offset+w.height*w.width {
		w.offset = (w.cursor - w.height*w.width + w.width) / w.width * w.width
	}
}

func (w *Window) cursorHead(_ int64) {
	w.cursor -= w.cursor % w.width
}

func (w *Window) cursorEnd(count int64) {
	w.cursor = util.MinInt64((w.cursor/w.width+util.MaxInt64(count, 1))*w.width-1, util.MaxInt64(w.length, 1)-1)
	if w.cursor >= w.offset+w.height*w.width {
		w.offset = (w.cursor - w.height*w.width + w.width) / w.width * w.width
	}
}

func (w *Window) scrollUp(count int64) {
	w.offset -= util.MinInt64(util.MaxInt64(count, 1), w.offset/w.width) * w.width
	if w.cursor >= w.offset+w.height*w.width {
		w.cursor -= ((w.cursor-w.offset-w.height*w.width)/w.width + 1) * w.width
	}
}

func (w *Window) scrollDown(count int64) {
	h := (util.MaxInt64(w.length, 1)+w.width-1)/w.width - w.height
	w.offset += util.MinInt64(util.MaxInt64(count, 1), h-w.offset/w.width) * w.width
	if w.cursor < w.offset {
		w.cursor += util.MinInt64((w.offset-w.cursor+w.width-1)/w.width*w.width, util.MaxInt64(w.length, 1)-1-w.cursor)
	}
}

func (w *Window) pageUp() {
	w.offset = util.MaxInt64(w.offset-(w.height-2)*w.width, 0)
	if w.offset == 0 {
		w.cursor = 0
	} else if w.cursor >= w.offset+w.height*w.width {
		w.cursor = w.offset + (w.height-1)*w.width
	}
}

func (w *Window) pageDown() {
	offset := util.MaxInt64(((w.length+w.width-1)/w.width-w.height)*w.width, 0)
	w.offset = util.MinInt64(w.offset+(w.height-2)*w.width, offset)
	if w.cursor < w.offset {
		w.cursor = w.offset
	} else if w.offset == offset {
		w.cursor = ((util.MaxInt64(w.length, 1)+w.width-1)/w.width - 1) * w.width
	}
}

func (w *Window) pageUpHalf() {
	w.offset = util.MaxInt64(w.offset-util.MaxInt64(w.height/2, 1)*w.width, 0)
	if w.offset == 0 {
		w.cursor = 0
	} else if w.cursor >= w.offset+w.height*w.width {
		w.cursor = w.offset + (w.height-1)*w.width
	}
}

func (w *Window) pageDownHalf() {
	offset := util.MaxInt64(((w.length+w.width-1)/w.width-w.height)*w.width, 0)
	w.offset = util.MinInt64(w.offset+util.MaxInt64(w.height/2, 1)*w.width, offset)
	if w.cursor < w.offset {
		w.cursor = w.offset
	} else if w.offset == offset {
		w.cursor = ((util.MaxInt64(w.length, 1)+w.width-1)/w.width - 1) * w.width
	}
}

func (w *Window) pageTop() {
	w.offset = 0
	w.cursor = 0
}

func (w *Window) pageEnd() {
	w.offset = util.MaxInt64(((w.length+w.width-1)/w.width-w.height)*w.width, 0)
	w.cursor = ((util.MaxInt64(w.length, 1)+w.width-1)/w.width - 1) * w.width
}

func isDigit(b byte) bool {
	return '\x30' <= b && b <= '\x39'
}

func isWhite(b byte) bool {
	return b == '\x00' || b == '\x09' || b == '\x0a' || b == '\x0d' || b == '\x20'
}

func (w *Window) jumpTo() {
	s := 50
	_, bytes, err := w.readBytes(util.MaxInt64(w.cursor-int64(s), 0), 2*s)
	if err != nil {
		return
	}
	var i, j int
	for i = s; i < 2*s && isWhite(bytes[i]); i++ {
	}
	if i == 2*s || !isDigit(bytes[i]) {
		return
	}
	for ; 0 < i && isDigit(bytes[i-1]); i-- {
	}
	for j = i; j < 2*s && isDigit(bytes[j]); j++ {
	}
	if j == 2*s {
		return
	}
	offset, _ := strconv.ParseInt(string(bytes[i:j]), 10, 64)
	if offset <= 0 || w.length <= offset {
		return
	}
	w.stack = append(w.stack, position{w.cursor, w.offset})
	w.cursor = offset
	w.offset = util.MaxInt64(offset-offset%w.width-util.MaxInt64(w.height/3, 0)*w.width, 0)
}

func (w *Window) jumpBack() {
	if len(w.stack) == 0 {
		return
	}
	w.cursor = w.stack[len(w.stack)-1].cursor
	w.offset = w.stack[len(w.stack)-1].offset
	w.stack = w.stack[:len(w.stack)-1]
}

func (w *Window) deleteByte(count int64) {
	if w.length == 0 {
		return
	}
	cnt := int(util.MinInt64(util.MinInt64(util.MaxInt64(count, 1), w.width-w.cursor%w.width), w.length-w.cursor))
	for i := 0; i < cnt; i++ {
		w.buffer.Delete(w.cursor)
		w.length--
		if w.cursor == w.length && w.cursor > 0 {
			w.cursor--
		}
	}
}

func (w *Window) deletePrevByte(count int64) {
	cnt := int(util.MinInt64(util.MaxInt64(count, 1), w.cursor%w.width))
	for i := 0; i < cnt; i++ {
		w.buffer.Delete(w.cursor - 1)
		w.cursor--
		w.length--
	}
}

func (w *Window) increment(count int64) {
	_, bytes, err := w.readBytes(w.cursor, 1)
	if err != nil {
		return
	}
	w.buffer.Replace(w.cursor, bytes[0]+byte(util.MaxInt64(count, 1)%256))
	if w.length == 0 {
		w.length++
	}
}

func (w *Window) decrement(count int64) {
	_, bytes, err := w.readBytes(w.cursor, 1)
	if err != nil {
		return
	}
	w.buffer.Replace(w.cursor, bytes[0]-byte(util.MaxInt64(count, 1)%256))
	if w.length == 0 {
		w.length++
	}
}

func (w *Window) startInsert() {
	w.append = w.length == 0
	w.extending = false
	w.pending = false
}

func (w *Window) startInsertHead() {
	w.cursorHead(0)
	w.append = w.length == 0
	w.extending = false
	w.pending = false
}

func (w *Window) startAppend() {
	w.append = true
	w.extending = false
	w.pending = false
	if w.length > 0 {
		w.cursor++
	}
	if w.cursor == w.length {
		w.extending = true
		w.length++
	}
	if w.cursor >= w.offset+w.height*w.width {
		w.offset = (w.cursor - w.height*w.width + w.width) / w.width * w.width
	}
}

func (w *Window) startAppendEnd() {
	w.cursorEnd(0)
	w.startAppend()
}

func (w *Window) startReplaceByte() {
	w.replaceByte = true
	w.append = false
	w.extending = false
	w.pending = false
}

func (w *Window) startReplace() {
	w.replaceByte = false
	w.append = false
	w.extending = false
	w.pending = false
}

func (w *Window) exitInsert() {
	w.pending = false
	if w.append {
		if w.extending && w.length > 0 {
			w.length--
		}
		if w.cursor > 0 {
			w.cursor--
		}
	}
}

func (w *Window) insert(mode Mode, b byte) {
	if w.pending {
		switch mode {
		case ModeInsert:
			w.buffer.Insert(w.cursor, w.pendingByte|b)
			w.cursor++
			w.length++
		case ModeReplace:
			w.buffer.Replace(w.cursor, w.pendingByte|b)
			if w.length == 0 {
				w.length++
			}
			if w.replaceByte {
				w.exitInsert()
			} else {
				w.cursor++
				if w.cursor == w.length {
					w.append = true
					w.extending = true
					w.length++
				}
			}
		}
		if w.cursor >= w.offset+w.height*w.width {
			w.offset = (w.cursor - w.height*w.width + w.width) / w.width * w.width
		}
		w.pending = false
		w.pendingByte = '\x00'
	} else {
		w.pending = true
		w.pendingByte = b << 4
	}
}

func (w *Window) insert0(mode Mode) {
	w.insert(mode, 0x00)
}

func (w *Window) insert1(mode Mode) {
	w.insert(mode, 0x01)
}

func (w *Window) insert2(mode Mode) {
	w.insert(mode, 0x02)
}

func (w *Window) insert3(mode Mode) {
	w.insert(mode, 0x03)
}

func (w *Window) insert4(mode Mode) {
	w.insert(mode, 0x04)
}

func (w *Window) insert5(mode Mode) {
	w.insert(mode, 0x05)
}

func (w *Window) insert6(mode Mode) {
	w.insert(mode, 0x06)
}

func (w *Window) insert7(mode Mode) {
	w.insert(mode, 0x07)
}

func (w *Window) insert8(mode Mode) {
	w.insert(mode, 0x08)
}

func (w *Window) insert9(mode Mode) {
	w.insert(mode, 0x09)
}

func (w *Window) insertA(mode Mode) {
	w.insert(mode, 0x0a)
}

func (w *Window) insertB(mode Mode) {
	w.insert(mode, 0x0b)
}

func (w *Window) insertC(mode Mode) {
	w.insert(mode, 0x0c)
}

func (w *Window) insertD(mode Mode) {
	w.insert(mode, 0x0d)
}

func (w *Window) insertE(mode Mode) {
	w.insert(mode, 0x0e)
}

func (w *Window) insertF(mode Mode) {
	w.insert(mode, 0x0f)
}

func (w *Window) backspace() {
	if w.pending {
		w.pending = false
		w.pendingByte = '\x00'
	} else if w.cursor > 0 {
		w.buffer.Delete(w.cursor - 1)
		w.cursor--
		w.length--
	}
}