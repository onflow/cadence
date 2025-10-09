access(all)
fun newArray(repeating value: Int, count: Int): [Int] {
    let array: [Int] = []
    for _ in InclusiveRange(0, count-1) {
        array.append(value)
    }
    return array
}

access(all)
fun fannkuch(_ n: Int): Int {
    let perm = newArray(repeating: 0, count: n)
    let count = newArray(repeating: 0, count: n)
    let perm1 = newArray(repeating: 0, count: n)

    for j in InclusiveRange(0, n-1) {
        perm1[j] = j
    }

    var f = 0
    var i = 0
    var k = 0
    var r = 0
    var flips = 0
    var nperm = 0
    var checksum = 0

    r = n
    while r > 0 {
        i = 0
        while r != 1 {
            count[r-1] = r
            r = r - 1
        }
        while i < n {
            perm[i] = perm1[i]
            i = i + 1
        }

        // Count flips and update max  and checksum
        f = 0
        k = perm[0]
        while k != 0 {
            i = 0
            while 2*i < k {
                let t = perm[i]
                perm[i] = perm[k-i]
                perm[k-i] = t
                i = i + 1
            }
            k = perm[0]
            f = f + 1
        }
        if f > flips {
            flips = f
        }

        if (nperm & 0x1) == 0 {
            checksum = checksum + f
        } else {
            checksum = checksum - f
        }

        // Use incremental change to generate another permutation
        var more = true
        while more {
            if r == n {
                log(checksum)
                return flips
            }
            let p0 = perm1[0]
            i = 0
            while i < r {
                let j = i+1
                perm1[i] = perm1[j]
                i = j
            }
            perm1[r] = p0

            count[r] = count[r] - 1
            if count[r] > 0 {
                more = false
            } else {
                r = r + 1
            }
        }
        nperm = nperm + 1
    }
    return flips
}

access(all)
fun main() {
    assert(fannkuch(7) == 16)
}
