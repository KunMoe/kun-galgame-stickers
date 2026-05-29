<script lang="ts">
  import Icon from '@iconify/svelte'
  import { page } from '$app/state'
  import { m } from '$lib/i18n'

  let open = $state(false)
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

        <form method="POST" action="/auth/logout" class="contents">
          <input type="hidden" name="return_to" value={returnTo} />
          <button
            type="submit"
            role="menuitem"
            class="flex items-center gap-2 px-3 py-1.5 text-left text-sm transition hover:bg-content2"
          >
            <Icon icon="line-md:log-out" class="text-lg" />
            <span>{m().auth.logout}</span>
          </button>
        </form>
      </div>
    {/if}
  </div>
{/if}
