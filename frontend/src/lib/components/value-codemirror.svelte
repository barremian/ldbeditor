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
    SearchQuery,
    closeSearchPanel,
    findNext,
    findPrevious,
    getSearchQuery,
    replaceAll,
    replaceNext,
    setSearchQuery,
  } from "@codemirror/search";
  import {
    EditorView,
    keymap,
    lineNumbers,
    runScopeHandlers,
    showTooltip,
    type Panel,
    type Tooltip,
    type ViewUpdate,
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

  const searchMatchCountLimit = 5000;

  type SearchMatchSummary = {
    current: number;
    total: number;
    overflow: boolean;
  };

  function getSearchMatchSummary(
    targetView: EditorView,
    query: SearchQuery
  ): SearchMatchSummary {
    if (!query.search || !query.valid) {
      return { current: 0, total: 0, overflow: false };
    }

    const mainSelection = targetView.state.selection.main;
    let total = 0;
    let current = 0;
    let nextFromSelection = 0;
    let overflow = false;

    const cursor = query.getCursor(targetView.state);
    while (true) {
      const step = cursor.next();
      if (step.done || !step.value) break;
      const match = step.value;
      total += 1;
      if (mainSelection.from === match.from && mainSelection.to === match.to) {
        current = total;
      }
      if (nextFromSelection === 0 && mainSelection.from <= match.from) {
        nextFromSelection = total;
      }
      if (total >= searchMatchCountLimit) {
        overflow = true;
        break;
      }
    }

    if (current === 0 && total > 0) {
      current = nextFromSelection || 1;
    }

    return { current, total, overflow };
  }

  class CustomSearchPanel implements Panel {
    dom: HTMLElement;
    searchField: HTMLInputElement;
    replaceField: HTMLInputElement | null = null;
    caseToggle: HTMLButtonElement;
    regexpToggle: HTMLButtonElement;
    wordToggle: HTMLButtonElement;
    matchSummary: HTMLSpanElement;
    previousButton: HTMLButtonElement;
    nextButton: HTMLButtonElement;
    replaceButton: HTMLButtonElement | null = null;
    replaceAllButton: HTMLButtonElement | null = null;
    query: SearchQuery;

    constructor(readonly view: EditorView) {
      this.query = getSearchQuery(view.state);
      this.commit = this.commit.bind(this);

      this.searchField = document.createElement("input");
      this.searchField.className = "cm-searchInput";
      this.searchField.name = "search";
      this.searchField.placeholder = "Find";
      this.searchField.setAttribute("aria-label", "Find");
      this.searchField.setAttribute("main-field", "true");
      this.searchField.addEventListener("input", this.commit);
      this.searchField.addEventListener("change", this.commit);

      this.caseToggle = this.createToggleButton("Aa", "Match case");
      this.regexpToggle = this.createToggleButton(".*", "Regex");
      this.wordToggle = this.createToggleButton("W", "Match whole word");

      const toggleGroup = document.createElement("div");
      toggleGroup.className = "cm-searchToggleGroup";
      toggleGroup.append(this.caseToggle, this.regexpToggle, this.wordToggle);

      const inputWrap = document.createElement("div");
      inputWrap.className = "cm-searchInputWrap";
      inputWrap.append(this.searchField, toggleGroup);

      this.matchSummary = document.createElement("span");
      this.matchSummary.className = "cm-searchCounter";
      this.matchSummary.textContent = "0/0";

      this.previousButton = this.createIconButton(
        "cm-searchIconButton",
        "‹",
        "Previous match",
        () => {
          findPrevious(this.view);
          this.refreshMatchSummary();
        }
      );
      this.nextButton = this.createIconButton(
        "cm-searchIconButton",
        "›",
        "Next match",
        () => {
          findNext(this.view);
          this.refreshMatchSummary();
        }
      );

      const navGroup = document.createElement("div");
      navGroup.className = "cm-searchNav";
      navGroup.append(this.previousButton, this.nextButton);

      const closeButton = this.createIconButton(
        "cm-searchCloseButton",
        "×",
        "Close search",
        () => {
          closeSearchPanel(this.view);
        }
      );

      const searchRow = document.createElement("div");
      searchRow.className = "cm-searchRow";
      searchRow.append(inputWrap, this.matchSummary, navGroup, closeButton);

      const panelRoot = document.createElement("div");
      panelRoot.className = "cm-search cm-search-custom";
      panelRoot.addEventListener("keydown", (event) => this.keydown(event));
      panelRoot.append(searchRow);

      if (!view.state.readOnly) {
        this.replaceField = document.createElement("input");
        this.replaceField.className = "cm-replaceInput";
        this.replaceField.name = "replace";
        this.replaceField.placeholder = "Replace";
        this.replaceField.setAttribute("aria-label", "Replace");
        this.replaceField.addEventListener("input", this.commit);
        this.replaceField.addEventListener("change", this.commit);

        this.replaceButton = this.createIconButton(
          "cm-searchReplaceButton",
          "↦",
          "Replace current match",
          () => replaceNext(this.view)
        );
        this.replaceAllButton = this.createIconButton(
          "cm-searchReplaceButton",
          "↦↦",
          "Replace all matches",
          () => replaceAll(this.view)
        );

        const replaceRow = document.createElement("div");
        replaceRow.className = "cm-searchReplaceRow";
        replaceRow.append(
          this.replaceField,
          this.replaceButton,
          this.replaceAllButton
        );
        panelRoot.append(replaceRow);
      }

      this.dom = panelRoot;
      this.setQuery(this.query);
      this.refreshMatchSummary();
    }

    private createToggleButton(label: string, ariaLabel: string) {
      return this.createIconButton(
        "cm-searchToggle",
        label,
        ariaLabel,
        (event) => {
          const target = event.currentTarget as HTMLButtonElement;
          this.setToggleState(target, !this.isToggleActive(target));
          this.commit();
        },
        true
      );
    }

    private createIconButton(
      className: string,
      text: string,
      ariaLabel: string,
      onClick: (event: MouseEvent) => void,
      toggle = false
    ) {
      const button = document.createElement("button");
      button.type = "button";
      button.className = className;
      button.textContent = text;
      button.title = ariaLabel;
      button.setAttribute("aria-label", ariaLabel);
      if (toggle) {
        button.setAttribute("aria-pressed", "false");
      }
      button.addEventListener("click", onClick);
      return button;
    }

    private isToggleActive(button: HTMLButtonElement) {
      return button.getAttribute("aria-pressed") === "true";
    }

    private setToggleState(button: HTMLButtonElement, active: boolean) {
      button.setAttribute("aria-pressed", active ? "true" : "false");
      button.classList.toggle("cm-active", active);
    }

    commit() {
      const query = new SearchQuery({
        search: this.searchField.value,
        replace: this.replaceField?.value ?? "",
        caseSensitive: this.isToggleActive(this.caseToggle),
        regexp: this.isToggleActive(this.regexpToggle),
        wholeWord: this.isToggleActive(this.wordToggle),
      });
      if (!query.eq(this.query)) {
        this.query = query;
        this.view.dispatch({ effects: setSearchQuery.of(query) });
      }
      this.refreshMatchSummary();
    }

    private keydown(event: KeyboardEvent) {
      if (runScopeHandlers(this.view, event, "search-panel")) {
        event.preventDefault();
      } else if (event.key === "Enter" && event.target === this.searchField) {
        event.preventDefault();
        (event.shiftKey ? findPrevious : findNext)(this.view);
      } else if (event.key === "Enter" && event.target === this.replaceField) {
        event.preventDefault();
        replaceNext(this.view);
      }
      this.refreshMatchSummary();
    }

    private refreshMatchSummary() {
      const summary = getSearchMatchSummary(this.view, this.query);
      const totalText = summary.overflow ? `${summary.total}+` : `${summary.total}`;
      this.matchSummary.textContent = `${summary.current}/${totalText}`;

      const hasValidSearch = this.query.valid && this.query.search.length > 0;
      const hasAnyMatch = hasValidSearch && summary.total > 0;
      this.previousButton.disabled = !hasAnyMatch;
      this.nextButton.disabled = !hasAnyMatch;
      if (this.replaceButton) this.replaceButton.disabled = !hasAnyMatch;
      if (this.replaceAllButton) this.replaceAllButton.disabled = !hasAnyMatch;

      this.searchField.classList.toggle(
        "cm-invalid",
        this.query.search.length > 0 && !this.query.valid
      );
    }

    update(update: ViewUpdate) {
      let handledQueryEffect = false;
      for (const transaction of update.transactions) {
        for (const effect of transaction.effects) {
          if (effect.is(setSearchQuery) && !effect.value.eq(this.query)) {
            this.setQuery(effect.value);
            handledQueryEffect = true;
          }
        }
      }
      if (handledQueryEffect || update.docChanged || update.selectionSet) {
        this.refreshMatchSummary();
      }
    }

    private setQuery(query: SearchQuery) {
      this.query = query;
      this.searchField.value = query.search;
      if (this.replaceField) {
        this.replaceField.value = query.replace;
      }
      this.setToggleState(this.caseToggle, query.caseSensitive);
      this.setToggleState(this.regexpToggle, query.regexp);
      this.setToggleState(this.wordToggle, query.wholeWord);
      this.searchField.classList.toggle(
        "cm-invalid",
        query.search.length > 0 && !query.valid
      );
    }

    mount() {
      this.searchField.select();
    }

    get top() {
      return true;
    }
  }

  function buildEditorState(doc: string) {
    return EditorState.create({
      doc,
      extensions: [
        lineNumbers(),
        history({ minDepth: 100 }),
        EditorView.lineWrapping,
        search({
          top: true,
          createPanel: (searchView) => new CustomSearchPanel(searchView),
        }),
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
            transition:
              "color 120ms ease, background-color 120ms ease, border-color 120ms ease",
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
