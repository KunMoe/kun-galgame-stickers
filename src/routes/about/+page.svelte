<script lang="ts">
  import { m } from '$lib/i18n'

  const packIds = [1, 2, 3, 4, 5, 6, 7]
</script>

<svelte:head>
  <title>{m().about.title}</title>
  <meta name="description" content={m().about.description} />
</svelte:head>

<article class="flex flex-col gap-12">
  <h1 class="text-center text-3xl font-bold">
    <a
      href="https://github.com/KUN1007/kun-galgame-stickers-sveltekit"
      target="_blank"
      rel="noopener noreferrer"
      class="text-primary hover:text-primary-600 underline underline-offset-4"
    >
      {m().about.repo}
    </a>
  </h1>

  <section class="flex flex-col gap-3">
    <h2 class="text-2xl font-semibold">
      <span class="text-primary">#</span>
      {m().about.introduction.heading}
    </h2>
    {#each m().about.introduction.lines as line (line)}
      <p>{line}</p>
    {/each}

    <video poster="/title.webp" controls loop playsinline class="w-full border">
      <source src="/introduction.mp4" type="video/mp4" />
    </video>

    <p class="mt-4">{m().about.introduction.bilibiliPrefix}</p>
    <p>
      Bilibili:
      <a
        href="https://www.bilibili.com/video/BV19u4m1c7ZN"
        class="text-primary hover:text-primary-600 underline underline-offset-4"
      >
        {m().about.introduction.bilibiliLink}
      </a>
    </p>

    <p class="mt-4">{m().about.introduction.purposeIntro}</p>
    <ul class="list-disc pl-6">
      {#each m().about.introduction.purpose as item (item)}
        <li>{item}</li>
      {/each}
    </ul>

    <p class="mt-4">{m().about.introduction.future}</p>
  </section>

  <section class="flex flex-col gap-3">
    <h2 class="text-2xl font-semibold">
      <span class="text-primary">#</span>
      {m().about.telegram.heading}
    </h2>
    <p>{m().about.telegram.intro}</p>
    <ul class="list-disc pl-6">
      {#each packIds as id (id)}
        <li>
          <a
            href="https://t.me/addstickers/KUNgal{id}"
            target="_blank"
            rel="noopener noreferrer"
            class="text-primary hover:text-primary-600 underline underline-offset-4"
          >
            {m().about.telegram.packLabel} [{id}]
          </a>
        </li>
      {/each}
    </ul>
    <p>{m().about.telegram.note}</p>
    <p>{m().about.telegram.downloadIntro}</p>
    <a
      href="https://github.com/KUN1007/kun-galgame-stickers-sveltekit/releases"
      target="_blank"
      rel="noopener noreferrer"
      class="text-primary hover:text-primary-600 underline underline-offset-4"
    >
      {m().about.telegram.downloadLink}
    </a>
  </section>

  <section class="flex flex-col gap-3">
    <h2 class="text-2xl font-semibold">
      <span class="text-primary">#</span>
      {m().about.rules.heading}
    </h2>
    <p>{m().about.rules.intro}</p>
    <ul class="list-disc pl-6">
      {#each m().about.rules.items as item (item)}
        <li>{item}</li>
      {/each}
    </ul>
    <strong>{m().about.rules.note}</strong>
  </section>

  <section class="flex flex-col gap-3">
    <h2 class="text-2xl font-semibold">
      <span class="text-primary">#</span>
      {m().about.games.heading}
    </h2>
    {#each m().about.games.lines as line (line)}
      <p>{line}</p>
    {/each}
    <a
      href="https://github.com/KUN1007/kungalgame-stickers/blob/main/introduction/game.md"
      target="_blank"
      rel="noopener noreferrer"
      class="text-primary hover:text-primary-600 underline underline-offset-4"
    >
      {m().about.games.linkLabel}
    </a>
  </section>

  <section class="flex flex-col gap-4">
    <h2 class="text-2xl font-semibold">
      <span class="text-primary">#</span>
      {m().about.faq.heading}
    </h2>
    {#each m().about.faq.items as item (item.q)}
      <div class="flex flex-col gap-1">
        <p class="text-primary font-bold">Q: {item.q}</p>
        <p>A: {item.a}</p>
      </div>
    {/each}
  </section>

  <section class="flex flex-col gap-2">
    <h2 class="text-2xl font-semibold">
      <span class="text-primary">#</span>
      {m().about.tips.heading}
    </h2>
    {#each m().about.tips.lines as line (line)}
      <p>{line}</p>
    {/each}
    <p class="text-default-500">{m().about.tips.note}</p>
  </section>
</article>
