access(all)
fun fib(_ n: Int): Int {
    if n < 2 {
        return n
    }
    return fib(n - 1) + fib(n - 2)
}

access(all)
fun main() {
    assert(fib(23) == 28657)
}
