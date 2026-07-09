import overview from './overview'
import channels from './channels'
import accounts from './accounts'
import resources from './resources'
import ops from './ops'
import settings from './settings'
import localOverrides from './local-overrides'
import { deepMergeLocale } from '../../../deepMergeLocale'

const upstreamAdminLocale = {
  ...overview,
  ...channels,
  ...accounts,
  ...resources,
  ...ops,
  ...settings,
}

export default deepMergeLocale(upstreamAdminLocale, localOverrides)
