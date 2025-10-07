<template>
  <div id="userLayout" class="w-full h-full relative">
    <div
      class="rounded-lg flex items-center justify-evenly w-full h-full md:w-screen md:h-screen md:bg-[#194bfb] bg-white"
    >
      <div class="md:w-3/5 w-10/12 h-full flex items-center justify-evenly">
        <div
          class="oblique h-[130%] w-3/5 bg-white dark:bg-slate-900 transform -rotate-12 absolute -ml-52"
        />
        <!-- 分割斜块 -->
        <div
          class="z-[999] pt-12 pb-10 md:w-96 w-full rounded-lg flex flex-col justify-between box-border"
        >
          <div>
            <div class="flex items-center justify-center">
              <Logo :size="6" />
            </div>
            <div class="mb-9">
              <p class="text-center text-4xl font-bold">
                {{ $GIN_VUE_ADMIN.appName }}
              </p>
              <p class="text-center text-sm font-normal text-gray-500 mt-2.5">
                A management platform using Golang and Vue
              </p>
            </div>
            <el-form
              ref="loginForm"
              :model="loginFormData"
              :rules="rules"
              :validate-on-rule-change="false"
              @keyup.enter="submitForm"
            >
              <el-form-item prop="username" class="mb-6">
                <el-input
                  v-model="loginFormData.username"
                  size="large"
                  placeholder="请输入用户名"
                  suffix-icon="user"
                />
              </el-form-item>
              <el-form-item prop="password" class="mb-6">
                <el-input
                  v-model="loginFormData.password"
                  show-password
                  size="large"
                  type="password"
                  placeholder="请输入密码"
                />
              </el-form-item>
              <el-form-item
                v-if="loginFormData.openCaptcha"
                prop="captcha"
                class="mb-6"
              >
                <div class="flex w-full justify-between">
                  <el-input
                    v-model="loginFormData.captcha"
                    placeholder="请输入验证码"
                    size="large"
                    class="flex-1 mr-5"
                  />
                  <div class="w-1/3 h-11 bg-[#c3d4f2] rounded">
                    <img
                      v-if="picPath"
                      class="w-full h-full"
                      :src="picPath"
                      alt="请输入验证码"
                      @click="loginVerify()"
                    />
                  </div>
                </div>
              </el-form-item>
              <el-form-item class="mb-6">
                <el-button
                  class="shadow shadow-active h-11 w-full"
                  type="primary"
                  size="large"
                  @click="submitForm"
                  >登 录</el-button>
              </el-form-item>
              <el-form-item class="mb-6">
                <el-button
                  class="shadow shadow-active h-11 w-full"
                  type="primary"
                  size="large"
                  @click="checkInit"
                  >前往初始化</el-button>
              </el-form-item>
            </el-form>

          <!-- OIDC第三方登录 -->
          <div v-if="showOidc" class="mt-6">
            <el-divider content-position="center">
              <span class="text-gray-500 text-sm">第三方登录</span>
            </el-divider>
            <OidcLogin />
          </div>
          </div>
        </div>
      </div>
      <div class="hidden md:block w-1/2 h-full float-right bg-[#194bfb]">
        <img
          class="h-full"
          src="@/assets/login_right_banner.jpg"
          alt="banner"
        />
      </div>
    </div>

    <BottomInfo class="left-0 right-0 absolute bottom-3 mx-auto w-full z-20">
      <div class="links items-center justify-center gap-2 hidden md:flex">
        <a href="https://www.gin-vue-admin.com/" target="_blank">
          <img src="@/assets/docs.png" class="w-8 h-8" alt="文档" />
        </a>
        <a href="https://support.qq.com/product/371961" target="_blank">
          <img src="@/assets/kefu.png" class="w-8 h-8" alt="客服" />
        </a>
        <a
          href="https://github.com/flipped-aurora/gin-vue-admin"
          target="_blank"
        >
          <img src="@/assets/github.png" class="w-8 h-8" alt="github" />
        </a>
        <a href="https://space.bilibili.com/322210472" target="_blank">
          <img src="@/assets/video.png" class="w-8 h-8" alt="视频站" />
        </a>
      </div>
    </BottomInfo>
  </div>
</template>

<script setup>
  import { captcha } from '@/api/user'
  import { checkDB } from '@/api/initdb'
  import BottomInfo from '@/components/bottomInfo/bottomInfo.vue'
  import OidcLogin from '@/components/oidc/oidcLogin.vue'
  import { reactive, ref, onMounted } from 'vue'
  import { ElMessage } from 'element-plus'
  import { useRouter, useRoute } from 'vue-router'
  import { useUserStore } from '@/pinia/modules/user'
  import { oidcCallback } from '@/api/oidc'
  import Logo from '@/components/logo/index.vue'

  defineOptions({
    name: 'Login'
  })

  const router = useRouter()
  const route = useRoute()
  const userStore = useUserStore()

  // 检查OIDC回调
  onMounted(async () => {
    console.log('Login page mounted, checking for OIDC callback...')
    console.log('Current URL:', window.location.href)
    console.log('Current route:', route.path)

    // 从URL中提取OIDC回调参数
    const url = new URL(window.location.href)
    const code = url.searchParams.get('code')
    const state = url.searchParams.get('state')

    if (code && state) {
      console.log('Detected OIDC callback parameters, processing...')
      ElMessage.info('正在处理登录回调...')

      try {
        // 处理OIDC回调
        const { data } = await oidcCallback({ code, state })
        console.log('OIDC callback response:', data)

        // 检查响应数据
        if (!data || !data.user || !data.token) {
          throw new Error('OIDC回调返回数据无效: ' + JSON.stringify(data))
        }

        // 设置用户信息和token
        await userStore.setUserInfo(data.user)
        userStore.setToken(data.token)
        console.log('User info and token set successfully')

        // 初始化路由信息
        console.log('Initializing async routes...')
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

        // 强制替换URL，移除hash中的login部分
        window.location.hash = ''

        // 检查是否有重定向地址
        if (router.currentRoute.value.query.redirect) {
          await router.replace(router.currentRoute.value.query.redirect)
          return
        }

        // 跳转到用户的默认页面
        if (!router.hasRoute(data.user.authority.defaultRouter)) {
          ElMessage.error('不存在可以登陆的首页，请联系管理员进行配置: '+ JSON.stringify(data.user))
          return
        }
        await router.replace({ name: data.user.authority.defaultRouter })
      } catch (error) {
        console.error('OIDC callback processing failed:', error)
        ElMessage.error('登录失败: ' + (error.message || '未知错误'))
      }
    }
  })
  const captchaRequiredLength = ref(6)
  // 验证函数
  const checkUsername = (rule, value, callback) => {
    if (value.length < 5) {
      return callback(new Error('请输入正确的用户名'))
    } else {
      callback()
    }
  }
  const checkPassword = (rule, value, callback) => {
    if (value.length < 6) {
      return callback(new Error('请输入正确的密码'))
    } else {
      callback()
    }
  }
  const checkCaptcha = (rule, value, callback) => {
    if (!loginFormData.openCaptcha) {
      return callback()
    }
    const sanitizedValue = (value || '').replace(/\s+/g, '')
    if (!sanitizedValue) {
      return callback(new Error('请输入验证码'))
    }
    if (!/^\d+$/.test(sanitizedValue)) {
      return callback(new Error('验证码须为数字'))
    }
    if (sanitizedValue.length < captchaRequiredLength.value) {
      return callback(
        new Error(`请输入至少${captchaRequiredLength.value}位数字验证码`)
      )
    }
    if (sanitizedValue !== value) {
      loginFormData.captcha = sanitizedValue
    }
    callback()
  }

  // 获取验证码
  const loginVerify = async () => {
    const ele = await captcha()
    captchaRequiredLength.value = Number(ele.data?.captchaLength) || 0
    picPath.value = ele.data?.picPath
    loginFormData.captchaId = ele.data?.captchaId
    loginFormData.openCaptcha = ele.data?.openCaptcha
  }
  loginVerify()

  // 登录相关操作
  const loginForm = ref(null)
  const picPath = ref('')
  const showOidc = ref(true) // 是否显示OIDC登录（默认开启用于测试）
  const loginFormData = reactive({
    username: 'admin',
    password: '',
    captcha: '',
    captchaId: '',
    openCaptcha: false
  })
  const rules = reactive({
    username: [{ validator: checkUsername, trigger: 'blur' }],
    password: [{ validator: checkPassword, trigger: 'blur' }],
    captcha: [{ validator: checkCaptcha, trigger: 'blur' }]
  })

  const login = async () => {
    return await userStore.LoginIn(loginFormData)
  }
  const submitForm = () => {
    loginForm.value.validate(async (v) => {
      if (!v) {
        // 未通过前端静态验证
        ElMessage({
          type: 'error',
          message: '请正确填写登录信息',
          showClose: true
        })
        return false
      }

      // 通过验证，请求登陆
      const flag = await login()

      // 登陆失败，刷新验证码
      if (!flag) {
        await loginVerify()
        return false
      }

      // 登陆成功
      return true
    })
  }

  // 跳转初始化
  const checkInit = async () => {
    const res = await checkDB()
    if (res.code === 0) {
      if (res.data?.needInit) {
        userStore.NeedInit()
        await router.push({ name: 'Init' })
      } else {
        ElMessage({
          type: 'info',
          message: '已配置数据库信息，无法初始化'
        })
      }
    }
  }
</script>
