access(all)
fun fib(_ n: Int): Int {
    if n == 0 {
        return 0
    }

    let f = [0, 1]

    var i = 2
    while i <= n {
        f.append(f[i - 1] + f[i - 2])
        i = i + 1
    }

    return f[n]
}

access(all)
fun main() {
    assert(fib(23) == 28657)
}
