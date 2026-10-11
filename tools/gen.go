//go:build ignore

// gen writes the dominoes sounds into client/sounds: its own (a tile slapped
// down, a knock on the table, tiles drawn, a count of fives, a hand won) and the Concord Arcade
// kit. Run from the repo root: go run tools/gen.go
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
	sound("slap.wav", .22, func(t float64) float64 { // a tile slapped on the card table
		return .55*noise(t)*math.Exp(-t*90) + .5*math.Sin(2*math.Pi*(180-120*t)*t)*math.Exp(-t*30) + .3*math.Sin(2*math.Pi*1900*t)*math.Exp(-t*160)
	})
	sound("knock.wav", .34, func(t float64) float64 { // two knuckle knocks
		v := 0.0
		for i := 0; i < 2; i++ {
			if s := t - float64(i)*.16; s >= 0 {
				v += .6*math.Sin(2*math.Pi*140*s)*math.Exp(-s*45) + .3*noise(s)*math.Exp(-s*120)
			}
		}
		return v
	})
	sound("draw.wav", .3, func(t float64) float64 { // tiles slid out of the boneyard
		v := 0.0
		for i, f := range []float64{2600, 2200, 2400} {
			if s := t - float64(i)*.07; s >= 0 {
				v += .25*math.Sin(2*math.Pi*f*s)*math.Exp(-s*150) + .15*noise(s)*math.Exp(-s*90)
			}
		}
		return v
	})
	sound("hand.wav", .55, func(t float64) float64 { // a hand won: three notes up
		v := 0.0
		for i, f := range []float64{523.25, 659.25, 783.99} {
			if s := t - float64(i)*.1; s >= 0 {
				v += .22 * math.Tanh(3*math.Sin(2*math.Pi*f*s)) * math.Exp(-s*7)
			}
		}
		return v
	})
	sound("five.wav", .4, func(t float64) float64 { // a count of fives: a quick bright arpeggio
		v := 0.0
		for i, f := range []float64{783.99, 1174.66, 1567.98} {
			if s := t - float64(i)*.06; s >= 0 {
				v += .2 * math.Tanh(2.5*math.Sin(2*math.Pi*f*s)) * math.Exp(-s*12)
			}
		}
		return v
	})
	fmt.Println("dominoes sounds and the arcade kit written to client/sounds")
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
