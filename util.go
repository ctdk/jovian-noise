package main

import (
	"github.com/soniakeys/unit"
	"math"
)

const fullCircle = 2 * math.Pi
const toRad = math.Pi / 180
const toDeg = unit.Angle(180 / math.Pi)

// TODO: These really ought to be changed to use radians to reduce the amount
// of conversions back and forth between radians and degrees.

// from http://www.projectpluto.com/grs_form.htm
func meridianCorrection(jd float64) float64 {
	jupMean := (jd - 2455636.938) * 360 / 4332.89709
	eqnCenter := 5.55 * math.Sin(jupMean*toRad)
	angle := ((jd-2451870.628)*360/398.884 - eqnCenter) * toRad
	correction := 11*math.Sin(angle) + 5*math.Cos(angle) - 1.25*math.Cos(jupMean) - eqnCenter
	return correction
}

func systemIIIMeridian(jd float64) unit.Angle {
	correction := meridianCorrection(jd)
	mFull := 138.41 + 870.4535567*jd + correction
	m := unit.Angle(math.Mod(mFull*toRad, fullCircle))
	return m
}

func ioPos(jd float64, dist float64) unit.Angle {
	// snagged the equations for this from
	// https://github.com/akkana/scripts/blob/master/jsjupiter/jupiter.js
	d := jd - 2415020
	v := reg(134.63 + 0.00111587*d*toRad)
	eAnomaly := reg((358.476 + 0.9856003*d) * toRad)
	jAnomaly := reg((225.328 + 0.0830853*d + 0.33*math.Sin(v)) * toRad)
	j := reg((221.647 + 0.9025179*d - 0.33*math.Sin(v)) * toRad)
	a := reg((1.916*math.Sin(eAnomaly) + 0.020*math.Sin(2*eAnomaly)) * toRad)
	b := reg((5.552*math.Sin(jAnomaly) + 0.167*math.Sin(2*jAnomaly)) * toRad)
	k := reg(j + a - b)
	rvE := 1.00014 - 0.01672*math.Cos(eAnomaly) - 0.00014*math.Cos(2*eAnomaly)
	psi := math.Asin(rvE / dist * math.Sin(k))

	ioAngle := reg((84.5506+203.4058630*(d-dist/173))*toRad + psi - b)

	return unit.Angle(math.Mod(ioAngle+math.Pi, fullCircle))
}

func reg(a float64) float64 {
	return math.Mod(a, fullCircle)
}

func source(m unit.Angle, io unit.Angle) radioSource {
	var s radioSource
	meridian := m.Deg()
	ioDeg := io.Deg()

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

func distance(eLon unit.Angle, eDistance float64, jLon unit.Angle, jDistance float64) float64 {
	angle := angleCalc(eLon, jLon)
	d2 := math.Pow(eDistance, 2) + math.Pow(jDistance, 2) - 2*eDistance*jDistance*math.Cos(angle.Rad())
	d := math.Sqrt(d2)
	return d
}

func angleCalc(e, j unit.Angle) unit.Angle {
	var a unit.Angle
	var b unit.Angle
	var angle unit.Angle
	if e < j {
		a, b = e, j
	} else {
		b, a = e, j
	}
	c1 := b - a
	c2 := fullCircle + a - b
	if c2 < c1 {
		angle = c2
	} else {
		angle = c1
	}
	return angle
}
