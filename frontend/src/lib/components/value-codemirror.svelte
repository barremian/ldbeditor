<script lang="ts">
  import { createEventDispatcher, onDestroy, onMount } from "svelte";
  import { Annotation, Compartment, EditorState } from "@codemirror/state";
  import { history } from "@codemirror/commands";
  import { closeSearchPanel } from "@codemirror/search";
  import { EditorView, lineNumbers } from "@codemirror/view";
  import { cn } from "$lib/utils.js";
  import { searchExtensions } from "$lib/codemirror/search-panel";
  import { codemirrorTheme } from "$lib/codemirror/theme";
  import {
    clearReadOnlyTooltip,
    handleReadOnlyEditAttempt,
    isEditAttemptKey,
    readOnlyTooltipField,
  } from "$lib/codemirror/readonly-tooltip";

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
        ...searchExtensions,
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
          if (update.transactions.some((tx) => tx.annotation(parentValueSync))) {
            return;
          }

          const nextValue = update.state.doc.toString();
          if (nextValue === value) return;
          value = nextValue;
          dispatch("change", { value: nextValue });
        }),
        codemirrorTheme,
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
      clearReadOnlyTooltip(view);
    } else {
      closeSearchPanel(view);
    }

    appliedReadOnly = readOnly;
  }

  onDestroy(() => {
    if (!view) return;
    clearReadOnlyTooltip(view);
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
