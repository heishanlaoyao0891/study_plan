<script setup lang="ts">
import { onLaunch } from "@dcloudio/uni-app";
import { isLoggedIn } from "@/api/request";
import { StudyTaskApi } from "@/api";

onLaunch(() => {
  // 已登录 → 停留在 tab 首页；未登录 → 跳转登录页
  if (!isLoggedIn()) {
    // 这里用 reLaunch 而非 redirectTo/switchTab，因为它不在 tabBar 中
    uni.reLaunch({ url: "/pages/login/login" });
	} else {
		StudyTaskApi.compensateMidnight().catch(() => {});
  }
});
</script>

<style>
/* 全局样式 */
</style>
