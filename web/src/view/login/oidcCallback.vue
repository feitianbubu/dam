<template>
  <div class="oidc-callback">
    <div class="loading-container">
      <el-loading-directive />
      <p>正在处理登录...</p>
    </div>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { oidcCallback } from '@/api/oidc'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/pinia/modules/user'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

onMounted(async () => {
  console.log('OIDC callback mounted, processing...')

  try {
    // 从Vue Router query参数中获取回调参数
    const code = route.query.code
    const state = route.query.state

    console.log('Extracted code:', code ? 'exists' : 'missing')
    console.log('Extracted state:', state ? 'exists' : 'missing')

    if (!code || !state) {
      ElMessage.error('无效的回调参数，缺少code或state')
      setTimeout(() => {
        router.push('/login')
      }, 2000)
      return
    }

    // 处理OIDC回调
    console.log('Calling oidcCallback API...')
    const { data } = await oidcCallback({ code, state })
    console.log('OIDC callback response received')

    // 设置用户信息和token
    await userStore.setUserInfo(data.user)
    userStore.setToken(data.token)
    console.log('User info and token set successfully')

    // 初始化路由信息
    const { useRouterStore } = await import('@/pinia/modules/router')
    const routerStore = useRouterStore()
    await routerStore.SetAsyncRouter()
    const asyncRouters = routerStore.asyncRouters
    console.log('Async routes loaded:', asyncRouters.length)

    // 注册到路由表里
    asyncRouters.forEach((asyncRouter) => {
      router.addRoute(asyncRouter)
    })
    console.log('Routes registered to router')

    ElMessage.success('登录成功')

    // 强制替换URL，移除回调参数
    window.location.hash = ''

    // 检查是否有重定向地址
    if(router.currentRoute.value.query.redirect) {
      await router.replace(router.currentRoute.value.query.redirect)
      return
    }

    // 跳转到用户的默认页面
    if (!router.hasRoute(data.user.authority.defaultRouter)) {
      ElMessage.error('不存在可以登陆的首页，请联系管理员进行配置: ' + data.user.authority.defaultRouter)
      router.push('/layout/dashboard')
    } else {
      await router.replace({ name: data.user.authority.defaultRouter })
    }
  } catch (error) {
    console.error('OIDC callback processing failed:', error)
    ElMessage.error('登录失败: ' + (error.message || '未知错误'))
    router.push('/login')
  }
})
</script>

<style scoped>
.oidc-callback {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background-color: #f5f5f5;
}

.loading-container {
  text-align: center;
  padding: 40px;
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

.loading-container p {
  margin-top: 20px;
  color: #666;
}
</style>