import { describe, expect, it } from 'vitest'

import {
  assertImageInputFileSizes,
  assertImageInputFileSize,
  assertImageInputPayloadSize,
  assertReferenceImageCount,
  MAX_IMAGE_INPUT_FILE_BYTES,
  MAX_IMAGE_INPUT_PAYLOAD_BYTES,
  MAX_REFERENCE_IMAGE_COUNT,
} from './imageApiShared'

describe('managed image input limits', () => {
  it('matches the backend per-file and reference-count boundaries', () => {
    expect(MAX_IMAGE_INPUT_FILE_BYTES).toBe(20 * 1024 * 1024)
    expect(MAX_REFERENCE_IMAGE_COUNT).toBe(16)
    expect(() => assertReferenceImageCount(MAX_REFERENCE_IMAGE_COUNT)).not.toThrow()
    expect(() => assertReferenceImageCount(MAX_REFERENCE_IMAGE_COUNT + 1)).toThrow('16')
  })

  it('rejects an oversized reference image before sending a request', () => {
    expect(() => assertImageInputFileSize('参考图 1', MAX_IMAGE_INPUT_FILE_BYTES + 1)).toThrow('参考图 1')
    expect(() => assertImageInputFileSizes(['data:image/png;base64,QQ=='])).not.toThrow()
  })

  it('keeps total payloads below the backend 256 MiB request ceiling', () => {
    expect(MAX_IMAGE_INPUT_PAYLOAD_BYTES).toBe(240 * 1024 * 1024)
    expect(() => assertImageInputPayloadSize(MAX_IMAGE_INPUT_PAYLOAD_BYTES)).not.toThrow()
    expect(() => assertImageInputPayloadSize(MAX_IMAGE_INPUT_PAYLOAD_BYTES + 1)).toThrow('240.0 MiB')
  })
})
