import { StateEffect, StateField } from "@codemirror/state";
import {
  EditorView,
  showTooltip,
  type Tooltip,
} from "@codemirror/view";

const readOnlyTooltipDurationMs = 1800;
const readOnlyTooltipThrottleMs = 250;

const tooltipTimeouts = new WeakMap<EditorView, ReturnType<typeof setTimeout>>();
const lastShownAt = new WeakMap<EditorView, number>();

export const hideReadOnlyTooltipEffect = StateEffect.define<null>();
export const showReadOnlyTooltipEffect = StateEffect.define<{
  pos: number;
  message: string;
  above: boolean;
}>();

export const readOnlyTooltipField = StateField.define<readonly Tooltip[]>({
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

function clearReadOnlyTooltipTimeout(view: EditorView) {
  const timeout = tooltipTimeouts.get(view);
  if (!timeout) return;

  clearTimeout(timeout);
  tooltipTimeouts.delete(view);
}

function hideReadOnlyTooltip(view: EditorView) {
  view.dispatch({ effects: [hideReadOnlyTooltipEffect.of(null)] });
}

function scheduleReadOnlyTooltipHide(view: EditorView) {
  clearReadOnlyTooltipTimeout(view);
  const timeout = setTimeout(() => {
    tooltipTimeouts.delete(view);
    hideReadOnlyTooltip(view);
  }, readOnlyTooltipDurationMs);
  tooltipTimeouts.set(view, timeout);
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

function showReadOnlyTooltip(view: EditorView) {
  const now = Date.now();
  const last = lastShownAt.get(view) ?? 0;
  if (now - last < readOnlyTooltipThrottleMs) return;

  lastShownAt.set(view, now);
  const pos = view.state.selection.main.head;
  view.dispatch({
    effects: [
      showReadOnlyTooltipEffect.of({
        pos,
        message: "Cannot edit in read-only mode",
        above: shouldShowTooltipAbove(view, pos),
      }),
    ],
  });

  scheduleReadOnlyTooltipHide(view);
}

export function handleReadOnlyEditAttempt(
  targetView: EditorView,
  event?: KeyboardEvent | ClipboardEvent | DragEvent | InputEvent
) {
  if (!targetView.state.readOnly) return false;
  event?.preventDefault();
  showReadOnlyTooltip(targetView);
  return true;
}

export function isEditAttemptKey(event: KeyboardEvent) {
  if (event.key === "Backspace" || event.key === "Delete") return true;
  if (event.metaKey || event.ctrlKey || event.altKey) return false;
  if (event.key.length === 1) return true;
  return event.key === "Enter";
}

export function clearReadOnlyTooltip(view: EditorView) {
  clearReadOnlyTooltipTimeout(view);
  hideReadOnlyTooltip(view);
}
