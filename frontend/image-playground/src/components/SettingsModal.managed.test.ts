import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const source = readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), './SettingsModal.tsx'), 'utf8')

describe('managed SettingsModal', () => {
  it('uses a managed-only API panel without exposing credential actions', () => {
    expect(source).toContain("import { getManagedSnapshot, isManagedMode } from '../lib/managedMode'")
    expect(source).toContain('const managedMode = isManagedMode()')
    expect(source).toContain('Sub2API 托管配置')
    expect(source).toContain('密钥由主站安全托管，不在此页面显示、编辑或复制')
    expect(source).toContain('managedMode ? (')
  })

  it('keeps model inputs editable but locks agent wiring and config backup controls', () => {
    expect(source).toContain('updateManagedProfileModel')
    expect(source).toContain('托管模式下 Agent 固定使用 Responses + Images 双配置')
    expect(source).toContain('const [exportConfig, setExportConfig] = useState(!managedMode)')
    expect(source).toContain('const [importConfig, setImportConfig] = useState(!managedMode)')
    expect(source).toContain('const [clearConfig, setClearConfig] = useState(!managedMode)')
  })
})
