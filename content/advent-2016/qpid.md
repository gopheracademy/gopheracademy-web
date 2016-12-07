+++
author = ["Brian Ketelsen"]
date = "2016-12-07T05:26:58-05:00"
series = ["Advent 2016"]
title = "QPID - Go Powered BBQ"

+++

Two of my favorite things to do are write Go code and make BBQ.  This fall, I started a project that combined these passions into an interesting project.  When I got a [new BBQ grill](https://myronmixonsmokers.com/h20-water-cookers/) this year, I wanted to find a way to control the temperature of the fire box programmatically.  Some Internet research led me to [Justin Dean's PitmasterPi project](https://github.com/justindean/PitmasterPi).  It's written in Python, but Justin was kind enough to include a great writeup on both the software and hardware he used to control his grill.  I knew this would be a good foundation for my project, so I ordered similar parts and set out to replicate the project in Go.

The brains of the system is a Raspberry Pi.  I think mine's an older 2-series model, rather than the newer ones, but I had several sitting in a drawer and figured this would be a good way to put one to use.  I started with Justin's [hardware list](https://github.com/justindean/PitmasterPi#pitmasterpi-hardware) but knew that my much larger firebox would require a larger fan to feed oxygen to the fire.  I settled on a 57 cubic feet per minute air conditioner blower I found on Amazon.com. 

After a few anxious days of waiting all of the hardware arrived and I set out building assembling the system.  I know that Florida can be pretty rough on anything that gets left outside for more than 10 minutes, so I used a 1/4" mono cable (guitar cable style) to connect the blower to the power relay that feeds it.

Here's a [photo](https://drive.google.com/file/d/1HQN1epsoRWeWiVbuYW6BwYCCLMSCYZY1qQ/view?usp=sharing) of the assembled Pi with the thermocoupler board attached to it.  After assembling the parts, I set out to replicate Justin's python code in Go.  It didn't take me long to remember that the [Gobot](http://gobot.io) project is the best starting place for any hardware project written in Go, so I read up on the documentation and used Gobot as my starting point for what eventually became [qpid](https://github.com/bbqgophers/qpid).

QPID uses Gobot to control the the temperature of the grill by controlling the oxygen that feeds the fire.  The routine that monitors the temperature and decides how much air to blow into the fire box is a [pid controller](https://en.wikipedia.org/wiki/PID_controller).  Fortunately someone else [already wrote one for Go](https://github.com/felixge/pidctrl), so I used that Go PID controller as the heart of the control loop for the grill system.

Here's a [shaky Blair Witch Project video](https://drive.google.com/file/d/1k-Xoz5JupY0_w_h1_rbpXAqAHwJ8oFywAQ/view?usp=sharing) of the first bench test of QPID running on my kitchen table.

Next I set out to modify my grill to seal the firebox so that the blower was the only source of oxygen for the fire chamber.  I used a [thick piece of stainless steel](https://drive.google.com/file/d/1oJV_x9w0V8HWFJ66l8SPC-IU77L5CYG6Cw/view?usp=sharing) to replace the adjustable vent on the left side of the grill, and cut out a hole to mount the air conditioner blower. Here's a video of the first [field test](https://drive.google.com/file/d/1d9pcQD1ISA0ySs2vaV4fg7Kmd2wNYdLj0Q/view?usp=sharing) of the hardware in action.

Next I added [prometheus](https://prometheus.io) integration to the QPID code so I could monitor it from inside and have alerting on the temperatures.  Prometheus is so simple to use it only took a few lines of code to add excellent monitoring to the application.  Both QPID and prometheus run on the Raspberry Pi, and even during a cook, the CPU usage never rises above 3%.  That's the efficiency of Go!

My [first cook](https://drive.google.com/file/d/13Sb7fMmrx6GmWChBfgTCnaJiRPKu4UVJqg/view?usp=sharing) turned out [really well](https://drive.google.com/file/d/1k8OmvgC8AZEmIIN9Fy52J2XEeD9uCUqQiA/view?usp=sharing), but I was plagued by wildly varying temperature readings from the temperature probe inside the grill.  I contacted Justin on [twitter](https://twitter.com/justinmdean) and he suggested that the probe might be shorting on a metal part of the grill.  I stripped the housing from an ethernet cable and wrapped the probe wire with it as a temporary fix.  That solved the problem!

With the help of the #bbq channel in the Gopher Slack, I've made several additions and improvements to the code since then.  It's still quite alpha quality: missing the ability to change the target temperature without restarting the application.  But generally it's pretty amazing being able to fill the firebox with wood in the early morning, start QPID and walk away for hours without having to tend the fire.  Every few hours I'll go back outside and add some more wood to the fire, but most of the time I just keep an eye on the Grafana graphs from inside the house.

QPID powered the Ketelsen [Thanksgiving Turkey](https://drive.google.com/file/d/12BOndwHK5J86RsvVNedjjC1_kat1-k8lNw/view?usp=sharing) this year, and it was the best turkey I've ever grilled.  Thanks to Justin's inspiration, the awesome Gobot team, and a little bit of soldering, we have a fun project powered by Go that helped to join two of my favorite things: Go and BBQ.

If you love BBQ and Go, we'd love your contributions on the [QPID Project](https://github.com/bbqgophers/qpid).  You can find a [parts list and general notes](https://github.com/bbqgophers/links) in our Github organization if you're looking to get started automating your cooking.
