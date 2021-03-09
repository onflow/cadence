module.exports = {
	preset: "ts-jest",
	setupFiles: ["<rootDir>/tests_config/run_spec.js"],
	snapshotSerializers: ["jest-snapshot-serializer-raw"],
	testEnvironment: "node",
	transform: {
		".(ts|tsx)": "ts-jest",
	},
	testRegex: "jsfmt\\.spec\\.js$|__tests__/.*\\.js$|scripts/.*\\.test\\.js$",
	moduleFileExtensions: ["ts", "tsx", "js"],
	watchPlugins: [
		"jest-watch-typeahead/filename",
		"jest-watch-typeahead/testname",
	],
}
