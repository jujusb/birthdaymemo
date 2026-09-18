<script setup lang="ts">
// Shown when the backend runs with `--guest-only`: only public share links
// (`#/s/:token`) are available. All other pages (login, calendar, settings,
// admin, ...) do not exist on the guest service by design.
</script>

<template>
  <div class="guest-blocked">
    <div class="card guest-blocked-card">
      <h1>404</h1>
      <p class="en">This page is not available on the public share service. Only share links can be opened here.</p>
      <p class="zh">该页面在公开分享服务上不可用，这里仅支持打开分享链接。</p>
    </div>
  </div>
</template>

<style scoped>
.guest-blocked {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
}
.guest-blocked-card {
  max-width: 480px;
  padding: 32px 28px;
  text-align: center;
}
.guest-blocked-card h1 {
  font-size: 40px;
  margin: 0 0 12px;
}
.guest-blocked-card .en {
  margin: 0 0 8px;
}
.guest-blocked-card .zh {
  margin: 0;
  color: var(--color-text-muted);
  font-size: 13px;
}
</style>
