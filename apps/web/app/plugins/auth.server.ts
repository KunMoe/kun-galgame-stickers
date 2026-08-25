export default defineNuxtPlugin(async () => {
  const user = useAuthUser()
  user.value = await fetchMe()
})
