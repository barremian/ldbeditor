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
