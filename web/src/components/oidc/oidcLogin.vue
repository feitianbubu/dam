<template>
  <div class="oidc-login">
    <el-button
      v-for="provider in providers"
      :key="provider.name"
      :icon="provider.icon"
      @click="handleOidcLogin(provider.name)"
      class="oidc-btn"
      :loading="loading === provider.name"
    >
      {{ provider.label }}
    </el-button>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getOidcAuthURL } from '@/api/oidc'

const loading = ref('')

// 支持的OIDC提供商
const providers = ref([
  {
    name: 'clinx',
    label: 'Clinx登录',
    icon: 'Avatar'
  }
])

// 处理OIDC登录
const handleOidcLogin = async (provider) => {
  try {
    loading.value = provider

    // 获取授权URL
    const { data } = await getOidcAuthURL({ provider })

    if (data && data.authUrl) {
      ElMessage.info('正在跳转到授权页面...')

      // 直接跳转到授权页面，而不是使用弹窗
      window.location.href = data.authUrl
    }
  } catch (error) {
    loading.value = ''

    // 获取错误信息
    let errorMessage = '获取授权链接失败'
    if (error.response && error.response.data && error.response.data.msg) {
      errorMessage = error.response.data.msg
    } else if (error.message) {
      errorMessage = error.message
    }

    // 根据错误类型给出更友好的提示
    if (errorMessage.includes('连接失败') || errorMessage.includes('无法连接')) {
      ElMessage.error('OIDC服务提供方连接失败，请检查服务是否正常运行或联系管理员')
    } else if (errorMessage.includes('未启用')) {
      ElMessage.error('OIDC功能未启用，请联系管理员')
    } else if (errorMessage.includes('不支持')) {
      ElMessage.error('不支持的OIDC服务提供方，请联系管理员')
    } else {
      ElMessage.error('登录失败: ' + errorMessage)
    }
  }
}
</script>

<style scoped>
.oidc-login {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin: 20px 0;
}

.oidc-btn {
  width: 100%;
  justify-content: flex-start;
}

.oidc-btn:hover {
  transform: translateY(-1px);
  transition: all 0.2s ease;
}
</style>