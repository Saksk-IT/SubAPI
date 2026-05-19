import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, beforeEach, vi } from 'vitest'
import { createI18n } from 'vue-i18n'
import NativeDialogHost from '@/components/common/NativeDialogHost.vue'
import { nativeConfirm, nativeDialogState, nativePrompt } from '@/services/nativeDialog'

vi.mock('@/i18n', () => ({
  i18n: {
    global: {
      t: (key: string) => {
        const messages: Record<string, string> = {
          'common.cancel': '取消',
          'common.confirm': '确认',
          'common.notice': '提示',
          'common.prompt': '输入'
        }
        return messages[key] || key
      }
    }
  }
}))

const i18n = createI18n({
  legacy: false,
  locale: 'zh',
  messages: {
    zh: {
      common: {
        cancel: '取消',
        confirm: '确认',
        notice: '提示',
        prompt: '输入'
      }
    }
  }
})

function mountHost() {
  return mount(NativeDialogHost, {
    global: {
      plugins: [i18n],
      stubs: {
        BaseDialog: {
          props: ['show', 'title'],
          template: '<section v-if="show"><h2>{{ title }}</h2><slot /><slot name="footer" /></section>'
        }
      }
    }
  })
}

describe('NativeDialogHost', () => {
  beforeEach(() => {
    nativeDialogState.queue = []
  })

  it('resolves confirm as true after confirm button click', async () => {
    const wrapper = mountHost()
    const result = nativeConfirm('确认执行？')

    await flushPromises()
    expect(wrapper.text()).toContain('确认执行？')

    await wrapper.findAll('button').at(-1)!.trigger('click')
    await expect(result).resolves.toBe(true)
  })

  it('resolves confirm as false after cancel button click', async () => {
    const wrapper = mountHost()
    const result = nativeConfirm('确认取消？')

    await flushPromises()
    await wrapper.find('button').trigger('click')

    await expect(result).resolves.toBe(false)
  })

  it('returns prompt input value on confirm', async () => {
    const wrapper = mountHost()
    const result = nativePrompt('请输入密码', { inputType: 'password' })

    await flushPromises()
    await wrapper.find('input').setValue('secret')
    await wrapper.findAll('button').at(-1)!.trigger('click')

    await expect(result).resolves.toBe('secret')
  })
})
