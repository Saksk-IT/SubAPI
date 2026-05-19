import { reactive } from 'vue'
import { i18n } from '@/i18n'

export type NativeDialogVariant = 'default' | 'danger'
export type NativeDialogKind = 'alert' | 'confirm' | 'prompt'

export interface NativeDialogOptions {
  title?: string
  message: string
  confirmText?: string
  cancelText?: string
  variant?: NativeDialogVariant
  placeholder?: string
  defaultValue?: string
  inputType?: 'text' | 'password'
}

export interface NativeDialogRequest extends NativeDialogOptions {
  id: number
  kind: NativeDialogKind
  resolve: (value: boolean | string | null) => void
}

export const nativeDialogState = reactive({
  queue: [] as NativeDialogRequest[]
})

let requestId = 0

function translate(key: string): string {
  const value = i18n.global.t(key)
  return typeof value === 'string' ? value : key
}

function enqueueDialog(kind: NativeDialogKind, options: NativeDialogOptions) {
  return new Promise<boolean | string | null>((resolve) => {
    const request: NativeDialogRequest = {
      ...options,
      id: ++requestId,
      kind,
      resolve
    }
    nativeDialogState.queue = [...nativeDialogState.queue, request]
  })
}

export async function nativeAlert(
  message: string,
  options: Omit<NativeDialogOptions, 'message'> = {}
): Promise<void> {
  await enqueueDialog('alert', {
    title: options.title || translate('common.notice'),
    message,
    confirmText: options.confirmText || translate('common.confirm'),
    variant: options.variant || 'default'
  })
}

export async function nativeConfirm(
  message: string,
  options: Omit<NativeDialogOptions, 'message'> = {}
): Promise<boolean> {
  const result = await enqueueDialog('confirm', {
    title: options.title || translate('common.confirm'),
    message,
    confirmText: options.confirmText || translate('common.confirm'),
    cancelText: options.cancelText || translate('common.cancel'),
    variant: options.variant || 'default'
  })
  return result === true
}

export async function nativePrompt(
  message: string,
  options: Omit<NativeDialogOptions, 'message'> = {}
): Promise<string | null> {
  const result = await enqueueDialog('prompt', {
    title: options.title || translate('common.prompt'),
    message,
    confirmText: options.confirmText || translate('common.confirm'),
    cancelText: options.cancelText || translate('common.cancel'),
    variant: options.variant || 'default',
    placeholder: options.placeholder,
    defaultValue: options.defaultValue || '',
    inputType: options.inputType || 'text'
  })
  return typeof result === 'string' ? result : null
}

export function resolveNativeDialog(request: NativeDialogRequest, value: boolean | string | null): void {
  nativeDialogState.queue = nativeDialogState.queue.filter((item) => item.id !== request.id)
  request.resolve(value)
}
