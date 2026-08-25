<script lang="ts">
  import { navigating } from '$app/state'

  let visible = $state(false)
  let progress = $state(0)
  let timer: ReturnType<typeof setInterval> | null = null
  let hideTimer: ReturnType<typeof setTimeout> | null = null

  const start = () => {
    if (timer) clearInterval(timer)
    if (hideTimer) clearTimeout(hideTimer)
    visible = true
    progress = 0.08
    timer = setInterval(() => {
      const remaining = 0.99 - progress
      progress += Math.min(remaining * 0.08, 0.04)
      if (progress >= 0.99) {
        progress = 0.99
        if (timer) clearInterval(timer)
      }
    }, 200)
  }

  const finish = () => {
    if (timer) clearInterval(timer)
    progress = 1
    hideTimer = setTimeout(() => {
      visible = false
      progress = 0
    }, 350)
  }

  $effect(() => {
    if (navigating.to) start()
    else if (visible) finish()
  })
</script>

{#if visible}
  <output
    role="progressbar"
    aria-valuenow={Math.round(progress * 100)}
    aria-valuemin="0"
    aria-valuemax="100"
    class="bg-primary fixed inset-x-0 top-0 z-[9999] h-[3px] origin-left transition-[transform,opacity] duration-200 ease-out"
    style:transform="scaleX({progress})"
    style:opacity={progress === 1 ? 0 : 1}
  ></output>
{/if}
