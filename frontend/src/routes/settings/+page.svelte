<script lang="ts">
  import { Browser } from "@wailsio/runtime";
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from "$lib/components/ui/card";
  import { Button } from "$lib/components/ui/button";
  import { Tabs, TabsContent, TabsList, TabsTrigger } from "$lib/components/ui/tabs";

  const DONATION_URL = "https://github.com/sponsors/barremian";

  let launchOnStartup = true;
  let openEmptyWindowMode = "new";
  let autosaveWithVersions = true;
  let externalChangesMode = "update";

  let preferredTheme = "system";
  let compactDensity = false;
  let showStatusBadges = true;

  let isOpeningDonation = false;

  async function openDonationPage() {
    isOpeningDonation = true;
    try {
      await Browser.OpenURL(DONATION_URL);
    } finally {
      isOpeningDonation = false;
    }
  }
</script>

<main class="flex min-h-screen flex-col bg-background text-foreground">
  <header class="border-b border-border/80 bg-muted/20 px-6 py-4">
    <h1 class="text-base font-semibold">Settings</h1>
    <p class="mt-1 text-xs text-muted-foreground">
      Configure LevelDB Editor preferences.
    </p>
  </header>

  <div class="min-h-0 flex-1 overflow-auto p-6">
    <Tabs value="general" class="mx-auto flex w-full max-w-3xl flex-col gap-6">
      <TabsList class="grid h-auto w-full grid-cols-3 gap-2 bg-transparent p-0">
        <TabsTrigger
          value="general"
          class="justify-center rounded-md border border-border/70 bg-background py-2 text-sm data-[state=active]:border-primary/40 data-[state=active]:bg-muted/70"
        >
          General
        </TabsTrigger>
        <TabsTrigger
          value="appearance"
          class="justify-center rounded-md border border-border/70 bg-background py-2 text-sm data-[state=active]:border-primary/40 data-[state=active]:bg-muted/70"
        >
          Appearance
        </TabsTrigger>
        <TabsTrigger
          value="donation"
          class="justify-center rounded-md border border-border/70 bg-background py-2 text-sm data-[state=active]:border-primary/40 data-[state=active]:bg-muted/70"
        >
          Donation
        </TabsTrigger>
      </TabsList>

      <TabsContent value="general" class="mt-0">
        <Card class="border-border/70">
          <CardHeader>
            <CardTitle class="text-base">General</CardTitle>
            <CardDescription>
              Startup and document behavior preferences.
            </CardDescription>
          </CardHeader>
          <CardContent class="space-y-6 text-sm">
            <label class="flex items-center gap-3">
              <input type="checkbox" bind:checked={launchOnStartup} />
              <span>Reopen windows from last session</span>
            </label>

            <div class="space-y-2">
              <p class="font-medium">When nothing else is open:</p>
              <select
                bind:value={openEmptyWindowMode}
                class="rounded-md border border-input bg-background px-3 py-2 text-sm"
              >
                <option value="new">Create New Document</option>
                <option value="open">Show Open Dialog</option>
                <option value="none">No Action</option>
              </select>
            </div>

            <label class="flex items-center gap-3">
              <input type="checkbox" bind:checked={autosaveWithVersions} />
              <span>Enable Auto Save with Versions</span>
            </label>

            <fieldset class="space-y-2">
              <legend class="font-medium">
                When document is changed by another application:
              </legend>
              <label class="flex items-center gap-2">
                <input
                  type="radio"
                  name="external-changes"
                  value="keep"
                  bind:group={externalChangesMode}
                />
                <span>Keep current editor content</span>
              </label>
              <label class="flex items-center gap-2">
                <input
                  type="radio"
                  name="external-changes"
                  value="ask"
                  bind:group={externalChangesMode}
                />
                <span>Ask how to resolve</span>
              </label>
              <label class="flex items-center gap-2">
                <input
                  type="radio"
                  name="external-changes"
                  value="update"
                  bind:group={externalChangesMode}
                />
                <span>Update to modified edition</span>
              </label>
            </fieldset>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="appearance" class="mt-0">
        <Card class="border-border/70">
          <CardHeader>
            <CardTitle class="text-base">Appearance</CardTitle>
            <CardDescription>
              Choose theme and visual density for the editor.
            </CardDescription>
          </CardHeader>
          <CardContent class="space-y-6 text-sm">
            <div class="space-y-2">
              <p class="font-medium">Theme</p>
              <select
                bind:value={preferredTheme}
                class="rounded-md border border-input bg-background px-3 py-2 text-sm"
              >
                <option value="system">System</option>
                <option value="light">Light</option>
                <option value="dark">Dark</option>
              </select>
            </div>

            <label class="flex items-center gap-3">
              <input type="checkbox" bind:checked={compactDensity} />
              <span>Use compact spacing in list views</span>
            </label>

            <label class="flex items-center gap-3">
              <input type="checkbox" bind:checked={showStatusBadges} />
              <span>Show status badges in tabs</span>
            </label>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="donation" class="mt-0">
        <Card class="border-border/70">
          <CardHeader>
            <CardTitle class="text-base">Donation</CardTitle>
            <CardDescription>
              Support continued development of LevelDB Editor.
            </CardDescription>
          </CardHeader>
          <CardContent class="space-y-4 text-sm">
            <p class="text-muted-foreground">
              If this app saves you time, supporting it helps keep improvements and
              maintenance moving.
            </p>
            <Button
              type="button"
              on:click={openDonationPage}
              disabled={isOpeningDonation}
            >
              {isOpeningDonation ? "Opening..." : "Open sponsorship page"}
            </Button>
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>
  </div>
</main>
