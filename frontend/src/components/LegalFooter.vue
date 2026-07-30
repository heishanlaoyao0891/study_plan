<template>
  <view class="legal-footer-root">
    <!-- #ifdef H5 -->
    <view class="legal-footer">
      <a class="legal-link" :href="icpUrl" target="_blank" rel="noopener noreferrer">鄂ICP备2026038065号</a>
      <template v-if="publicSecurityRecordNo">
        <text class="legal-separator">|</text>
        <a class="legal-link police-link" :href="publicSecurityUrl" target="_blank" rel="noopener noreferrer">
          <image v-if="publicSecurityIcon" class="police-icon" :src="publicSecurityIcon" mode="aspectFit" />
          <text>{{ publicSecurityRecordNo }}</text>
        </a>
      </template>
    </view>
    <!-- #endif -->
  </view>
</template>

<script setup lang="ts">
const icpUrl = 'https://beian.miit.gov.cn/'
const publicSecurityRecordNo = ''
const publicSecurityUrl = ''
const publicSecurityIcon = ''
</script>

<style scoped lang="scss">
.legal-footer-root {
  width: 100%;
}

.legal-footer {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 10rpx;
  margin-top: 34rpx;
  padding: 18rpx 20rpx 0;
  color: #9a8f9b;
  font-size: 22rpx;
  line-height: 1.6;
  text-align: center;
}

.legal-link {
  color: #8b7d85;
  text-decoration: none;
}

.legal-link:hover {
  color: #e95f85;
  text-decoration: underline;
}

.legal-separator {
  color: #d8cbd3;
}

.police-link {
  display: inline-flex;
  align-items: center;
  gap: 6rpx;
}

.police-icon {
  width: 24rpx;
  height: 24rpx;
}

@media (min-width: 800px) {
  .legal-footer-root {
    grid-column: 1 / -1;
  }

  .legal-footer {
    margin-top: -10px;
    padding-top: 0;
    gap: 8px;
    font-size: 13px;
  }
}
</style>
