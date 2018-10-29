var average = get("average", 0)
var count = get("count", 0)
average = ((average * count) + message.value) / (count+1)
count++
set("average", average)
set("current", message.value)
set("count", count)

var title = "The lifetime average of the CPU temperature is "+ average.toFixed(2) +"C"
display("title", title)
display("description", "Its current temperature is "+message.value.toFixed(2) + "C")
