<template>
  <view class="page">
    <view class="hero">
      <view class="kicker">你的学习名片</view>
      <view class="title">先取一个独一无二的昵称</view>
      <view class="desc">学习伙伴会通过昵称找到你。无需手机号，也不会展示你的微信身份。</view>
    </view>
    <view class="card">
      <view class="label">昵称</view>
      <input v-model="nickname" class="input" maxlength="40" placeholder="2-20 个字符" :focus="true" @input="error = ''" />
      <view class="hint">支持中英文与数字，请勿填写联系方式。</view>
      <view class="error" v-if="error">{{ error }}</view>
      <button class="primary" :loading="saving" :disabled="saving" @click="save">保存并开始学习</button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { AuthApi } from '@/api'
import { normalizeDisplayText, unicodeLength } from '@/utils/text'

const nickname = ref('')
const saving = ref(false)
const error = ref('')
const editMode = ref(false)

onLoad((query: any) => { editMode.value = query?.mode === 'edit' })

async function save() {
  const value = normalizeDisplayText(nickname.value)
  const length = unicodeLength(value)
  if (length < 2 || length > 20) {
    error.value = '昵称长度需要为 2-20 个字符'
    return
  }
  if (/[\u0000-\u001f\u007f]/.test(value)) {
    error.value = '昵称不能包含控制字符'
    return
  }
  saving.value = true
  try {
    await AuthApi.setNickname(value)
    uni.setStorageSync('nickname_completed', true)
    if (editMode.value) uni.navigateBack()
    else uni.reLaunch({ url: uni.getStorageSync('onboarding_completed') ? '/pages/checkin/checkin' : '/pages/onboarding/onboarding' })
  } catch (e: any) {
    error.value = e?.code === 409 ? '这个昵称已被使用，请换一个试试' : (e?.message || '昵称保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<style lang="scss">
.page { min-height: 100vh; box-sizing: border-box; padding: 52rpx 32rpx; background: linear-gradient(180deg, #fff0f7, #fffaf0 55%, #f7fbff); }
.hero { padding: 42rpx 34rpx; border-radius: 34rpx; background: linear-gradient(135deg, #ff8fab, #ffc36a); color: #fff; box-shadow: 0 20rpx 44rpx rgba(255,143,171,.25); }
.kicker { display: inline-block; padding: 8rpx 18rpx; border-radius: 99rpx; background: rgba(255,255,255,.24); font-size: 22rpx; font-weight: 800; }
.title { margin-top: 22rpx; font-size: 44rpx; line-height: 1.25; font-weight: 900; }
.desc { margin-top: 14rpx; color: rgba(255,255,255,.9); font-size: 25rpx; line-height: 1.6; }
.card { margin-top: 22rpx; padding: 34rpx; border-radius: 30rpx; background: #fff; border: 1rpx solid #ffe0ea; box-shadow: 0 14rpx 34rpx rgba(255,143,171,.12); }
.label { color: #4b2b3f; font-size: 27rpx; font-weight: 800; }
.input { box-sizing: border-box; width: 100%; height: 88rpx; margin-top: 14rpx; padding: 0 22rpx; border: 2rpx solid #ffd3e0; border-radius: 18rpx; background: #fff9fb; font-size: 29rpx; }
.hint { margin-top: 12rpx; color: #7b8498; font-size: 22rpx; }
.error { margin-top: 16rpx; color: #cf4359; font-size: 24rpx; }
.primary { margin-top: 30rpx; border-radius: 999rpx; background: linear-gradient(135deg, #ff7aa2, #ffb45c); color: #fff; font-size: 28rpx; font-weight: 800; }
</style>
