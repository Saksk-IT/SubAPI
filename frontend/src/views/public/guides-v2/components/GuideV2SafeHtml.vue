<script setup lang="ts">
defineProps<{ readonly html: string }>()
</script>

<template>
  <!-- html 只能来自 guide-v2.parser 的 DOMPurify 白名单输出。 -->
  <div class="guide-v2-rich-text" v-html="html" />
</template>
