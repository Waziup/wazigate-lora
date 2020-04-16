const path = require("path");

module.exports = {
	mode: "production",

	devtool: "source-map",

	resolve: {
		extensions: [".ts", ".tsx", ".scss", ".css", ".js"]
	},

	entry: ["./src/hook.tsx"],

	plugins: [],

	module: {
		rules: [
			{
				test: /\.ts(x?)$/,
				exclude: /node_modules/,
				use: "ts-loader"
			},
			{
				test: /\.s[ac]ss$/i,
				use: ["style-loader", "css-loader", "sass-loader"],
				exclude: /node_modules/
			},
			{
				test: /\.woff(2)?(\?v=[0-9]\.[0-9]\.[0-9])?$/,
				loader: "url-loader?limit=10000&mimetype=application/font-woff"
			},
			{
				test: /\.(ttf|eot)(\?v=[0-9]\.[0-9]\.[0-9])?$/,
				loader: "file-loader"
			},
			{
				test: /\.(png|svg|jpg|gif)$/,
				loader: "file-loader",
				options: {
					publicPath: "img",
					outputPath: "img"
				}
			},
			{
				enforce: "pre",
				test: /\.js$/,
				loader: "source-map-loader"
			}
		]
	},

	output: {
		filename: "hook.js",
		path: path.resolve(__dirname, "dist")
	},

	externals: [
		{
			react: "React",
			"react-dom": "ReactDOM",
			"@material-ui/core": "MaterialUI",
			"@babel/runtime/helpers/interopRequireDefault": "BabelRuntimeHelpers.interopRequireDefault",
			"@babel/runtime/helpers/extends": "BabelRuntimeHelpers.extends",
		},
		(_, module, callback) => {
			match = module.match(/^@material-ui\/core\/(\w+)$/);
			if (match !== null) return callback(null, `MaterialUI.${match[1]}`);
			callback();
		} 
	]
};