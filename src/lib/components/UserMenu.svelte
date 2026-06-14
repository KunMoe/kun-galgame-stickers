<script lang="ts">
  import Icon from '@iconify/svelte'
  import { page } from '$app/state'
  import { m } from '$lib/i18n'

  let open = $state(false)
  let showLogout = $state(false)
  let container: HTMLDivElement | undefined = $state()

  const user = $derived(page.data.user)
  const displayName = $derived(user?.name ?? user?.email ?? m().auth.anonymous)
  const initials = $derived(
    (user?.name ?? user?.email ?? '?').trim().slice(0, 1).toUpperCase()
  )
  const returnTo = $derived(page.url.pathname + page.url.search)
  const loginHref = $derived(`/auth/login?return_to=${encodeURIComponent(returnTo)}`)
  const registerHref = $derived(`/auth/register?return_to=${encodeURIComponent(returnTo)}`)
  const profileHref = $derived(
    `https://oauth.kungal.com/profile?return=${encodeURIComponent(page.url.href)}`
  )

  const handleBlur = () => {
    requestAnimationFrame(() => {
      if (container && !container.contains(document.activeElement)) open = false
    })
  }
</script>

{#if !user}
  <div class="flex items-center gap-1">
    <a
      href={registerHref}
      class="hidden h-9 items-center px-3 text-sm text-primary transition hover:bg-content2 sm:flex"
      data-sveltekit-preload-data="off"
    >
      {m().auth.register}
    </a>
    <a
      href={loginHref}
      class="flex h-9 items-center gap-2 px-3 text-sm text-primary transition hover:bg-content2"
      data-sveltekit-preload-data="off"
    >
      <Icon icon="line-md:log-in" class="text-xl" />
      <span class="hidden sm:inline">{m().auth.login}</span>
    </a>
  </div>
{:else}
  <div bind:this={container} class="relative" onfocusout={handleBlur}>
    <button
      type="button"
      aria-haspopup="menu"
      aria-expanded={open}
      onclick={() => (open = !open)}
      class="flex h-9 items-center gap-2 px-2 text-sm transition hover:bg-content2"
    >
      {#if user.picture}
        <img
          src={user.picture}
          alt=""
          width="28"
          height="28"
          referrerpolicy="no-referrer"
          class="h-7 w-7 object-cover"
        />
      {:else}
        <span
          aria-hidden="true"
          class="flex h-7 w-7 items-center justify-center bg-primary text-xs font-bold text-white"
        >
          {initials}
        </span>
      {/if}
      <span class="hidden max-w-32 truncate sm:inline">{displayName}</span>
    </button>

    {#if open}
      <div
        role="menu"
        class="absolute top-12 right-0 z-50 flex w-48 flex-col gap-1 border bg-content1 p-2 shadow-lg"
      >
        <div class="border-b px-3 py-2 text-sm">
          <p class="font-semibold">{displayName}</p>
          {#if user.email}
            <p class="truncate text-xs text-default-500">{user.email}</p>
          {/if}
        </div>

        <a
          href={profileHref}
          target="_blank"
          rel="noopener noreferrer"
          role="menuitem"
          class="flex items-center gap-2 px-3 py-1.5 text-sm transition hover:bg-content2"
        >
          <Icon icon="line-md:account" class="text-lg" />
          <span>{m().auth.profile}</span>
        </a>

        <button
          type="button"
          role="menuitem"
          onclick={() => {
            open = false
            showLogout = true
          }}
          class="flex items-center gap-2 px-3 py-1.5 text-left text-sm transition hover:bg-content2"
        >
          <Icon icon="line-md:log-out" class="text-lg" />
          <span>{m().auth.logout}</span>
        </button>
      </div>
    {/if}
  </div>

  {#if showLogout}
    <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
    <div
      class="fixed inset-0 z-[60] flex items-center justify-center bg-black/40 p-4"
      role="presentation"
      onclick={() => (showLogout = false)}
    >
      <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
      <div
        role="dialog"
        aria-modal="true"
        tabindex="-1"
        class="w-full max-w-md border bg-content1 p-5 shadow-xl"
        onclick={(event) => event.stopPropagation()}
      >
        <h2 class="text-base font-semibold">{m().auth.logoutTitle}</h2>
        <p class="mt-1 text-sm text-default-500">{m().auth.logoutPrompt}</p>

        <form method="POST" action="/auth/logout" class="mt-4 block">
          <input type="hidden" name="return_to" value={returnTo} />
          <input type="hidden" name="scope" value="everywhere" />
          <button
            type="submit"
            class="flex w-full flex-col gap-1 border border-primary p-3 text-left transition hover:bg-content2"
          >
            <span class="flex items-center gap-2 text-sm font-medium">
              <Icon icon="line-md:log-out" class="text-lg text-primary" />
              {m().auth.logoutEverywhere}
            </span>
            <span class="text-xs text-default-500">{m().auth.logoutEverywhereHint}</span>
          </button>
        </form>

        <form method="POST" action="/auth/logout" class="mt-3 block">
          <input type="hidden" name="return_to" value={returnTo} />
          <input type="hidden" name="scope" value="local" />
          <button
            type="submit"
            class="flex w-full flex-col gap-1 border p-3 text-left transition hover:bg-content2"
          >
            <span class="text-sm font-medium">{m().auth.logoutLocal}</span>
            <span class="text-xs text-default-500">{m().auth.logoutLocalHint}</span>
          </button>
        </form>

        <div class="mt-4 flex justify-end">
          <button
            type="button"
            onclick={() => (showLogout = false)}
            class="px-3 py-1.5 text-sm transition hover:bg-content2"
          >
            {m().auth.cancel}
          </button>
        </div>
      </div>
    </div>
  {/if}
{/if}
