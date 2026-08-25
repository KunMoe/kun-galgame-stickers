<script setup lang="ts">
definePageMeta({ layout: false })

const route = useRoute()
const localePath = useLocalePath()
const { t } = useI18n()
const error = ref('')

onMounted(async () => {
  const code = String(route.query.code ?? '')
  const returnedState = String(route.query.state ?? '')
  const savedState = useCookie('oauth_state')
  const codeVerifier = useCookie('oauth_code_verifier')

  const stateValue = savedState.value
  const verifierValue = codeVerifier.value
  savedState.value = null
  codeVerifier.value = null

  if (!code || returnedState !== stateValue || !verifierValue) {
    error.value = t('auth.callbackError')
    await navigateTo(localePath('/'))
    return
  }

  const user = await exchangeOAuthCode(code, verifierValue)
  if (user) {
    useAuthUser().value = user
    await navigateTo(consumeOAuthReturnTo())
    return
  }

  error.value = t('auth.callbackError')
  await navigateTo(localePath('/'))
})
</script>

<template>
  <div class="grid min-h-dvh place-items-center">
    <KunLoading :loading="true">
      <span>{{ error || t('auth.login') }}</span>
    </KunLoading>
  </div>
</template>
