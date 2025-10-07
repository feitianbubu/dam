import service from '@/utils/request'

// 获取OIDC授权URL
export function getOidcAuthURL(data) {
  return service({
    url: '/oidc/auth',
    method: 'post',
    data
  })
}

// OIDC回调处理
export function oidcCallback(data) {
  return service({
    url: '/oidc/callback',
    method: 'post',
    data
  })
}

// 获取用户的OIDC绑定
export function getOidcUsers() {
  return service({
    url: '/oidc/users',
    method: 'get'
  })
}

// 解绑OIDC
export function unlinkOidc(provider) {
  return service({
    url: `/oidc/unlink/${provider}`,
    method: 'delete'
  })
}

// 获取OIDC登出URL
export function getOidcLogoutURL() {
  return service({
    url: '/oidc/logout-url',
    method: 'get'
  })
}