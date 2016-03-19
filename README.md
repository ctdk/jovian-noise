jovian-noise is a program that will forecast possible upcoming Jupiter decameter radio storms, which you can hear on a shortwave radio (best between 18MHz and 23MHz), with a suitable receiver and antenna.

This program should be reasonably accurate, but there's always a possibility that the event will fail to materialize for one reason or another - possibly a bug with the program, possibly a bug with Jupiter itself.

This was originally inspired by a QBASIC program at http://www.spaceacademy.net.au/spacelab/projects/jovrad/jovrad.htm, but uses external libraries for many of the calculations and can optionally limit the returned results to when Jupiter will be above the horizon at your location.

To run this program, you will need to obtain the VSOP87 files for planet locations (an archive is located at ftp://cdsarc.u-strasbg.fr/pub/cats/VI%2F81/) and place them in a directory somewhere. The environment variable VSOP87 must be set to the path of the directory with the VSOP87 files.

```
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
```

###Credits

Many web pages went into getting this together. The most immediately useful for this program were:

* http://www.spaceacademy.net.au/spacelab/projects/jovrad/jovrad.htm
* http://www.projectpluto.com/grs_form.htm
* https://github.com/akkana/scripts/blob/master/jsjupiter/jupiter.js

### License

Copyright 2016, Jeremy Bingham, under the terms of the MIT License.
