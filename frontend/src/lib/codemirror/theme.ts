import { EditorView } from "@codemirror/view";

const searchToggleTooltipDelayMs = 90;

export const codemirrorTheme = EditorView.theme({
  "&": {
    height: "100%",
  },
  ".cm-scroller": {
    overflow: "auto",
    fontFamily:
      "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, Liberation Mono, Courier New, monospace",
  },
  ".cm-content": {
    padding: "0.5rem 0.5rem 0.5rem 0.75rem",
    fontSize: "var(--editor-font-size, 0.875rem)",
    lineHeight: "1.625",
    caretColor: "hsl(var(--foreground) / 1)",
  },
  ".cm-gutterElement": {
    fontSize: "var(--editor-font-size, 0.875rem)",
    lineHeight: "1.625",
  },
  ".cm-gutters": {
    borderRight: "1px solid hsl(var(--border) / 0.75)",
    backgroundColor: "transparent",
  },
  ".cm-lineNumbers": {
    minWidth: "2.5rem",
  },
  ".cm-lineNumbers .cm-gutterElement": {
    padding: "0 0.5rem 0 0.75rem",
  },
  ".cm-activeLineGutter": {
    backgroundColor: "transparent",
  },
  ".cm-panels": {
    backgroundColor: "hsl(var(--background) / 1)",
    color: "hsl(var(--foreground) / 1)",
  },
  ".cm-panels.cm-panels-top": {
    borderBottom: "1px solid hsl(var(--border) / 0.9)",
  },
  ".cm-panels.cm-panels-bottom": {
    borderTop: "1px solid hsl(var(--border) / 0.9)",
  },
  ".cm-panel.cm-search.cm-search-custom": {
    display: "flex",
    flexDirection: "column",
    rowGap: "0.45rem",
    padding: "0.5rem 0.6rem",
    backgroundColor: "transparent",
    color: "hsl(var(--foreground) / 1)",
  },
  ".cm-panel.cm-search.cm-search-custom .cm-searchRow": {
    display: "grid",
    gridTemplateColumns: "minmax(16rem, 1fr) auto auto auto",
    alignItems: "center",
    columnGap: "0.45rem",
    minWidth: "0",
  },
  ".cm-panel.cm-search.cm-search-custom .cm-searchInputWrap": {
    position: "relative",
    minWidth: "0",
  },
  ".cm-panel.cm-search.cm-search-custom .cm-searchInput": {
    width: "100%",
    minHeight: "1.9rem",
    borderRadius: "0.375rem",
    border: "1px solid hsl(var(--input) / 1)",
    backgroundColor: "hsl(var(--background) / 1)",
    color: "hsl(var(--foreground) / 1)",
    padding: "0 4.45rem 0 0.6rem",
    fontSize: "0.8rem",
    lineHeight: "1.1",
    outline: "none",
    transition: "border-color 120ms ease, box-shadow 120ms ease",
  },
  ".cm-panel.cm-search.cm-search-custom .cm-replaceInput": {
    width: "100%",
    minHeight: "1.9rem",
    borderRadius: "0.375rem",
    border: "1px solid hsl(var(--input) / 1)",
    backgroundColor: "hsl(var(--background) / 1)",
    color: "hsl(var(--foreground) / 1)",
    padding: "0 0.6rem",
    fontSize: "0.8rem",
    lineHeight: "1.1",
    outline: "none",
    transition: "border-color 120ms ease, box-shadow 120ms ease",
  },
  ".cm-panel.cm-search.cm-search-custom .cm-searchInput::placeholder, .cm-panel.cm-search.cm-search-custom .cm-replaceInput::placeholder":
    {
      color: "hsl(var(--muted-foreground) / 0.85)",
    },
  ".cm-panel.cm-search.cm-search-custom .cm-searchInput:focus, .cm-panel.cm-search.cm-search-custom .cm-replaceInput:focus":
    {
      borderColor: "hsl(var(--ring) / 1)",
      boxShadow: "0 0 0 2px hsl(var(--ring) / 0.3)",
    },
  ".cm-panel.cm-search.cm-search-custom .cm-searchInput.cm-invalid": {
    borderColor: "hsl(var(--destructive) / 0.9)",
    boxShadow: "0 0 0 2px hsl(var(--destructive) / 0.2)",
  },
  ".cm-panel.cm-search.cm-search-custom .cm-searchToggleGroup": {
    position: "absolute",
    right: "0.35rem",
    top: "50%",
    transform: "translateY(-50%)",
    display: "inline-flex",
    alignItems: "center",
    gap: "0.2rem",
  },
  ".cm-panel.cm-search.cm-search-custom .cm-searchToggle": {
    minWidth: "1.25rem",
    minHeight: "1.25rem",
    borderRadius: "0.25rem",
    border: "1px solid transparent",
    backgroundColor: "transparent",
    color: "hsl(var(--muted-foreground) / 1)",
    padding: "0 0.2rem",
    fontSize: "0.68rem",
    lineHeight: "1",
    cursor: "pointer",
    position: "relative",
    transition:
      "color 120ms ease, background-color 120ms ease, border-color 120ms ease",
  },
  ".cm-panel.cm-search.cm-search-custom .cm-searchToggle::after": {
    content: "attr(data-tooltip)",
    position: "absolute",
    left: "50%",
    top: "calc(100% + 0.35rem)",
    transform: "translate(-50%, -0.2rem)",
    borderRadius: "0.25rem",
    backgroundColor: "hsl(var(--popover) / 1)",
    color: "hsl(var(--popover-foreground) / 1)",
    border: "1px solid hsl(var(--primary) / 0.65)",
    boxShadow:
      "0 10px 20px hsl(var(--foreground) / 0.18), 0 0 0 1px hsl(var(--background) / 0.65)",
    padding: "0.3rem 0.55rem",
    fontSize: "0.8rem",
    fontWeight: "400",
    lineHeight: "1.2",
    whiteSpace: "nowrap",
    pointerEvents: "none",
    opacity: "0",
    visibility: "hidden",
    zIndex: "15",
    transition: `opacity 100ms ease ${searchToggleTooltipDelayMs}ms, transform 100ms ease ${searchToggleTooltipDelayMs}ms, visibility 0s linear ${searchToggleTooltipDelayMs}ms`,
  },
  ".cm-panel.cm-search.cm-search-custom .cm-searchToggle:hover::after, .cm-panel.cm-search.cm-search-custom .cm-searchToggle:focus-visible::after":
    {
      opacity: "1",
      visibility: "visible",
      transform: "translate(-50%, 0.05rem)",
      transitionDelay: "0ms",
    },
  ".cm-panel.cm-search.cm-search-custom .cm-searchToggle.cm-active": {
    color: "hsl(var(--primary-foreground) / 1)",
    backgroundColor: "hsl(var(--primary) / 1)",
    borderColor: "hsl(var(--primary) / 0.85)",
  },
  ".cm-panel.cm-search.cm-search-custom .cm-searchCounter": {
    minWidth: "4.1rem",
    textAlign: "right",
    fontSize: "0.75rem",
    color: "hsl(var(--muted-foreground) / 1)",
  },
  ".cm-panel.cm-search.cm-search-custom .cm-searchNav": {
    display: "inline-flex",
    alignItems: "center",
    gap: "0.25rem",
  },
  ".cm-panel.cm-search.cm-search-custom .cm-searchIconButton, .cm-panel.cm-search.cm-search-custom .cm-searchCloseButton, .cm-panel.cm-search.cm-search-custom .cm-searchReplaceButton":
    {
      minHeight: "1.9rem",
      borderRadius: "0.375rem",
      border: "1px solid hsl(var(--border) / 1)",
      backgroundColor: "hsl(var(--background) / 1)",
      color: "hsl(var(--foreground) / 1)",
      padding: "0 0.5rem",
      fontSize: "0.82rem",
      lineHeight: "1",
      cursor: "pointer",
      transition:
        "background-color 120ms ease, border-color 120ms ease, box-shadow 120ms ease",
    },
  ".cm-panel.cm-search.cm-search-custom .cm-searchIconButton": {
    minWidth: "1.9rem",
    padding: "0",
  },
  ".cm-panel.cm-search.cm-search-custom .cm-searchCloseButton": {
    minWidth: "1.9rem",
    padding: "0",
  },
  ".cm-panel.cm-search.cm-search-custom .cm-searchReplaceButton": {
    minWidth: "2.2rem",
    padding: "0 0.45rem",
    fontSize: "0.72rem",
    letterSpacing: "0.02em",
  },
  ".cm-panel.cm-search.cm-search-custom .cm-searchIconButton:hover, .cm-panel.cm-search.cm-search-custom .cm-searchCloseButton:hover, .cm-panel.cm-search.cm-search-custom .cm-searchReplaceButton:hover":
    {
      backgroundColor: "hsl(var(--accent) / 1)",
      color: "hsl(var(--accent-foreground) / 1)",
    },
  ".cm-panel.cm-search.cm-search-custom .cm-searchIconButton:focus-visible, .cm-panel.cm-search.cm-search-custom .cm-searchCloseButton:focus-visible, .cm-panel.cm-search.cm-search-custom .cm-searchReplaceButton:focus-visible, .cm-panel.cm-search.cm-search-custom .cm-searchToggle:focus-visible":
    {
      borderColor: "hsl(var(--ring) / 1)",
      boxShadow: "0 0 0 2px hsl(var(--ring) / 0.3)",
      outline: "none",
    },
  ".cm-panel.cm-search.cm-search-custom .cm-searchIconButton:disabled, .cm-panel.cm-search.cm-search-custom .cm-searchCloseButton:disabled, .cm-panel.cm-search.cm-search-custom .cm-searchReplaceButton:disabled":
    {
      opacity: "0.55",
      cursor: "not-allowed",
    },
  ".cm-panel.cm-search.cm-search-custom .cm-searchReplaceRow": {
    display: "grid",
    gridTemplateColumns: "minmax(16rem, 1fr) auto auto",
    alignItems: "center",
    columnGap: "0.45rem",
    minWidth: "0",
  },
  ".cm-tooltip.cm-tooltip-readonly-edit": {
    maxWidth: "22rem",
    border: "1px solid hsl(var(--primary) / 0.65)",
    borderRadius: "0.25rem",
    backgroundColor: "hsl(var(--popover) / 1)",
    color: "hsl(var(--popover-foreground) / 1)",
    padding: "0.3rem 0.55rem",
    fontSize: "0.8rem",
    fontWeight: "400",
    lineHeight: "1.2",
    boxShadow:
      "0 10px 20px hsl(var(--foreground) / 0.18), 0 0 0 1px hsl(var(--background) / 0.65)",
  },
  ".cm-tooltip.cm-tooltip-readonly-edit .cm-tooltip-arrow:before": {
    borderTopColor: "hsl(var(--primary) / 0.65)",
    borderBottomColor: "hsl(var(--primary) / 0.65)",
  },
  ".cm-tooltip.cm-tooltip-readonly-edit .cm-tooltip-arrow:after": {
    borderTopColor: "hsl(var(--popover) / 1)",
    borderBottomColor: "hsl(var(--popover) / 1)",
  },
  ".cm-cursor, .cm-dropCursor": {
    borderLeftColor: "hsl(var(--foreground) / 1)",
  },
  ".cm-fat-cursor": {
    backgroundColor: "hsl(var(--foreground) / 0.75)",
    color: "hsl(var(--background) / 1)",
  },
});
