

export const index = 1;
let component_cache;
export const component = async () => component_cache ??= (await import('../entries/fallbacks/error.svelte.js')).default;
export const imports = ["_app/immutable/nodes/1.DFBH8t8r.js","_app/immutable/chunks/B2QSKqcC.js","_app/immutable/chunks/ByQpmFxN.js","_app/immutable/chunks/DZ-uHPa5.js"];
export const stylesheets = [];
export const fonts = [];
