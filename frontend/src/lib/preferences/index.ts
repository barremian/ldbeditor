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

import { initializeTheme } from "./theme";
import { initializeConfirmCloseLastTabPreference } from "./confirm-close-tab";
import { initializeValueFormatPreference } from "./value-format";
import { initializeKeyPaneWidthPreference } from "./pane-width";
import { initializeRecentPathsPreference } from "./recent-paths";

export function initializeAllPreferences(): void {
  initializeTheme();
  initializeConfirmCloseLastTabPreference();
  initializeValueFormatPreference();
  initializeKeyPaneWidthPreference();
  initializeRecentPathsPreference();
}
