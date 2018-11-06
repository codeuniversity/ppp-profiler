var average = get("average", 0)
var count = get("count", 0)
average = ((average * count) + message.value) / (count+1)
count++
set("average", average)
set("current", message.value)
set("count", count)

var t = "The lifetime average of the CPU temperature is "+ average.toFixed(2) +"C"
title(t)
description("Its current temperature is "+message.value.toFixed(2) + "C")

if (message.value > 60) {
  action("You should cool it down")
}
