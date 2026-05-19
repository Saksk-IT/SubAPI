<template>
  <BaseDialog
    v-if="activeDialog"
    :show="true"
    :title="activeDialog.title || fallbackTitle"
    width="narrow"
    :close-on-click-outside="false"
    :z-index="80"
    @close="handleCancel"
  >
    <form class="space-y-4" @submit.prevent="handleConfirm">
      <p class="whitespace-pre-line text-sm leading-6 text-gray-600 dark:text-gray-400">
        {{ activeDialog.message }}
      </p>
      <input
        v-if="activeDialog.kind === 'prompt'"
        ref="inputRef"
        v-model="promptValue"
        :type="activeDialog.inputType || 'text'"
        class="input"
        :placeholder="activeDialog.placeholder"
        autocomplete="off"
      />
    </form>

    <template #footer>
      <div class="flex w-full flex-col-reverse gap-2 sm:w-auto sm:flex-row sm:justify-end sm:gap-3">
        <button
          v-if="activeDialog.kind !== 'alert'"
          type="button"
          class="btn btn-secondary"
          @click="handleCancel"
        >
          {{ activeDialog.cancelText || t('common.cancel') }}
        </button>
        <button
          type="button"
          :class="[
            'btn',
            activeDialog.variant === 'danger' ? 'btn-danger' : 'btn-primary'
          ]"
          @click="handleConfirm"
        >
          {{ activeDialog.confirmText || t('common.confirm') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import {
  nativeDialogState,
  resolveNativeDialog,
  type NativeDialogRequest
} from '@/services/nativeDialog'

const { t } = useI18n()

const inputRef = ref<HTMLInputElement | null>(null)
const promptValue = ref('')

const activeDialog = computed<NativeDialogRequest | null>(() => nativeDialogState.queue[0] || null)
const fallbackTitle = computed(() => t('common.notice'))

watch(
  activeDialog,
  async (dialog) => {
    if (!dialog) return
    promptValue.value = dialog.defaultValue || ''
    if (dialog.kind === 'prompt') {
      await nextTick()
      inputRef.value?.focus()
      inputRef.value?.select()
    }
  },
  { immediate: true }
)

function handleConfirm(): void {
  const dialog = activeDialog.value
  if (!dialog) return

  const value = dialog.kind === 'prompt'
    ? promptValue.value
    : true
  resolveNativeDialog(dialog, value)
}

function handleCancel(): void {
  const dialog = activeDialog.value
  if (!dialog) return

  resolveNativeDialog(dialog, dialog.kind === 'alert' ? true : null)
}
</script>
