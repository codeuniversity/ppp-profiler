var average = get("average", 0)
var count = get("count", 0)
average = ((average * count) + message.value) / (count+1)
count++
set("average", average)
set("current", message.value)
set("count", count)

display("average", average)
display("current", message.value)
