import { parse } from "./parse"
import { print } from "./print"
import { preprocess } from "./preprocess"

export const languages = [
	{
		name: "Cadence",
		parsers: ["cadence"],
		since: "1.0.0",
		extensions: [".cdc"],
		tmScope: "source.cadence",
		vscodeLanguageIds: ["cadence"],
	},
]

export const parsers = {
	cadence: {
		parse,
		astFormat: "cadence",
		locStart(node: any) {
			return node.StartPos.Offset
		},
		locEnd(node: any) {
			return node.EndPos.Offset
		},
	},
}

export const printers = {
	cadence: {
		preprocess,
		print,
	},
}
