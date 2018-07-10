// This file has one signicant function, 'Bar()' that converts a slice of
// intervals spanning a 24-hour period into a textual bar that can be printed.
// The algorithm it uses for doing this is:
// 1. break the day up into 60 "characters" each of which represents 24 minutes,
//    and then break the character up into 8 bits, each representing 3 minutes
// 2. A bit is "on" if most of its 3 minutes is covered by intervals in the
//    slice of intervals, and off otherwise
// 3. Once all the bits in a character have been determined, compare it to each
//    of the bytes in 'blockMask' below, and choose the blockMask byte that is
//    bitwise closest.
// 4. Each of the bytes maps to a unicode character (possibly in inverted video
//    mode--e.g. a partially-filled left box that has been inverted to become a
//    partially-filled right box). Print the control character/unicode character
//    corresponding to the blockMask byte from (3).

package main

import (
	"bytes"
	"fmt"
	"time"

	"github.com/msteffen/golang-time-tracker/api"
)

// Note that the unicode box drawing characters look like:
// full box -> left eighth box
// 0x2588 ...  0x258f
// █ ▉ ▊ ▋ ▌ ▍ ▎ ▏
const fullBlock = 0x2588

// Also used
const lightVerticalLine = 0x2502 // = [│], about 1/8
const thickVerticalLine = 0x2503 // = [┃], about 3/8

var blockMask = [...]byte{
	// 0 - off, 1 - full
	0x00, 0xff,

	// [2-8] left boxes:
	0xfe, // 11111110
	0xfc, // 11111100
	0xf8, // 11111000
	0xf0, // 11110000
	0xe0, // 11100000
	0xc0, // 11000000
	0x80, // 10000000

	// [9-15] right boxes
	0x7f, // 01111111
	0x3f, // 00111111
	0x1f, // 00011111
	0x0f, // 00001111
	0x07, // 00000111
	0x03, // 00000011
	0x01, // 00000001

	// [16-26] thin vertical line
	0x40, // 01000000
	0x20, // 00100000
	0x10, // 00010000
	0x08, // 00001000
	0x04, // 00000100
	0x02, // 00000010
	0x60, // 01100000
	0x30, // 00110000
	0x18, // 00011000
	0x0c, // 00001100
	0x06, // 00000110

	// [27-37] inverted thin vertical line
	0xbf, // 10111111
	0xdf, // 11011111
	0xef, // 11101111
	0xf7, // 11110111
	0xfb, // 11111011
	0xfd, // 11111101
	0x9f, // 10011111
	0xcf, // 11001111
	0xe7, // 11100111
	0xf3, // 11110011
	0xf9, // 11111001

	// [38-47] thick vertical line
	0x70, // 01110000
	0x78, // 01111000
	0x7c, // 01111100
	0x7e, // 01111110
	0x38, // 00111000
	0x3c, // 00111100
	0x3e, // 00111110
	0x1c, // 00011100
	0x1e, // 00011110
	0x0e, // 00001110

	// [48-57] inverted thick vertical line
	0x8f, // 10001111
	0x87, // 10000111
	0x83, // 10000011
	0x81, // 10000001
	0xc7, // 11000111
	0xc3, // 11000011
	0xc1, // 11000001
	0xe3, // 11100011
	0xe1, // 11100001
	0xf1, // 11110001
}

// bits counts the number of ones in 'c'
func bits(c byte) byte {
	for i, m := range []byte{0x55, 0x33, 0x0f} {
		var p byte = 1 << byte(i)
		c = ((c >> p) & m) + (c & m)
	}
	return c
}

func Leq(l, r time.Time) bool {
	return l.Before(r) || l.Equal(r)
}

func MaxT(l, r time.Time) time.Time {
	if l.Before(r) {
		return r
	}
	return l
}

func MinT(l, r time.Time) time.Time {
	if l.Before(r) {
		return l
	}
	return r
}

// eighths rounds the given duration in seconds to the nearest 1/8 of 24 minutes
// The result is a rune so that it can be used to do unicode arithmetic
// (e.g. fullBlock + x)
func eighths(duration int64) rune {
	// 1/8 of (24 * 60) seconds = 180. (x+90)/180 => round to the nearest eighth
	return rune((duration + 90) / 180)
}

type barOp struct {
	// buf contains result of computing the day's bar
	buf bytes.Buffer

	// whether the buffer has had any block characters written to it yet
	empty bool

	// whether the buffer has set the terminal to be inverted
	inverted bool
}

func newBarOp() *barOp {
	op := &barOp{
		empty: true,
	}
	op.buf.WriteByte('[')
	return op
}

// writeInverted is a helper function that writes 'r' to b.buf with inverted
// colors (i.e. if colors are already inverted, it just writes 'r')
func (b *barOp) writeInverted(r rune) {
	if b.empty || !b.inverted {
		// SGR code -- 7 = inverted, 33 = set fg to yellow
		b.buf.Write([]byte{033, '[', '7', ';', '3', '3', 'm'})
		b.empty = false
		b.inverted = true
	}
	b.buf.WriteRune(r)
}

// writeInverted is a helper function that writes 'r' to b.buf with non-inverted
// colors (if colors are already normal, it just writes 'r')
func (b *barOp) writeNormal(r rune) {
	if b.empty {
		// set fg to yellow (colors are already normal)
		b.buf.Write([]byte{033, '[', '3', '3', 'm'})
		b.empty = false
	} else if b.inverted {
		// SGR code -- 0 = normal, 33 = set fg to yellow
		b.buf.Write([]byte{033, '[', '0', ';', '3', '3', 'm'})
		b.inverted = false
	}
	b.buf.WriteRune(r)
}

// finish adds the necessary trailing characters to b.buf and returns it as a
// string
func (b *barOp) finish() string {
	// reset colors completely and close with ']'
	b.buf.Write([]byte{033, '[', 'm', ']'})
	return b.buf.String()
}

func (b *barOp) put(idx int) {
	i := rune(idx)
	switch {
	case i == 0:
		// 0 - off
		b.writeInverted(fullBlock)
	case i == 1:
		// 1 - full
		b.writeNormal(fullBlock)
	case i <= 8:
		// [2-8] left boxes:
		b.writeNormal(fullBlock + i - 1)
	case i <= 15:
		// [9-15] right boxes
		b.writeInverted(fullBlock + 16 - i)
	case i <= 26:
		// [16-26] thin vertical line
		b.writeNormal(lightVerticalLine)
	case i <= 37:
		// [27-37] inverted thin vertical line
		b.writeInverted(lightVerticalLine)
	case i <= 47:
		// [38-47] thick vertical line
		b.writeNormal(thickVerticalLine)
	case i <= 57:
		// [48-57] inverted thick vertical line
		b.writeInverted(thickVerticalLine)
	}
}

var emptyBar = func() string {
	op := newBarOp()
	for i := 0; i < 60; i++ {
		op.writeInverted(fullBlock)
	}
	return op.finish()
}()

// Bar generates a bar containing a day's worth of intervals (for raw 't' cmd)
func Bar(morning time.Time, intervals []api.Interval) (res string) {
	if len(intervals) == 0 {
		return emptyBar // special case; no intervals
	}

	// - A bar/line represents one day
	// - each bar/line is 60 chars => each char is 24 minutes (60*24 mins per day)
	// - each char is 8 bits. Because bars are rendered from left to right, bits
	//   are reversed within their byte (high bit = earlier):
	//         0            0            0            1             1       ...
	//   [0:00, 0:03) [0:03, 0:06) [0:06, 0:09) [0:09, 0:12), [0:12, 0:15), ....
	var (
		op = newBarOp()

		// left and right boundary of current window (3 minutes/one bit, in loop)
		cl, cr = time.Time{}, morning

		// Current interval index, and left/right boundaries
		n      = 0
		il, ir = time.Unix(intervals[0].Start, 0), time.Unix(intervals[0].End, 0)

		// The current "character" (24-minute window)
		window byte
	)
	fmt.Printf("I: [%d,%d]\n", int(il.Sub(morning).Minutes()), int(ir.Sub(morning).Minutes()))
	for i := 0; i < (60 * 8); i++ {
		cl = cr
		cr = cl.Add(3 * time.Minute)

		// Determine amount of interval in [cl, cr]
		var duration time.Duration
		for {
			if n == len(intervals) {
				break // no more intervals to overlap
			}
			if cr.Before(il) {
				break
			}
			if Leq(cl, ir) {
				// il <= cr and cl <= ir, there is overlap
				duration += MinT(cr, ir).Sub(MaxT(cl, il))
			}
			if Leq(cr, ir) {
				break // intervals[n] overlaps next bit as well
			}

			n++
			if n < len(intervals) {
				il, ir = time.Unix(intervals[n].Start, 0), time.Unix(intervals[n].End, 0)
			}
			fmt.Printf("I: [%d,%d]\n", int(il.Sub(morning).Minutes()), int(ir.Sub(morning).Minutes()))
		}
		if duration > 90*time.Second {
			// fmt.Printf("window (%s) |= (1 << (7-(%d%%8))\n", bin(window), i)
			// fmt.Printf("%d%%8 = %d, 7-(%d%%8) = %d\n1 << (7-(%d%%8)) = %s\n", i, i%8, i, 7-(i%8), i, bin(byte(byte(1)<<byte(7-(i%8)))))
			window |= (1 << byte(7-(i%8)))
			// fmt.Printf("window is now %s\n", bin(window))
		}

		if i%8 == 7 {
			// Window is filled out -- append to bar
			// fmt.Printf("window: %s\n", bin(window))
			if window == 0 || window == 0xff {
				op.put(int(window >> 7)) // hack -- works for put(0) and put(1)
			} else {
				best := -1
				bestCount := byte(8)
				for j, b := range blockMask {
					diff := bits(b ^ window)
					if diff < bestCount {
						// fmt.Printf("bits(%s ^ %s = %s) = %d\n", bin(b), bin(window), bin(b^window), bits(b^window))
						best = j
						bestCount = diff
					}
					if diff == 0 {
						break
					}
				}
				op.put(best)
			}
			window = 0
		}
	}
	return op.finish()
}

func bin(x byte) string {
	var result [8]byte
	for i := 0; i < 8; i++ {
		if x%2 == 1 {
			result[7-i] = '1'
		} else {
			result[7-i] = '0'
		}
		x >>= 1
	}
	return string(result[:])
}
