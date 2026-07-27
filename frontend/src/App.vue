<script setup lang="ts">
import { onLaunch } from "@dcloudio/uni-app";
import { isLoggedIn } from "@/api/request";
import { AuthApi, StudyTaskApi } from "@/api";
import { unicodeLength } from "@/utils/text";
import { routeForUser } from "@/utils/auth-routing";
import { getBanState } from "@/utils/ban-state";
import { invitationFromLaunch, startMiniProgramAuth } from "@/utils/mp-auth";

onLaunch((options) => {
	if (getBanState()) {
		uni.reLaunch({ url: "/pages/banned/banned" });
		return;
	}
  // #ifdef MP-WEIXIN
  if (!isLoggedIn()) {
    uni.reLaunch({ url: "/pages/login/login" });
    startMiniProgramAuth(invitationFromLaunch(options));
    return;
  }
  // #endif

  // #ifndef MP-WEIXIN
  // 已登录 → 停留在 tab 首页；未登录 → 跳转登录页
  if (!isLoggedIn()) {
    // 这里用 reLaunch 而非 redirectTo/switchTab，因为它不在 tabBar 中
    uni.reLaunch({ url: "/pages/login/login" });
  } else {
    AuthApi.me().then(user => {
      if (!user.nickname || unicodeLength(user.nickname) < 2) {
        uni.reLaunch({ url: "/pages/nickname/nickname" });
        return;
      }
      const route = routeForUser(user)
      if (route !== '/pages/checkin/checkin') {
        uni.reLaunch({ url: route })
        return
      }
      StudyTaskApi.compensateMidnight().catch(() => {});
    }).catch(() => {});
  }
  // #endif

  // #ifdef MP-WEIXIN
  AuthApi.me().then(user => {
    const route = routeForUser(user)
    if (route !== '/pages/checkin/checkin') uni.reLaunch({ url: route })
    else StudyTaskApi.compensateMidnight().catch(() => {})
  }).catch(() => {})
  // #endif
});
</script>

<style>
/* 全局样式 */
</style>
