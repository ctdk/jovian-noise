package main

import (
	"github.com/soniakeys/unit"
	"time"
)

type jupiterPosition struct {
	entryDate time.Time
	rising unit.Time
	transit unit.Time
	set unit.Time
	ra unit.RA
	dec unit.Angle
}

func (jp *jupiterPosition) Skip(secs unit.Time) bool {
	return !((jp.rising < jp.set && jp.rising < secs && secs < jp.set) || (jp.rising > jp.set && (secs > jp.rising || secs < jp.set)))
}
