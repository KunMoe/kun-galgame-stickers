export default defineNuxtRouteMiddleware((to) => {
  const user = useAuthUser()
  if (user.value) return
  if (import.meta.client) {
    startOAuthLogin(to.fullPath)
    return abortNavigation()
  }
  return navigateTo('/')
})
