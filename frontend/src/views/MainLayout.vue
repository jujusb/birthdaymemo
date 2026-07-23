<script setup lang="ts">
import { RouterView } from 'vue-router'
import Topbar from '@/components/Topbar.vue'
import ToastContainer from '@/components/ToastContainer.vue'
</script>

<template>
  <div class="layout">
    <ToastContainer />
    <Topbar />
    <main class="content">
      <RouterView />
    </main>
  </div>
</template>

<style scoped>
.layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
}
.content {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 16px;
}

@media (max-width: 768px) {
  .content {
    padding: 12px;
  }
}
</style>
