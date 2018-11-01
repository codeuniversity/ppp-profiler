var sum = get("sum", 0)
sum += message.value
set("sum", sum)

display("title", "Sum")
display("description", "The sum is " + sum.toFixed(2))
