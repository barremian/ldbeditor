export {
  initializeTheme,
  themePreference,
  type ThemePreference,
  THEME_STORAGE_KEY,
} from "./theme";
export {
  confirmCloseLastTabPreference,
  initializeConfirmCloseLastTabPreference,
} from "./confirm-close-tab";
export {
  valueFormatPreference,
  initializeValueFormatPreference,
  type ValueFormatPrefsMap,
} from "./value-format";
export {
  keyPaneWidthPreference,
  initializeKeyPaneWidthPreference,
} from "./pane-width";
export {
  recentPathsPreference,
  initializeRecentPathsPreference,
  type RecentPathItem,
  MAX_RECENT_PATHS,
} from "./recent-paths";
export {
  editorFontSizePreference,
  initializeEditorFontSizePreference,
  EDITOR_FONT_SIZE_OPTIONS,
  EDITOR_FONT_SIZE_STORAGE_KEY,
  EDITOR_FONT_SIZE_VALUES,
  type EditorFontSizeValue,
} from "./editor-font-size";

import { initializeTheme } from "./theme";
import { initializeConfirmCloseLastTabPreference } from "./confirm-close-tab";
import { initializeValueFormatPreference } from "./value-format";
import { initializeKeyPaneWidthPreference } from "./pane-width";
import { initializeRecentPathsPreference } from "./recent-paths";
import { initializeEditorFontSizePreference } from "./editor-font-size";

export function initializeAllPreferences(): void {
  initializeTheme();
  initializeConfirmCloseLastTabPreference();
  initializeValueFormatPreference();
  initializeKeyPaneWidthPreference();
  initializeRecentPathsPreference();
  initializeEditorFontSizePreference();
}
