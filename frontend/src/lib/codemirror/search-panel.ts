import { historyKeymap, defaultKeymap } from "@codemirror/commands";
import {
  closeSearchPanel,
  findNext,
  findPrevious,
  getSearchQuery,
  highlightSelectionMatches,
  replaceAll,
  replaceNext,
  search,
  SearchQuery,
  searchKeymap,
  setSearchQuery,
} from "@codemirror/search";
import { keymap } from "@codemirror/view";
import {
  EditorView,
  runScopeHandlers,
  type Panel,
  type ViewUpdate,
} from "@codemirror/view";

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
    button.setAttribute("aria-label", ariaLabel);

    if (toggle) {
      button.setAttribute("aria-pressed", "false");
      button.setAttribute("data-tooltip", ariaLabel);
    } else {
      button.title = ariaLabel;
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
      event.stopPropagation();
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

export const searchExtensions = [
  search({
    top: true,
    createPanel: (searchView) => new CustomSearchPanel(searchView),
  }),
  highlightSelectionMatches(),
  keymap.of([...searchKeymap, ...defaultKeymap, ...historyKeymap]),
];
