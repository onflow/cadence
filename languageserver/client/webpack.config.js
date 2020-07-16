const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin');

module.exports = {
    mode: 'development',
    entry: [ "./src/wasm_exec.js", "./src/index.ts"],
    output: {
        filename: "bundle.js"
    },
    resolve: {
        extensions: [".ts", ".js"],
        alias: {
            vscode: require.resolve("monaco-languageclient/lib/vscode-compatibility")
        }
    },
    module: {
        rules: [
            { test: /\.ts$/, loader: "ts-loader" },
            {
                test: /\.css$/,
                use: ["style-loader", "css-loader"]
              },
              {
                test: /\.ttf$/,
                use: ["file-loader"]
              }
        ]
    },
    plugins: [
        new MonacoWebpackPlugin({
            languages: []
        })
    ],
    node: {
        net: 'empty'
    }
}
