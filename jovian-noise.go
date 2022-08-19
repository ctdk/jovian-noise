// Originally inspired by:
// http://www.spaceacademy.net.au/spacelab/projects/jovrad/jovrad.htm,
// but extended and updated with some external libraries.
//
// Copyright 2016-2022 Jeremy Bingham, under the MIT License.
// See the LICENSE file in this repository, or
// http://www.opensource.org/licenses/MIT

/*
jovian-noise is a program that will forecast possible upcoming Jupiter decameter radio storms, which you can hear on a shortwave radio (best between 18MHz and 23MHz), with a suitable receiver and antenna.

This program should be reasonably accurate, but there's always a possibility that the event will fail to materialize for one reason or another - possibly a bug with the program, possibly a bug with Jupiter itself.

This was originally inspired by a QBASIC program at http://www.spaceacademy.net.au/spacelab/projects/jovrad/jovrad.htm, but uses external libraries for many of the calculations and can optionally limit the returned results to when Jupiter will be above the horizon at your location.

To run this program, you will need to obtain the VSOP87 files for planet locations (an archive is located at ftp://cdsarc.u-strasbg.fr/pub/cats/VI%2F81/, but a github mirror located at https://github.com/ctdk/vsop87 is probably easiest) and place them in a directory somewhere. The environment variable VSOP87 must be set to the path of the directory with the VSOP87 files.

    Usage of ./jovian-noise:
      -duration duration
            Duration (in golang ParseDuration format) from the start time to calculate the forecast (default 720h0m0s)
      -interval int
            Interval in minutes to calculate the forecast (default 30)
      -lat int
            Optional latitute. If given, will limit results to when Jupiter is above the horizon at this location. Requires -lon
      -local
            Optionally use this computer's timzone to display results. Conflicts with -timezone and -offset-hours.
      -lon int
            Optional longitude. If given, will limit results to when Jupiter is above the horizon at this location. Requires -lat
      -non-io-a
            Include forecasts for the non-Io-A radio source.
      -offset-hours float
            Optional offset in hours east of UTC to display results. Offsets to the west should be given with negative numbers (e.g. '-offset-hours -7'). Conflicts with -timezone and -local.
      -output string
            How to format the forecast for output. Currently acceptable options are: text (default), json. (default "text")
      -start-time string
            Start time (in RFC 3339 format) to calculate Jupiter radio storm forecasts (defaults to the start of the current hour)
      -timezone string
            Optional timezone for displaying results. Conflicts with -offset-hours and -local.
      -version
            Print version number and exit.

Credits

Many web pages went into getting this together. The most immediately useful for this program were:

* http://www.spaceacademy.net.au/spacelab/projects/jovrad/jovrad.htm

* http://www.projectpluto.com/grs_form.htm

* https://github.com/akkana/scripts/blob/master/jsjupiter/jupiter.js

More Information about Jupiter amateur radio astronomy

* http://www.radiosky.com/rjcentral.html

* http://www.thrushobservatory.org/radio.htm

* http://radiojove.gsfc.nasa.gov/ -- NASA's Radio JOVE Project

License

Copyright 2016-2022, Jeremy Bingham, under the terms of the MIT License.

*/
package main

import (
	"flag"
	"fmt"
	"github.com/soniakeys/meeus/v3/coord"
	"github.com/soniakeys/meeus/v3/elliptic"
	"github.com/soniakeys/meeus/v3/julian"
	pp "github.com/soniakeys/meeus/v3/planetposition"
	"github.com/soniakeys/meeus/v3/rise"
	"github.com/soniakeys/meeus/v3/sidereal"
	"github.com/soniakeys/unit"
	"log"
	"math"
	"os"
	"time"
)

const version string = "0.3.1"
const jpFormat string = "2006-01-02"
const oneDay time.Duration = 24 * time.Hour

func main() {
	startTime := flag.String("start-time", "", "Start time (in RFC 3339 format) to calculate Jupiter radio storm forecasts (defaults to the start of the current hour)")
	dur := flag.Duration("duration", 30*oneDay, "Duration (in golang ParseDuration format) from the start time to calculate the forecast")
	interval := flag.Int("interval", 30, "Interval in minutes to calculate the forecast")
	tz := flag.String("timezone", "", "Optional timezone for displaying results. Conflicts with -offset-hours and -local.")
	offsetHours := flag.Float64("offset-hours", 0, "Optional offset in hours east of UTC to display results. Offsets to the west should be given with negative numbers (e.g. '-offset-hours -7' or '-offset-hours 9.5'). Conflicts with -timezone and -local.")
	localTZ := flag.Bool("local", false, "Optionally use this computer's timzone to display results. Conflicts with -timezone and -offset-hours.")
	lat := flag.Int("lat", 0, "Optional latitute. If given, will limit results to when Jupiter is above the horizon at this location. Requires -lon")
	lon := flag.Int("lon", 0, "Optional longitude. If given, will limit results to when Jupiter is above the horizon at this location. Requires -lat")
	ver := flag.Bool("version", false, "Print version number and exit.")
	nonIoA := flag.Bool("non-io-a", false, "Include forecasts for the non-Io-A radio source.")
	output := flag.String("output", "text", "How to format the forecast for output. Currently acceptable options are: text (default), json.")

	jData := new(jupiterData)
	jData.Intervals = make([]*forecastInterval, 0)

	var t time.Time
	flag.Parse()

	if *ver {
		fmt.Printf("jovian-noise version %s\n", version)
		os.Exit(0)
	}

	if *interval < 1 {
		fmt.Printf("-interval must be at least 1 minute.\n")
		os.Exit(1)
	}
	if *dur < time.Duration(*interval)*time.Minute {
		fmt.Printf("-duration really should be longer than the interval specified.\n")
		os.Exit(1)
	}

	roundOffset := int(*offsetHours)
	// bleh
	if *tz != "" && roundOffset != 0 || *tz != "" && *localTZ || roundOffset != 0 && *localTZ {
		fmt.Printf("One of the -timezone, -offset-hours, and -local flags can be specified, but not more than that. None are required, however.")
		os.Exit(1)
	}

	if *tz != "" {
		if loc, err := time.LoadLocation(*tz); err != nil {
			fmt.Printf("Error loading timezone %s: %w\n", *tz, err)
		} else {
			jData.Location = loc
		}
	} else if roundOffset != 0 {
		m := time.Duration(*offsetHours * 60)
		secondsEast := int((m * time.Minute).Seconds())
		jData.Location = time.FixedZone("Manual Offset Zone", secondsEast)
	} else if *localTZ {
		jData.Location = time.Local
	}

	jData.Duration = *dur
	jData.Interval = *interval

	if *startTime == "" {
		t = time.Now().UTC().Truncate(time.Hour)
	} else {
		var err error
		t, err = time.Parse(time.RFC3339, *startTime)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		t = t.UTC()
	}
	jData.StartTime = t

	var latSet, lonSet bool
	for _, v := range os.Args {
		if v == "-lat" {
			latSet = true
		} else if v == "-lon" {
			lonSet = true
		}
	}

	if latSet && !lonSet || !latSet && lonSet {
		log.Println("Both -lat and -lon, or neither, must be supplied")
		os.Exit(1)
	}

	if latSet && lonSet {
		// for some reason this figures longitude backwards from
		// the way everyone else does it.
		if *lon != 0 {
			*lon = -*lon
			if *lon < 0 {
				*lon += 360
			}
		}

		jData.Coords.Lon = unit.NewAngle('+', *lon, 0, 0)
		jData.Coords.Lat = unit.NewAngle('+', *lat, 0, 0)
		jData.LocalForecast = true
	}

	earth, err := pp.LoadPlanet(pp.Earth)
	if err != nil {
		panic(err)
	}
	jupiter, err := pp.LoadPlanet(pp.Jupiter)
	if err != nil {
		panic(err)
	}
	endTime := t.Add(*dur - time.Second)
	jData.EndTime = endTime

	var jupPositions map[string]*jupiterPosition

	if jData.LocalForecast {
		// Calculate Jupiter's positions ahead of time.
		jupPositions = make(map[string]*jupiterPosition, endTime.Sub(t)/time.Hour/24/2)

		tJup := t.Add(-oneDay)
		for tJup.Before(endTime.Add(2 * oneDay)) {
			rounded := tJup.Truncate(oneDay)
			// subtle, but:
			rjd := julian.TimeToJD(rounded)
			ra, dec := elliptic.Position(jupiter, earth, rjd)
			th0 := sidereal.Apparent0UT(rjd)
			h0 := rise.Stdh0Stellar
			rising, transit, set, err := rise.ApproxTimes(jData.Coords, h0, th0, ra, dec)
			if err != nil {
				log.Fatal(err)
			}
			jp := &jupiterPosition{EntryDate: rounded, Rising: rising, Transit: transit, Set: set, RA: ra, Dec: dec}
			jupPositions[rounded.Format(jpFormat)] = jp

			tJup = tJup.Add(oneDay)
		}
		jData.JupiterPositions = jupPositions
	}

	for t.Before(endTime) {
		jd := julian.TimeToJD(t)
		var skip bool
		var jp *jupiterPosition
		if jData.LocalForecast {
			// round the day off
			var ok bool
			rounded := t.Truncate(oneDay)
			secs := unit.Time(t.Sub(rounded) / time.Second)
			if jp, ok = jupPositions[rounded.Format(jpFormat)]; ok {
				skip = jp.Skip(secs)
			} else {
				log.Fatalf("Strange, no precalculated Jupiter position for %v under key %s was found.", rounded, rounded.Format(jpFormat))
			}
		}
		if !skip {
			el, _, eDist := earth.Position2000(jd)
			jl, _, jDist := jupiter.Position2000(jd)
			meridian := systemIIIMeridian(jd)
			dist := distance(el, eDist, jl, jDist)
			ioPhase := ioPos(jd, dist)
			rSource := source(meridian, ioPhase)

			if rSource != NoEvent && (rSource != NonIoA || *nonIoA) {
				fi := new(forecastInterval)
				fi.Instant = t
				fi.IoPhase = ioPhase
				fi.Meridian = meridian
				fi.Distance = dist
				fi.RadioSource = rSource

				if jData.LocalForecast {
					cur := float64(t.Sub(jp.EntryDate) / time.Second)
					correctTransit, err := jData.GetCorrectTransit(t)
					if err != nil {
						log.Fatal(err)
					}
					diff := cur - float64(correctTransit)
					fi.TransitHA = unit.HourAngleFromSec(diff)
					az, alt := coord.EqToHz(jp.RA, jp.Dec, jData.Coords.Lat, jData.Coords.Lon, sidereal.Apparent(jd))
					fi.AltAz = &hzCoords{Altitude: alt, Azimuth: az + math.Pi}
				}
				jData.Intervals = append(jData.Intervals, fi)
			}
		}
		t = t.Add(time.Duration(*interval) * time.Minute)
	}

	switch *output {
	case "text":
		if err := outputText(jData); err != nil {
			log.Fatal(err)
		}
	case "json":
		if err := outputJSON(jData); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("Output format '%s' is not a valid selection. Aborting.", *output)
	}
	os.Exit(0)
}
