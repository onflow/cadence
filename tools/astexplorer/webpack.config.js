const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin')
const path = require('path')
const HtmlWebpackPlugin = require("html-webpack-plugin");
const CopyPlugin = require('copy-webpack-plugin');

module.exports = {
  entry: [ "./src/index.tsx"],
  output: {
    filename: "bundle.js"
  },
  resolve: {
    extensions: [".ts", ".tsx", ".js"]
  },
  module: {
    rules: [
      {
        test: /\.tsx?$/,
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
          from: 'node_modules/@onflow/cadence-parser/dist/cadence-parser.wasm',
          to: 'cadence-parser.wasm'
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
