package main

import (
	"fmt"
	"github.com/soniakeys/meeus/v3/globe"
	"github.com/soniakeys/unit"
	"math"
	"time"
)

type jupiterPosition struct {
	EntryDate time.Time
	Rising unit.Time
	Transit unit.Time
	Set unit.Time
	RA unit.RA
	Dec unit.Angle
}

type radioSource int

const dayUnitTime unit.Time = 24 * 60 * 60 // 86400

const recommendCutoff float64 = 3.0

const (
	NoEvent = iota
	IoA
	IoB
	IoC
	NonIoA
)

var radioSourceNames = []string{
	"Io-A",
	"Io-B",
	"Io-C",
	"non-Io-A",
}

type hzCoords struct {
	Altitude unit.Angle
	Azimuth unit.Angle
}

type jupiterData struct {
	StartTime time.Time
	EndTime time.Time
	Duration time.Duration
	Interval int
	Coords globe.Coord
	DisplayLongitude int
	LocalForecast bool
	JupiterPositions map[string]*jupiterPosition
	Intervals []*forecastInterval
}

type forecastInterval struct {
	Instant time.Time
	IoPhase float64
	Meridian float64
	Distance float64
	RadioSource radioSource
	TransitHA unit.HourAngle
	AltAz hzCoords
}

func (s radioSource) String() string {
	return radioSourceNames[s-1]
}

func (jp *jupiterPosition) Skip(secs unit.Time) bool {
	return !((jp.Rising < jp.Set && jp.Rising < secs && secs < jp.Set) || (jp.Rising > jp.Set && (secs > jp.Rising || secs < jp.Set)))
}

func (fi *forecastInterval) recommended() bool {
	return math.Abs(float64(fi.TransitHA)) < recommendCutoff
}

func (jd *jupiterData) GetCorrectTransit(entryDate time.Time) (unit.Time, error) {
	rounded := entryDate.Truncate(oneDay)
	jp, ok := jd.JupiterPositions[rounded.Format(jpFormat)]
	if !ok {
		return 0, fmt.Errorf("Aiieeeee, no Jupiter position found for key %s!", rounded.Format(jpFormat))
	}

	cur := unit.Time(entryDate.Sub(jp.EntryDate) / time.Second)

	// catch the pathologic case
	if jp.Skip(cur) {
		return 0, fmt.Errorf("Jupiter is not visible during the given time: %s %f.", rounded.Format(jpFormat), cur)
	}

	var trn unit.Time

	switch {
	case jp.Rising < jp.Set || (jp.Rising > jp.Set && (jp.Transit > jp.Rising && cur > jp.Rising || jp.Transit < jp.Set && cur < jp.Set)):
		trn = jp.Transit
	case jp.Transit > jp.Rising:
		// previous day
		pjp, ok := jd.JupiterPositions[rounded.Add(-oneDay).Format(jpFormat)]
		if !ok {
			return 0, fmt.Errorf("No Jupiter position available for the day before %s", rounded.Format(jpFormat))
		}
		offset := dayUnitTime - pjp.Transit
		trn = -(cur + offset)
	case jp.Transit < jp.Set:
		// next day
		njp, ok := jd.JupiterPositions[rounded.Add(oneDay).Format(jpFormat)]
		if !ok {
			return 0, fmt.Errorf("No Jupiter position available for the day after %s", rounded.Format(jpFormat))
		}
		offset := dayUnitTime - cur
		trn = njp.Transit + offset
	default:
		return 0, fmt.Errorf("Shouldn't have gotten here, but did. %s %f.", rounded.Format(jpFormat), cur)
	}

	return trn, nil
}
