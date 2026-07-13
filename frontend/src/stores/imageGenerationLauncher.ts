import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useImageGenerationLauncherStore = defineStore('image-generation-launcher', () => {
  const isOpen = ref(false)

  function open(): void {
    isOpen.value = true
  }

  function close(): void {
    isOpen.value = false
  }

  return { isOpen, open, close }
})
