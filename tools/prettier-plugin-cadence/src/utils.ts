import { doc } from "prettier"
const { line, group, softline } = doc.builders

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

function smartJoin(separator: string, parts: any[]) {
	let result: any[];
	let lastPart: any;

	result = [];

	parts.filter(x => x).forEach((part, index) => {
		const firstPart = part.parts ? part.parts[0] : part;
		if (
			index > 0 &&
			firstPart !== ":" &&
			firstPart !== "..." &&
			firstPart !== ")" &&
			firstPart !== "<" &&
			lastPart !== "("
		) {
			result.push(separator);
		}

		lastPart = part;
		result.push(part);
	});

	return concat(result);
}

function concat(parts: any[]) {
	parts = parts.filter(x => x);

	switch (parts.length) {
		case 0:
			return "";
		case 1:
			return parts[0];
		default:
			return doc.builders.concat(parts);
	}
}

export function printList(path: any, print: any) {
	return concat([
		softline,
		group(smartJoin(concat([",", line]), path.map(print, "layout"))),
	])
}
