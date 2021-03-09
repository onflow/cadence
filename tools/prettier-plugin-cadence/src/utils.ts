export function splitByLength(input: string, size: number): string[] {
	const result = input.split(" ").reduce(
		(acc: any, word: string): any => {
			const newSlice =
				acc.slice !== "" ? [acc.slice, word].join(" ") : `/// ${word}`
			if (newSlice.length < size) {
				acc.slice = newSlice
			} else {
				acc.arr.push(acc.slice)
				acc.slice = ""
			}
			return acc
		},
		{
			slice: "",
			arr: [],
		}
	)
	return result.arr
}
