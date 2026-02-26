export const manifest = (() => {
function __memo(fn) {
	let value;
	return () => value ??= (value = fn());
}

return {
	appDir: "_app",
	appPath: "_app",
	assets: new Set(["Inter-Medium.ttf","favicon.png","style.css","svelte.svg","wails.png"]),
	mimeTypes: {".ttf":"font/ttf",".png":"image/png",".css":"text/css",".svg":"image/svg+xml"},
	_: {
		client: {start:"_app/immutable/entry/start.BOyrKNLE.js",app:"_app/immutable/entry/app.D4vX9x5j.js",imports:["_app/immutable/entry/start.BOyrKNLE.js","_app/immutable/chunks/DZ-uHPa5.js","_app/immutable/chunks/B2QSKqcC.js","_app/immutable/entry/app.D4vX9x5j.js","_app/immutable/chunks/B2QSKqcC.js","_app/immutable/chunks/ByQpmFxN.js"],stylesheets:[],fonts:[],uses_env_dynamic_public:false},
		nodes: [
			__memo(() => import('./nodes/0.js')),
			__memo(() => import('./nodes/1.js'))
		],
		remotes: {
			
		},
		routes: [
			
		],
		prerendered_routes: new Set(["/"]),
		matchers: async () => {
			
			return {  };
		},
		server_assets: {}
	}
}
})();
