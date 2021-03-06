// Originally inspired by:
// http://www.spaceacademy.net.au/spacelab/projects/jovrad/jovrad.htm,
// but extended and updated with some external libraries.
//
// Copyright 2016 Jeremy Bingham, under the MIT License.
// See the LICENSE file in this repository, or 
// http://www.opensource.org/licenses/MIT

/*
jovian-noise is a program that will forecast possible upcoming Jupiter decameter radio storms, which you can hear on a shortwave radio (best between 18MHz and 23MHz), with a suitable receiver and antenna.

This program should be reasonably accurate, but there's always a possibility that the event will fail to materialize for one reason or another - possibly a bug with the program, possibly a bug with Jupiter itself.

This was originally inspired by a QBASIC program at http://www.spaceacademy.net.au/spacelab/projects/jovrad/jovrad.htm, but uses external libraries for many of the calculations and can optionally limit the returned results to when Jupiter will be above the horizon at your location.

To run this program, you will need to obtain the VSOP87 files for planet locations (an archive is located at ftp://cdsarc.u-strasbg.fr/pub/cats/VI%2F81/) and place them in a directory somewhere. The environment variable VSOP87 must be set to the path of the directory with the VSOP87 files.

    Usage of ./jovian-noise:
      -duration duration
       	    Duration (in golang ParseDuration format) from the start time to calculate the forecast (default 720h0m0s)
      -interval int
    	    Interval in minutes to calculate the forecast (default 30)
      -lat int
    	    Optional latitute. If given, will limit results to when Jupiter is above the horizon at this location. Requires -lon
      -lon int
    	    Optional longitude. If given, will limit results to when Jupiter is above the horizon at this location. Requires -lat
      -start-time string
    	    Start time (in RFC 3339 format) to calculate Jupiter radio storm forecasts (defaults to now)

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

Copyright 2016, Jeremy Bingham, under the terms of the MIT License.

*/
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"time"
	"github.com/soniakeys/meeus/elliptic"
	"github.com/soniakeys/meeus/sidereal"
	"github.com/soniakeys/meeus/globe"
	"github.com/soniakeys/meeus/rise"
	"github.com/soniakeys/meeus/julian"
	pp "github.com/soniakeys/meeus/planetposition"
	"github.com/soniakeys/unit"
)

var months = []string{
	"",
	"Jan",
	"Feb",
	"Mar",
	"Apr",
	"May",
	"Jun",
	"Jul",
	"Aug",
	"Sep",
	"Oct",
	"Nov",
	"Dec",
}

const version string = "0.1.6"

var toRad = math.Pi / 180
var toDeg = unit.Angle(180 / math.Pi)

func main() {
	startTime := flag.String("start-time", "", "Start time (in RFC 3339 format) to calculate Jupiter radio storm forecasts (defaults to now)")
	dur := flag.Duration("duration", 30 * 24 * time.Hour, "Duration (in golang ParseDuration format) from the start time to calculate the forecast")
	interval := flag.Int("interval", 30, "Interval in minutes to calculate the forecast")
	lat := flag.Int("lat", 0, "Optional latitute. If given, will limit results to when Jupiter is above the horizon at this location. Requires -lon")
	lon := flag.Int("lon", 0, "Optional longitude. If given, will limit results to when Jupiter is above the horizon at this location. Requires -lat")
	ver := flag.Bool("version", false, "Print version number and exit.")
	nonIoA := flag.Bool("non-io-a", false, "Include forecasts for the non-Io-A radio source.")

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
	if *dur < time.Duration(*interval) * time.Minute {
		fmt.Printf("-duration really should be longer than the interval specified.\n")
		os.Exit(1)
	}

	if *startTime == "" {
		t = time.Now().UTC()
	} else {
		var err error
		t, err = time.Parse(time.RFC3339, *startTime)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		t = t.UTC()
	}
	if *lat != 0 && *lon == 0 || *lat == 0 && *lon != 0 {
		log.Println("Both -lat and -lon, or neither, must be supplied")
		os.Exit(1)
	}
	var risen bool
	var coords globe.Coord
	if *lat != 0 && *lon != 0 {
		// for some reason this figures longitude backwards from
		// the way everyone else does it.
		*lon = -*lon
		if *lon < 0 {
			*lon += 360
		}
		coords.Lon = unit.NewAngle('+', *lon, 0, 0)
		coords.Lat = unit.NewAngle('+', *lat, 0, 0)
		risen = true
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
	fmt.Printf("################################################################################\n")
	fmt.Printf("\t\tJovian Decameter Radio Storm Forecast for:\n")
	fmt.Printf("\t\t    %s\n", t)
	fmt.Printf("\t\t\t\tuntil:\n\t\t    %s\n", endTime)
	
	var localHead, fmtStr string
	if *lat != 0 && *lon != 0 {
		lz, loff := t.Local().Zone()
		fmt.Printf("\t\t     Local time zone: %s (%05d)\n", lz, (loff / 60 / 60) * 100)
		fmt.Printf("\t\t   --- For coordinates %dº, %dº ---\n", *lat, *lon)
		localHead = " HH:MM (local) |"
		fmtStr = "%3d  %s %2d  %02d:%02d         %02d:%02d %s   %6.2f     %6.2f   %4.2f       %s\n"
	} else {
		fmtStr = "%3d  %s %2d  %02d:%02d         %6.2f     %6.2f   %4.2f       %s\n"
	}
	fmt.Printf("################################################################################\n")
	fmt.Printf("DY | Date  | HH:MM (UTC) |%s Io Phase | CML    | Dist(AU) | source\n", localHead)
	fmt.Printf("--------------------------------------------------------------------------------\n")

	
	
	for t.Before(endTime) {
		jd := julian.TimeToJD(t)
		var skip bool
		if risen {
			// round the day off
			rounded := t.Round(24 * time.Hour)
			if rounded.After(t) {
				rounded = rounded.Add(-24 * time.Hour)
			}
			// subtle, but:
			rjd := julian.TimeToJD(rounded)
			secs := unit.Time(t.Sub(rounded) / time.Second)
			ra, dec := elliptic.Position(jupiter, earth, rjd)
			th0 := sidereal.Apparent0UT(rjd)
			h0 := rise.Stdh0Stellar
			rising, _, set, err := rise.ApproxTimes(coords, h0, th0, ra, dec)
			if err != nil {
				log.Println(err)
				skip = true
			}
			if !((rising < set && rising < secs && secs < set) || (rising > set && (secs > rising || secs < set))) {
				skip = true
			}
		}
		if !skip {
			el, _, eDist := earth.Position(jd)
			jl, _, jDist := jupiter.Position(jd)
			meridian := systemIIIMeridian(jd)
			eLon := float64(el * toDeg)
			jLon := float64(jl * toDeg)
			dist := distance(eLon, eDist, jLon, jDist)
			ioPhase := ioPos(jd, dist)
			rSource := source(meridian, ioPhase)
			
			if rSource != "" {
				if rSource != "non-Io-A" || *nonIoA {
					if *lat != 0 && *lon != 0 {
						l := t.Local()
						var localDay string
						if l.Day() != t.Day() {
							localDay = fmt.Sprintf("(%02d/%02d)", l.Month(), l.Day())
						} else {
							localDay = "       "
						}
						fmt.Printf(fmtStr, t.YearDay(), months[t.Month()], t.Day(), t.Hour(), t.Minute(), l.Hour(), l.Minute(), localDay, ioPhase, meridian, dist, rSource)
					} else {
						fmt.Printf(fmtStr, t.YearDay(), months[t.Month()], t.Day(), t.Hour(), t.Minute(), ioPhase, meridian, dist, rSource)
					}
				}
			}
		}
		t = t.Add(time.Duration(*interval) * time.Minute)
	}
}


// from http://www.projectpluto.com/grs_form.htm
func meridianCorrection(jd float64) float64 {
	jupMean := (jd - 2455636.938) * 360 / 4332.89709
	eqnCenter := 5.55 * math.Sin(jupMean)
	angle := (jd - 2451870.628) * 360 / 398.884 - eqnCenter
	correction := 11 * math.Sin(angle) + 5 * math.Cos(angle) - 1.25 * math.Cos(jupMean) - eqnCenter
	return correction
}

func systemIIIMeridian(jd float64) float64 {
	correction := meridianCorrection(jd)
	mFull := 138.41 + 870.4535567 * jd + correction
	m := math.Mod(mFull, 360)
	return m
}

func ioPos(jd float64, dist float64) float64 {
	// snagged the equations for this from
	// https://github.com/akkana/scripts/blob/master/jsjupiter/jupiter.js
	d := jd - 2415020
	v := reg(134.63 + 0.00111587 * d * toRad);
	eAnomaly := reg((358.476 + 0.9856003 * d) * toRad)
	jAnomaly := reg((225.328 + 0.0830853 * d + 0.33 * math.Sin(v)) * toRad)
	j := reg((221.647 + 0.9025179 * d - 0.33 * math.Sin(v)) * toRad)
	a := reg((1.916 * math.Sin(eAnomaly) + 0.020 * math.Sin(2 * eAnomaly)) * toRad)
	b := reg((5.552 * math.Sin(jAnomaly) + 0.167 * math.Sin(2 * jAnomaly)) * toRad)
	k := reg(j + a - b)
	rvE := 1.00014 - 0.01672 * math.Cos(eAnomaly) - 0.00014 * math.Cos(2 * eAnomaly)
	psi := math.Asin(rvE / dist * math.Sin(k))

	ioAngle := reg((84.5506 + 203.4058630 * (d - dist / 173)) * toRad + psi - b)

	return math.Mod(ioAngle * float64(toDeg) + 180, 360)
}

func reg(a float64) float64 {
	return math.Mod(a, 2 * math.Pi)
}

func source(meridian float64, ioDeg float64) string {
	var s string
	switch {
	case (meridian <= 270 && meridian >= 200) && (ioDeg < 260 && ioDeg > 205):
		s = "Io-A"
	case (meridian < 185 && meridian > 105) && (ioDeg < 110 && ioDeg > 80):
		s = "Io-B"
	case (meridian < 360 && meridian > 300 || meridian < 20 && meridian > 0) && (ioDeg < 260 && ioDeg > 225):
		s = "Io-C"
	case (meridian < 280 && meridian > 230):
		s = "non-Io-A"
	}
	return s
}

func distance(eLon, eDistance, jLon, jDistance float64) float64 {
	angle := angleCalc(eLon, jLon)
	d2 := math.Pow(eDistance, 2) + math.Pow(jDistance, 2) - 2 * eDistance * jDistance * math.Cos(angle * toRad)
	d := math.Sqrt(d2)
	return d
}

func angleCalc(e, j float64) float64 {
	var a float64
	var b float64
	var angle float64
	if e < j {
		a = e
		b = j
	} else {
		b = e
		a = j
	}
	c1 := b - a
	c2 := 360 + a - b
	if c2 < c1 {
		angle = c2
	} else {
		angle = c1
	}
	return angle
}
