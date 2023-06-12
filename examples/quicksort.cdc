
/// quickSort is qsort from "The C Programming Language".
///
/// > Our version of quicksort is not the fastest possible,
/// > but it's one of the simplest.
///
pub fun quickSort(_ items: &[AnyStruct], isLess: ((Int, Int): Bool)) {

    fun quickSortPart(leftIndex: Int, rightIndex: Int) {

        if leftIndex >= rightIndex {
            return
        }

        let pivotIndex = (leftIndex + rightIndex) / 2

        items[pivotIndex] <-> items[leftIndex]

        var lastIndex = leftIndex
        var index = leftIndex + 1
        while index <= rightIndex {
            if isLess(index, leftIndex) {
                lastIndex = lastIndex + 1
                items[lastIndex] <-> items[index]
            }
            index = index + 1
        }

        items[leftIndex] <-> items[lastIndex]

        quickSortPart(leftIndex: leftIndex, rightIndex: lastIndex - 1)
        quickSortPart(leftIndex: lastIndex + 1, rightIndex: rightIndex)
    }

    quickSortPart(
        leftIndex: 0,
        rightIndex: items.length - 1
    )
}

pub fun main() {
    let items = [5, 3, 7, 6, 2, 9]
    quickSort(
        &items as &[AnyStruct],
        isLess: fun (i: Int, j: Int): Bool {
            return items[i] < items[j]
        }
    )
    log(items)
}
