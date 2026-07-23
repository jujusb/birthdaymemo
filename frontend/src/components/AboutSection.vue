<script setup lang="ts">
</script>

<template>
  <section class="about-section card">
    <div class="about-logo">
      <img src="/logo512.png" alt="BirthDayMemo logo" />
    </div>
    <div class="about-info">
      <div class="about-title">🎂BirthDayMemo🎂</div>
      <div class="about-line">{{ '作者：mcBill' }}</div>
      <div class="about-line">
        <span>作者の小站：</span>
        <a href="https://mcbill.top" target="_blank" rel="noopener noreferrer">https://mcbill.top</a>
      </div>
      <div class="about-line">
        <span>Github：</span>
        <a href="https://github.com/mcbill1" target="_blank" rel="noopener noreferrer">https://github.com/mcbill1</a>
      </div>
    </div>
  </section>
</template>

<style scoped>
.about-section {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  margin-top: 16px;
}
.about-logo {
  flex-shrink: 0;
}
.about-logo img {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  object-fit: contain;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.12);
  display: block;
}
.about-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}
.about-title {
  font-size: 15px;
  font-weight: 700;
  margin-bottom: 4px;
}
.about-line {
  font-size: 13px;
  color: var(--color-text-muted);
  word-break: break-all;
}
.about-line a {
  color: var(--color-primary);
  text-decoration: none;
  word-break: break-all;
}
.about-line a:hover {
  text-decoration: underline;
}
@media (max-width: 480px) {
  .about-section {
    gap: 12px;
    padding: 12px;
  }
  .about-logo img {
    width: 48px;
    height: 48px;
  }
}
</style>
