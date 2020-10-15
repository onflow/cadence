const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin')
const path = require('path')
const HtmlWebpackPlugin = require("html-webpack-plugin");
const CopyPlugin = require('copy-webpack-plugin');

module.exports = {
    entry: [ "./src/index.ts"],
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
            {
                test: /\.ts$/,
                loader: "ts-loader"
            },
            {
                test: /\.css$/,
                use: ['style-loader', 'css-loader']
            },
            {
                test: /\.ttf$/,
                use: ['file-loader']
            },
        ]
    },
    plugins: [
        new MonacoWebpackPlugin({
            languages: []
        }),
        new HtmlWebpackPlugin({
            template: "./src/index.html",
        }),
        new CopyPlugin({
            patterns: [
                {
                    from: 'node_modules/@onflow/cadence-language-server/dist/cadence-language-server.wasm',
                    to: 'cadence-language-server.wasm'
                }
            ]
        }),
    ],
    devServer: {
        contentBase: path.join(__dirname, 'dist'),
        compress: true,
        port: 8000,
        writeToDisk: true
    },
    node: {
        crypto: 'empty',
        path: 'empty',
        os: 'empty',
        net: 'empty',
    }
}
