# maprange

A Go analyzer which detects uses of for-range statements over maps.
For such statements, iteration order is undefined / nondeterministic. 
