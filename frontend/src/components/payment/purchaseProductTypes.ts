export interface PurchaseProductMetric {
  label: string
  value: string
  tone?: 'muted' | 'strong'
}

export interface PurchaseProductViewModel {
  id: number
  name: string
  description: string
  detail?: string
  price: number
  original_price?: number
  tags: string[]
  features: string[]
}
