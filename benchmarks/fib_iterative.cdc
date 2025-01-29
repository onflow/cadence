access(all)
fun fib(_ n: Int): Int {
    var fib1 = 1
    var fib2 = 1
    var fibonacci = fib1
    var i = 2
    while i < n {
        fibonacci = fib1 + fib2
        fib1 = fib2
        fib2 = fibonacci
        i = i + 1
    }
    return fibonacci
}

access(all)
fun main() {
    assert(fib(23) == 28657)
}
