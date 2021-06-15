module.exports = {
  testEnvironment: 'node',
  "transform": {
      "^.+\\.[tj]s$": "ts-jest"
  },
  setupFilesAfterEnv: [ './tests/setup.js' ],
  testPathIgnorePatterns: ["/node_modules/", "/dist/"]
};
