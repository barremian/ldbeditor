<script lang="ts">
  import { createEventDispatcher, onDestroy, onMount } from "svelte";
  import { Annotation, Compartment, EditorState } from "@codemirror/state";
  import { history, historyKeymap, defaultKeymap } from "@codemirror/commands";
  import { searchKeymap, highlightSelectionMatches } from "@codemirror/search";
  import { EditorView, keymap, lineNumbers } from "@codemirror/view";
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

  let hostElement: HTMLDivElement | null = null;
  let view: EditorView | null = null;
  let appliedReadOnly = readOnly;

  function buildEditorState(doc: string) {
    return EditorState.create({
      doc,
      extensions: [
        lineNumbers(),
        history({ minDepth: 100 }),
        EditorView.lineWrapping,
        highlightSelectionMatches(),
        keymap.of([...searchKeymap, ...defaultKeymap, ...historyKeymap]),
        EditorView.contentAttributes.of({ spellcheck: "false" }),
        readOnlyCompartment.of(EditorState.readOnly.of(readOnly)),
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
    appliedReadOnly = readOnly;
  }

  onDestroy(() => {
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
