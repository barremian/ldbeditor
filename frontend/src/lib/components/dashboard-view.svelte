<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { Database, DatabaseZap, FolderOpen, History, Plus, X } from "lucide-svelte";
  import type { RecentPathItem } from "$lib/preferences";
  import { Button } from "$lib/components/ui/button";
  import { DropdownMenuItem } from "$lib/components/ui/dropdown-menu";
  import { SplitButton } from "$lib/components/ui/split-button";
  import { ThreeDotMenu } from "$lib/components/ui/three-dot-menu";
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from "$lib/components/ui/card";

  export let pendingCreatePath: string | null = null;
  export let isOpeningDatabase = false;
  export let isCreatingDatabase = false;
  export let errorMessage = "";
  export let recentPaths: RecentPathItem[] = [];

  const dispatch = createEventDispatcher<{
    open: void;
    openReadOnly: void;
    create: void;
    cancelCreate: void;
    removeRecent: { path: string };
    openRecent: { path: string };
    openRecentReadOnly: { path: string };
  }>();
</script>

<div class="flex min-h-0 flex-1 items-center justify-center p-6">
  {#if pendingCreatePath}
    <div
      class="flex w-full max-w-2xl flex-col items-center gap-6 px-8 py-10 text-center"
    >
      <div
        class="flex h-20 w-20 items-center justify-center rounded-full border border-border/70 bg-muted/60 text-primary shadow-sm"
      >
        <DatabaseZap class="h-10 w-10" />
      </div>

      <div class="w-full space-y-3">
        <h1 class="text-2xl font-semibold tracking-tight">
          No LevelDB database found
        </h1>
        <p class="text-sm leading-6 text-muted-foreground">
          The selected folder does not contain a LevelDB database yet. Create a
          new database at this location to continue.
        </p>
        <div
          class="w-full max-w-full rounded-lg border border-border/70 bg-muted/35 px-4 py-3"
        >
          <p
            class="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground"
          >
            Selected folder
          </p>
          <p
            class="mt-2 max-w-full break-words font-mono text-sm text-foreground [overflow-wrap:anywhere]"
          >
            {pendingCreatePath}
          </p>
        </div>
      </div>

      <div class="flex flex-wrap items-center justify-center gap-3">
        <Button
          variant="outline"
          disabled={isCreatingDatabase}
          on:click={() => dispatch("cancelCreate")}
        >
          Cancel
        </Button>
        <Button
          disabled={isCreatingDatabase}
          on:click={() => dispatch("create")}
        >
          <Plus class="mr-2 h-4 w-4" />
          {isCreatingDatabase ? "Creating..." : "Create database here"}
        </Button>
      </div>
    </div>
  {:else}
    <Card class="w-full max-w-2xl">
      <CardHeader class="space-y-4">
        <div class="flex items-center justify-between">
          <CardTitle class="flex items-center gap-2 text-2xl">
            <Database class="h-6 w-6 text-primary" />
            LevelDB Editor
          </CardTitle>
        </div>
        <CardDescription>
          Open a LevelDB folder to browse keys and inspect values instantly.
        </CardDescription>
      </CardHeader>
      <CardContent class="space-y-6">
        <SplitButton
          class="w-fit"
          disabled={isOpeningDatabase}
          triggerLabel="More open options"
          on:primary={() => dispatch("open")}
        >
          <span class="inline-flex items-center gap-2">
            <FolderOpen class="h-4 w-4" />
            {isOpeningDatabase ? "Opening..." : "Open LevelDB database"}
          </span>

          <DropdownMenuItem
            slot="menu"
            disabled={isOpeningDatabase}
            on:click={() => dispatch("openReadOnly")}
          >
            Open in read-only mode
          </DropdownMenuItem>
        </SplitButton>

        {#if errorMessage}
          <p class="text-sm text-destructive">{errorMessage}</p>
        {/if}

        {#if recentPaths.length > 0}
          <section class="space-y-3">
            <h2
              class="flex items-center gap-2 text-sm font-medium text-muted-foreground"
            >
              <History class="h-4 w-4" />
              Recently opened
            </h2>
            <div class="space-y-2">
              {#each recentPaths as item}
                <div class="group flex items-center gap-1">
                  <Button
                    variant="ghost"
                    class="min-w-0 flex-1 justify-start font-normal"
                    on:click={() =>
                      dispatch("openRecent", { path: item.path })}
                    title={item.path}
                    disabled={isOpeningDatabase}
                  >
                    <span class="block truncate">{item.label}</span>
                  </Button>

                  <ThreeDotMenu
                    variant="ghost"
                    class="h-7 w-7 text-muted-foreground opacity-0 transition-opacity hover:text-foreground group-hover:opacity-100 group-focus-within:opacity-100"
                    triggerLabel={`More options for ${item.label}`}
                    triggerTitle="More options"
                    menuClass="min-w-56"
                    disabled={isOpeningDatabase}
                  >
                    <DropdownMenuItem
                      slot="menu"
                      disabled={isOpeningDatabase}
                      on:click={() =>
                        dispatch("openRecentReadOnly", { path: item.path })}
                    >
                      Open in read-only mode
                    </DropdownMenuItem>
                  </ThreeDotMenu>

                  <Button
                    variant="ghost"
                    size="icon"
                    class="h-7 w-7 text-muted-foreground opacity-0 transition-opacity hover:text-foreground group-hover:opacity-100 group-focus-within:opacity-100"
                    aria-label={`Remove ${item.label} from recently opened`}
                    title="Remove from recently opened"
                    on:click={() => dispatch("removeRecent", { path: item.path })}
                    disabled={isOpeningDatabase}
                  >
                    <X class="h-3.5 w-3.5" />
                  </Button>
                </div>
              {/each}
            </div>
          </section>
        {/if}
      </CardContent>
    </Card>
  {/if}
</div>
