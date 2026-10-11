//go:build ignore

// gen writes the chess sounds into client/sounds: its own (a piece set down
// with the clock slapped, a capture, a check's bell, castling, a draw) and
// the Concord Arcade kit. Run from the repo root: go run tools/gen.go
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/JMThomas00/Concord/sdk/arcade"
)

const rate = 22050

func main() {
	check(arcade.WriteSoundKit("client"))
	sound("move.wav", .28, func(t float64) float64 { // a wooden click, then the clock's button slapped
		v := .45*math.Sin(2*math.Pi*1400*t)*math.Exp(-t*140) + .3*math.Sin(2*math.Pi*380*t)*math.Exp(-t*90)
		if s := t - .14; s >= 0 {
			v += .5*noise(s)*math.Exp(-s*90) + .25*math.Sin(2*math.Pi*220*s)*math.Exp(-s*60)
		}
		return v
	})
	sound("capture.wav", .3, func(t float64) float64 { // two pieces knocked together, then the slap
		v := .55*math.Sin(2*math.Pi*2100*t)*math.Exp(-t*110) + .4*math.Sin(2*math.Pi*600*t)*math.Exp(-t*50)
		if s := t - .16; s >= 0 {
			v += .45*noise(s)*math.Exp(-s*90) + .25*math.Sin(2*math.Pi*220*s)*math.Exp(-s*60)
		}
		return v
	})
	sound("check.wav", .7, func(t float64) float64 { // the bell on the clock
		return .3*math.Sin(2*math.Pi*1318.5*t)*math.Exp(-t*5) + .15*math.Sin(2*math.Pi*2637*t)*math.Exp(-t*8) +
			.35*math.Sin(2*math.Pi*1400*t)*math.Exp(-t*140)
	})
	sound("castle.wav", .3, func(t float64) float64 { // king, then rook
		v := 0.0
		for i, f := range []float64{1400, 1250} {
			if s := t - float64(i)*.1; s >= 0 {
				v += .45*math.Sin(2*math.Pi*f*s)*math.Exp(-s*130) + .25*math.Sin(2*math.Pi*360*s)*math.Exp(-s*80)
			}
		}
		return v
	})
	sound("draw.wav", .6, func(t float64) float64 { // two gentle notes, the second lower
		v := 0.0
		for i, f := range []float64{587.33, 493.88} {
			if s := t - float64(i)*.16; s >= 0 {
				v += .25 * math.Sin(2*math.Pi*f*s) * math.Exp(-s*6)
			}
		}
		return v
	})
	fmt.Println("chess sounds and the arcade kit written to client/sounds")
}

// noise is a repeatable hiss (the same every run).
func noise(t float64) float64 {
	x := math.Sin(t*12345.678) * 43758.5453
	return 2*(x-math.Floor(x)) - 1
}

func sound(name string, dur float64, f func(t float64) float64) {
	n := int(dur * rate)
	data := make([]byte, 2*n)
	for i := 0; i < n; i++ {
		t := float64(i) / rate
		fade := math.Min(1, (dur-t)/.01)
		v := max(-1, min(1, f(t)*fade))
		binary.LittleEndian.PutUint16(data[2*i:], uint16(int16(v*32000)))
	}
	var h bytes.Buffer
	h.WriteString("RIFF")
	binary.Write(&h, binary.LittleEndian, uint32(36+len(data)))
	h.WriteString("WAVEfmt ")
	for _, v := range []any{uint32(16), uint16(1), uint16(1), uint32(rate), uint32(rate * 2), uint16(2), uint16(16)} {
		binary.Write(&h, binary.LittleEndian, v)
	}
	h.WriteString("data")
	binary.Write(&h, binary.LittleEndian, uint32(len(data)))
	h.Write(data)
	path := filepath.Join("client", "sounds", name)
	check(os.MkdirAll(filepath.Dir(path), 0o755))
	check(os.WriteFile(path, h.Bytes(), 0o644))
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
