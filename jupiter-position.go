package main

import (
	"encoding/json"
	"fmt"
	"github.com/soniakeys/meeus/v3/globe"
	"github.com/soniakeys/unit"
	"math"
	"time"
)

type jupiterPosition struct {
	EntryDate time.Time  `json:"entry_date"`
	Rising    unit.Time  `json:"rising"`
	Transit   unit.Time  `json:"transit"`
	Set       unit.Time  `json:"set"`
	RA        unit.RA    `json:"ra"`
	Dec       unit.Angle `json:"dec"`
}

type radioSource int

const (
	NoEvent radioSource = iota
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

const dayUnitTime unit.Time = 24 * 60 * 60 // 86400
const recommendCutoff float64 = 3.0

type hzCoords struct {
	Altitude unit.Angle `json:"altitude"`
	Azimuth  unit.Angle `json:"azimuth"`
}

type jupiterData struct {
	StartTime        time.Time                   `json:"start_time"`
	EndTime          time.Time                   `json:"end_time"`
	Duration         time.Duration               `json:"duration"`
	Interval         int                         `json:"interval"`
	Coords           globe.Coord                 `json:"coords"`
	LocalForecast    bool                        `json:"local_forecast"`
	Location         *time.Location              `json:"location_data"`
	JupiterPositions map[string]*jupiterPosition `json:"jupiter_positions,omitempty"`
	Intervals        []*forecastInterval         `json:"intervals"`
}

type forecastInterval struct {
	Instant     time.Time      `json:"instant"`
	IoPhase     unit.Angle     `json:"io_phase"`
	Meridian    unit.Angle     `json:"meridian"`
	Distance    float64        `json:"distance"`
	RadioSource radioSource    `json:"radio_source"`
	TransitHA   unit.HourAngle `json:"transit_ha"`
	AltAz       *hzCoords      `json:"altaz,omitempty"`
}

func (s radioSource) String() string {
	return radioSourceNames[s-1]
}

func RadioSourceFromString(rs string) (radioSource, error) {
	var source radioSource
	for k, v := range radioSourceNames {
		if v == rs {
			source = radioSource(k) + 1
		}
	}
	if source == 0 {
		return NoEvent, fmt.Errorf("The name '%s' is not a valid radio source.", rs)
	}
	return source, nil
}

func (jp *jupiterPosition) Skip(secs unit.Time) bool {
	return !((jp.Rising < jp.Set && jp.Rising < secs && secs < jp.Set) || (jp.Rising > jp.Set && (secs > jp.Rising || secs < jp.Set)))
}

func (fi *forecastInterval) Recommended() bool {
	if fi.AltAz == nil {
		return false
	}
	return math.Abs(float64(fi.TransitHA.Hour())) < recommendCutoff
}

func (fi *forecastInterval) MarshalJSON() ([]byte, error) {
	type Alias forecastInterval
	return json.Marshal(&struct {
		RadioSource string `json:"radio_source"`
		*Alias
	}{
		RadioSource: fi.RadioSource.String(),
		Alias:       (*Alias)(fi),
	})
}

func (fi *forecastInterval) UnmarshalJSON(data []byte) error {
	type Jfi forecastInterval
	nj := &struct {
		RadioSource string `json:"radio_source"`
		*Jfi
	}{
		Jfi: (*Jfi)(fi),
	}
	if err := json.Unmarshal(data, &nj); err != nil {
		return err
	}
	rs, err := RadioSourceFromString(nj.RadioSource)
	if err != nil {
		return err
	}
	fi.RadioSource = rs
	return nil
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
			return 0, fmt.Errorf("No Jupiter position available for the day after %s.", rounded.Format(jpFormat))
		}
		offset := dayUnitTime - cur
		trn = njp.Transit + offset
	default:
		return 0, fmt.Errorf("Shouldn't have gotten here, but did. %s %f.", rounded.Format(jpFormat), cur)
	}

	return trn, nil
}
