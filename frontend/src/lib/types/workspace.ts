export type WorkspaceTab =
  | { id: string; type: "dashboard"; title: string }
  | { id: string; type: "database"; title: string; path: string };

export type PendingUnsavedClose = {
  tabId: string;
  closeWindowAfter: boolean;
};

export type TabViewState = {
  keys: string[];
  selectedKey: string | null;
  keySearchInput: string;
  debouncedKeySearch: string;
  keyListScrollTop: number | null;
  originalValueRaw: string;
  editorValueRaw: string;
  isValueEditing: boolean;
};

export function getTabLabel(path: string): string {
  return path.split(/[/\\]/).pop() || path;
}
