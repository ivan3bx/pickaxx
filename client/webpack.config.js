const path = require('path');
const MiniCssExtractPlugin = require('mini-css-extract-plugin');
const { CleanWebpackPlugin } = require('clean-webpack-plugin');
const webpack = require('webpack');

module.exports = {
    module: {
        rules: [
            {
                test: /\.js$/,
                exclude: /node_modules/
            },
            {
                test: /\.(png|jpe?g|gif)$/i,
                use: [
                    {
                        loader: 'file-loader',
                        options: {
                            name: '[name].[ext]'
                        }
                    }
                ]
            },
            {
                test: /\.(scss|css)$/,
                use: [
                    // devMode ? 'style-loader' : MiniCssExtractPlugin.loader, // inject CSS to page
                    // 'style-loader',
                    MiniCssExtractPlugin.loader,
                    // {
                    //     loader: 'file-loader',
                    //     options: {
                    //         name: '[name]_[contenthash].css',
                    //     }
                    // },
                    'css-loader', // translates CSS into CommonJS modules
                    'sass-loader', // compiles Sass to CSS
                ]
            },
        ]
    },
    entry: [
        "./src",
        // "./assets"
    ],
    output: {
        path: path.resolve(__dirname, './dist'),
        publicPath: '/assets/',
        filename: 'bundle.js'
    },
    plugins: [
        new CleanWebpackPlugin(),
        new webpack.ProvidePlugin({
            $: 'jquery',
            jQuery: 'jquery'
        }),
        new MiniCssExtractPlugin({
            filename: "styles.css",
        }),
    ],
    // stats: {
    //     modules: false,
    // },
    devServer: {
        contentBase: './dist'
    }
};
