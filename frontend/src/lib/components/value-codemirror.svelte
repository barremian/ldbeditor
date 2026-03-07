<script lang="ts">
  import { createEventDispatcher, onDestroy, onMount } from "svelte";
  import {
    Annotation,
    Compartment,
    EditorState,
    StateEffect,
    StateField,
  } from "@codemirror/state";
  import { history, historyKeymap, defaultKeymap } from "@codemirror/commands";
  import {
    search,
    searchKeymap,
    highlightSelectionMatches,
  } from "@codemirror/search";
  import {
    EditorView,
    keymap,
    lineNumbers,
    showTooltip,
    type Tooltip,
  } from "@codemirror/view";
  import { cn } from "$lib/utils.js";

  type $$Props = {
    value: string;
    readOnly: boolean;
    class?: string;
  };

  type $$Events = {
    change: { value: string };
  };

  export let value: $$Props["value"] = "";
  export let readOnly: $$Props["readOnly"] = true;

  let className: $$Props["class"] = undefined;
  export { className as class };

  const dispatch = createEventDispatcher<$$Events>();
  const parentValueSync = Annotation.define<boolean>();
  const readOnlyCompartment = new Compartment();
  const hideReadOnlyTooltipEffect = StateEffect.define<null>();
  const showReadOnlyTooltipEffect = StateEffect.define<{
    pos: number;
    message: string;
    above: boolean;
  }>();
  const readOnlyTooltipField = StateField.define<readonly Tooltip[]>({
    create: () => [],
    update(tooltips, transaction) {
      for (const effect of transaction.effects) {
        if (effect.is(hideReadOnlyTooltipEffect)) {
          return [];
        }
        if (effect.is(showReadOnlyTooltipEffect)) {
          const { pos, message, above } = effect.value;
          return [
            {
              pos,
              above,
              strictSide: false,
              arrow: true,
              create: () => {
                const dom = document.createElement("div");
                dom.className = "cm-tooltip-readonly-edit";
                dom.textContent = message;
                return { dom };
              },
            },
          ];
        }
      }
      return tooltips;
    },
    provide: (field) =>
      showTooltip.computeN([field], (state) => state.field(field)),
  });
  const readOnlyTooltipDurationMs = 1800;
  const readOnlyTooltipThrottleMs = 250;

  let hostElement: HTMLDivElement | null = null;
  let view: EditorView | null = null;
  let appliedReadOnly = readOnly;
  let readOnlyTooltipTimeout: ReturnType<typeof setTimeout> | null = null;
  let lastReadOnlyTooltipShownAt = 0;

  function clearReadOnlyTooltipTimeout() {
    if (!readOnlyTooltipTimeout) return;
    clearTimeout(readOnlyTooltipTimeout);
    readOnlyTooltipTimeout = null;
  }

  function hideReadOnlyTooltip() {
    if (!view) return;
    view.dispatch({ effects: [hideReadOnlyTooltipEffect.of(null)] });
  }

  function scheduleReadOnlyTooltipHide() {
    clearReadOnlyTooltipTimeout();
    readOnlyTooltipTimeout = setTimeout(() => {
      readOnlyTooltipTimeout = null;
      hideReadOnlyTooltip();
    }, readOnlyTooltipDurationMs);
  }

  function getReadOnlyTooltipMessage() {
    // Keep this fixed for read-only edit attempts.
    return "Cannot edit in read-only mode";
  }

  function shouldShowTooltipAbove(targetView: EditorView, pos: number) {
    const caretCoords = targetView.coordsAtPos(pos);
    if (!caretCoords) return true;

    const scrollerRect = targetView.scrollDOM.getBoundingClientRect();
    const minPreferredSpacePx = 40;
    const spaceAbove = caretCoords.top - scrollerRect.top;
    const spaceBelow = scrollerRect.bottom - caretCoords.bottom;

    if (spaceAbove < minPreferredSpacePx && spaceBelow > spaceAbove) {
      return false;
    }
    if (spaceBelow < minPreferredSpacePx && spaceAbove >= spaceBelow) {
      return true;
    }
    return spaceAbove >= spaceBelow;
  }

  function showReadOnlyTooltip(targetView: EditorView) {
    const now = Date.now();
    if (now - lastReadOnlyTooltipShownAt < readOnlyTooltipThrottleMs) return;
    lastReadOnlyTooltipShownAt = now;

    const pos = targetView.state.selection.main.head;
    targetView.dispatch({
      effects: [
        showReadOnlyTooltipEffect.of({
          pos,
          message: getReadOnlyTooltipMessage(),
          above: shouldShowTooltipAbove(targetView, pos),
        }),
      ],
    });
    scheduleReadOnlyTooltipHide();
  }

  function handleReadOnlyEditAttempt(
    targetView: EditorView,
    event?: KeyboardEvent | ClipboardEvent | DragEvent | InputEvent
  ) {
    if (!targetView.state.readOnly) return false;
    event?.preventDefault();
    showReadOnlyTooltip(targetView);
    return true;
  }

  function isEditAttemptKey(event: KeyboardEvent) {
    if (event.key === "Backspace" || event.key === "Delete") return true;
    if (event.metaKey || event.ctrlKey || event.altKey) return false;
    if (event.key.length === 1) return true;
    return event.key === "Enter";
  }

  function buildEditorState(doc: string) {
    return EditorState.create({
      doc,
      extensions: [
        lineNumbers(),
        history({ minDepth: 100 }),
        EditorView.lineWrapping,
        search({ top: true }),
        highlightSelectionMatches(),
        keymap.of([...searchKeymap, ...defaultKeymap, ...historyKeymap]),
        EditorView.contentAttributes.of({ spellcheck: "false" }),
        readOnlyCompartment.of(EditorState.readOnly.of(readOnly)),
        readOnlyTooltipField,
        EditorView.domEventHandlers({
          beforeinput: (event, targetView) =>
            handleReadOnlyEditAttempt(targetView, event),
          paste: (event, targetView) =>
            handleReadOnlyEditAttempt(targetView, event),
          drop: (event, targetView) =>
            handleReadOnlyEditAttempt(targetView, event),
          keydown: (event, targetView) => {
            if (!isEditAttemptKey(event)) return false;
            return handleReadOnlyEditAttempt(targetView, event);
          },
        }),
        EditorView.updateListener.of((update) => {
          if (!update.docChanged) return;
          if (
            update.transactions.some((tx) => tx.annotation(parentValueSync))
          ) {
            return;
          }
          const nextValue = update.state.doc.toString();
          if (nextValue === value) return;
          value = nextValue;
          dispatch("change", { value: nextValue });
        }),
        EditorView.theme({
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
            fontSize: "0.875rem",
            lineHeight: "1.625",
          },
          ".cm-gutterElement": {
            fontSize: "0.875rem",
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
          ".cm-panel.cm-search": {
            display: "grid",
            gridTemplateColumns:
              "minmax(12rem, 1fr) auto auto auto auto auto auto auto auto",
            alignItems: "center",
            columnGap: "0.5rem",
            rowGap: "0.4rem",
            padding: "0.45rem 0.6rem",
            backgroundColor: "transparent",
            color: "hsl(var(--foreground) / 1)",
          },
          ".cm-panel.cm-search label": {
            display: "inline-flex",
            alignItems: "center",
            gap: "0.35rem",
            fontSize: "0.75rem",
            color: "hsl(var(--muted-foreground) / 1)",
          },
          ".cm-panel.cm-search input:not([type='checkbox'])": {
            minHeight: "1.9rem",
            borderRadius: "0.375rem",
            border: "1px solid hsl(var(--input) / 1)",
            backgroundColor: "hsl(var(--background) / 1)",
            color: "hsl(var(--foreground) / 1)",
            padding: "0 0.55rem",
            fontSize: "0.8rem",
            lineHeight: "1.1",
            outline: "none",
            transition: "border-color 120ms ease, box-shadow 120ms ease",
          },
          ".cm-panel.cm-search input:not([type='checkbox'])::placeholder": {
            color: "hsl(var(--muted-foreground) / 0.85)",
          },
          ".cm-panel.cm-search input:not([type='checkbox']):focus": {
            borderColor: "hsl(var(--ring) / 1)",
            boxShadow: "0 0 0 2px hsl(var(--ring) / 0.3)",
          },
          ".cm-panel.cm-search input.cm-invalid": {
            borderColor: "hsl(var(--destructive) / 0.9)",
            boxShadow: "0 0 0 2px hsl(var(--destructive) / 0.2)",
          },
          ".cm-panel.cm-search input[type='checkbox']": {
            width: "0.85rem",
            height: "0.85rem",
            margin: "0",
            accentColor: "hsl(var(--primary) / 1)",
            transform: "translateY(-0.5px)",
          },
          ".cm-panel.cm-search input[name='search']": {
            gridColumn: "1",
            gridRow: "1",
            width: "100%",
          },
          ".cm-panel.cm-search button": {
            minHeight: "1.9rem",
            borderRadius: "0.375rem",
            border: "1px solid hsl(var(--border) / 1)",
            backgroundColor: "hsl(var(--background) / 1)",
            color: "hsl(var(--foreground) / 1)",
            padding: "0.15rem 0.55rem",
            fontSize: "0.75rem",
            lineHeight: "1.1",
            cursor: "pointer",
            transition:
              "background-color 120ms ease, border-color 120ms ease, box-shadow 120ms ease",
          },
          ".cm-panel.cm-search button:hover": {
            backgroundColor: "hsl(var(--accent) / 1)",
            color: "hsl(var(--accent-foreground) / 1)",
          },
          ".cm-panel.cm-search button:focus-visible": {
            borderColor: "hsl(var(--ring) / 1)",
            boxShadow: "0 0 0 2px hsl(var(--ring) / 0.3)",
            outline: "none",
          },
          ".cm-panel.cm-search button:disabled": {
            opacity: "0.55",
            cursor: "not-allowed",
          },
          ".cm-panel.cm-search .cm-searchMessage": {
            fontSize: "0.75rem",
            color: "hsl(var(--muted-foreground) / 1)",
            gridColumn: "1 / -1",
            gridRow: "3",
          },
          ".cm-panel.cm-search button[name='next']": {
            gridColumn: "2",
            gridRow: "1",
          },
          ".cm-panel.cm-search button[name='prev']": {
            gridColumn: "3",
            gridRow: "1",
          },
          ".cm-panel.cm-search button[name='select']": {
            gridColumn: "4",
            gridRow: "1",
          },
          ".cm-panel.cm-search button[name='close']": {
            gridColumn: "9",
            gridRow: "1",
            width: "2rem",
            minWidth: "2rem",
            minHeight: "2rem",
            padding: "0",
            display: "inline-flex",
            alignItems: "center",
            justifyContent: "center",
          },
          ".cm-panel.cm-search input[name='replace']": {
            gridColumn: "1",
            gridRow: "2",
            width: "100%",
          },
          ".cm-panel.cm-search button[name='replace']": {
            gridColumn: "2",
            gridRow: "2",
          },
          ".cm-panel.cm-search button[name='replaceAll']": {
            gridColumn: "3",
            gridRow: "2",
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
        }),
      ],
    });
  }

  function applyParentValue(nextValue: string) {
    if (!view) return;
    const currentValue = view.state.doc.toString();
    if (currentValue === nextValue) return;
    view.dispatch({
      changes: { from: 0, to: currentValue.length, insert: nextValue },
      annotations: parentValueSync.of(true),
    });
  }

  onMount(() => {
    if (!hostElement) return;
    view = new EditorView({
      state: buildEditorState(value ?? ""),
      parent: hostElement,
    });
    appliedReadOnly = readOnly;
  });

  $: if (view) {
    applyParentValue(value ?? "");
  }

  $: if (view && readOnly !== appliedReadOnly) {
    view.dispatch({
      effects: [
        readOnlyCompartment.reconfigure(EditorState.readOnly.of(readOnly)),
      ],
    });
    if (!readOnly) {
      clearReadOnlyTooltipTimeout();
      hideReadOnlyTooltip();
    }
    appliedReadOnly = readOnly;
  }

  onDestroy(() => {
    clearReadOnlyTooltipTimeout();
    if (!view) return;
    view.destroy();
    view = null;
  });
</script>

<div
  class={cn(
    "value-codemirror h-full min-h-0 flex-1 overflow-hidden",
    className
  )}
>
  <div bind:this={hostElement} class="h-full min-h-0 flex-1" />
</div>
