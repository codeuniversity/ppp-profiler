var sum = get("sum", 0)
sum += message.value
set("sum", sum)

title("Sum")
description("The sum is " + sum.toFixed(2))

if (sum > 9000) {
  action("It's over 9000!")
}

