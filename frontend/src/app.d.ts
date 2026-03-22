// Augment Wails custom events (merges with bindings generator output).
declare module "@wailsio/runtime" {
  namespace Events {
    interface CustomEvents {
      "app:closeActiveTabOrWindow": string;
      "app:saveValue": string;
    }
  }
}

// See https://kit.svelte.dev/docs/types#app
// for information about these interfaces
declare global {
  namespace App {
    // interface Error {}
    // interface Locals {}
    // interface PageData {}
    // interface PageState {}
    // interface Platform {}
  }
}

export {};
