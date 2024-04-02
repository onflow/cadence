const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin')
const HtmlWebpackPlugin = require("html-webpack-plugin");

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
  ],
  mode: 'development',
  devServer: {
    port: 8000,
    devMiddleware: {
      writeToDisk: true,
    },
    proxy: {
      '/api': 'http://127.0.0.1:3000',
    },
  },
}
