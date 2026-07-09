import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import admin from './admin'
import misc from './misc'
import localOverrides from './local-overrides'
import { deepMergeLocale } from '../../deepMergeLocale'

const upstreamLocale = {
  ...landing,
  ...common,
  ...dashboard,
  admin,
  ...misc,
}

export default deepMergeLocale(upstreamLocale, localOverrides)
