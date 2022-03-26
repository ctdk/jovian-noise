package main

import (
	"math"
)

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

func source(meridian float64, ioDeg float64) radioSource {
	var s radioSource
	switch {
	case (meridian <= 270 && meridian >= 200) && (ioDeg < 260 && ioDeg > 205):
		s = IoA
	case (meridian < 185 && meridian > 105) && (ioDeg < 110 && ioDeg > 80):
		s = IoB
	case (meridian < 360 && meridian > 300 || meridian < 20 && meridian > 0) && (ioDeg < 260 && ioDeg > 225):
		s = IoC
	case (meridian < 280 && meridian > 230):
		s = NonIoA
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
