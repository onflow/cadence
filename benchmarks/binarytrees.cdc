
access(all)
struct Tree {
    access(all)
    var left: Tree?

    access(all)
    var right: Tree?

    init(left: Tree?, right: Tree?) {
        self.left = left
        self.right = right
    }

    access(all)
    fun nodeCount(): Int {
        return 1
            + (self.left?.nodeCount() ?? 0)
            + (self.right?.nodeCount() ?? 0)
    }

    access(all)
    fun clear() {
        if (self.left != nil) {
            self.left?.clear()
            self.left = nil
            self.right?.clear()
            self.right = nil
        }
    }
}

access(all)
fun newTree(depth: Int): Tree {
    if depth == 0 {
        return Tree(left: nil, right: nil)
    }
    return Tree(
        left: newTree(depth: depth - 1),
        right: newTree(depth: depth - 1)
    )
}

access(all)
fun stretch(_ depth: Int) {
   log("stretch tree of depth \(depth), check: \(count(depth))")
}

access(all)
fun count(_ depth: Int): Int {
    let t = newTree(depth: depth)
    let c = t.nodeCount()
    t.clear()
    return c
}

access(all)
fun run(_ n: Int) {
    let minDepth = 4
    let maxDepth = minDepth + 2 > n ? minDepth + 2 : n
    let stretchDepth = maxDepth + 1

    stretch(stretchDepth)
    let longLivedTree = newTree(depth: maxDepth)

    for depth in InclusiveRange(minDepth, maxDepth, step: 2) {
        let iterations = 1 << (maxDepth - depth + minDepth)
        var sum = 0
        for _ in InclusiveRange(1, iterations, step: 1) {
            sum = sum + count(depth)
        }
        log("\(iterations), trees of depth \(depth), check: \(sum)")
    }
    let count = longLivedTree.nodeCount()
    longLivedTree.clear()
    log("long lived tree of depth \(maxDepth), check: \(count)")
}

access(all)
fun main() {
    run(10)
}
