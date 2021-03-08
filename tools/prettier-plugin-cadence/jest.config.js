module.exports = {
    preset: 'ts-jest',
    setupFiles: ['<rootDir>/tests_config/run_spec.js'],
    snapshotSerializers: ['<rootDir>/tests_config/raw-serializer.js'],
    testEnvironment: 'node',
    transform: {
        ".(ts|tsx)": "<rootDir>/node_modules/ts-jest/preprocessor.js"
    },
    testRegex: 'jsfmt\\.spec\\.js$|__tests__/.*\\.js$|scripts/.*\\.test\\.js$',
    moduleFileExtensions: ["ts", "tsx", "js"],
    watchPlugins: [
        'jest-watch-typeahead/filename',
        'jest-watch-typeahead/testname'
    ],
};
