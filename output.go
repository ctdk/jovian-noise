package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	sexa "github.com/soniakeys/sexagesimal"
	"os"
	"strings"
	"text/tabwriter"
	"text/template"
	"time"
)

type textOutput struct {
	Start    time.Time
	End      time.Time
	Lat      int
	Lon      int
	Local    bool
	Location string
	Offset   string
	Data     string
}

func outputJSON(jData *jupiterData) error {
	if j, err := json.MarshalIndent(jData, "", "\t"); err != nil {
		return err
	} else {
		os.Stdout.Write(j)
	}
	return nil
}

func outputText(jData *jupiterData) error {
	// Set the template up first in case anything somehow goes horribly
	// wrong.
	tmpl, err := template.New("textOut").Parse(strings.TrimSpace(textOutputTemplate))
	if err != nil {
		return err
	}

	outData := new(textOutput)
	outData.Start = jData.StartTime
	outData.End = jData.EndTime
	outData.Lat = int(jData.Coords.Lat.Deg())
	outData.Lon = -int(jData.Coords.Lon.Deg())
	outData.Local = jData.LocalForecast
	if jData.Location != nil {
		ztz, zoff := jData.StartTime.In(jData.Location).Zone()
		if jData.Location != time.Local {
			outData.Location = jData.Location.String()
		} else {
			outData.Location = ztz
		}

		zhours := zoff / 60 / 60
		zmin := zoff / 60 % 60
		if zmin < 0 {
			zmin = -zmin
		}
		outData.Offset = fmt.Sprintf("%+03d%02d", zhours, zmin)
	}
	if outData.Lon < -180 {
		outData.Lon += 360
	}

	// the actual data
	var b bytes.Buffer
	bio := bufio.NewWriter(&b)
	w := tabwriter.NewWriter(bio, 1, 8, 1, ' ', 0)

	var localHeading string
	var localDash string
	if jData.Location != nil {
		localHeading = "Local\t"
		localDash = "-----\t"
	}

	if jData.LocalForecast {
		fmt.Fprintf(w, "DY\tDate\tUTC\t%sPhase°\tCML\tDist.\tTrHA\tSrc\tAlt.\tAz.\tRec\t\n", localHeading)
		fmt.Fprintf(w, "--\t----\t---\t%s------\t---\t-----\t----\t---\t----\t---\t---\t\n", localDash)
		for _, fi := range jData.Intervals {
			var rec string
			if fi.Recommended() {
				rec = "Y"
			} else {
				rec = "N"
			}
			var localData string
			if jData.Location != nil {
				var nextDay string
				l := fi.Instant.In(jData.Location)
				if l.YearDay() != fi.Instant.YearDay() {
					nextDay = "*"
				}
				localData = fmt.Sprintf("%s%s\t", l.Format("15:04"), nextDay)
			}

			fmt.Fprintf(w, "%d\t%s \t%s\t%s%0.2f\t%0.2f\t%0.2f\t%+0.2f\t%s\t%0.2j\t%0.2j\t%s\t\n", fi.Instant.YearDay(), fi.Instant.Format("Jan 02"), fi.Instant.Format("15:04"), localData, fi.IoPhase.Deg(), fi.Meridian.Deg(), fi.Distance, fi.TransitHA.Hour(), fi.RadioSource, sexa.FmtAngle(fi.AltAz.Altitude), sexa.FmtAngle(fi.AltAz.Azimuth), rec)
		}
	} else {
		fmt.Fprintf(w, "DY\tDate\tUTC\t%sPhase°\tCML\tDist.\tSrc\t\n", localHeading)
		fmt.Fprintf(w, "--\t----\t---\t%s------\t---\t-----\t---\t\n", localDash)
		for _, fi := range jData.Intervals {
			var localData string
			if jData.Location != nil {
				var nextDay string
				l := fi.Instant.In(jData.Location)
				if l.YearDay() != fi.Instant.YearDay() {
					nextDay = "*"
				}
				localData = fmt.Sprintf("%s%s\t", l.Format("15:04"), nextDay)
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%s%0.2f\t%0.2f\t%0.2f\t%s\t\n", fi.Instant.YearDay(), fi.Instant.Format("Jan 02"), fi.Instant.Format("15:04"), localData, fi.IoPhase.Deg(), fi.Meridian.Deg(), fi.Distance, fi.RadioSource)
		}
	}

	w.Flush()
	bio.Flush()
	outData.Data = strings.TrimSpace(b.String())

	if err = tmpl.Execute(os.Stdout, outData); err != nil {
		return err
	}

	return nil
}
