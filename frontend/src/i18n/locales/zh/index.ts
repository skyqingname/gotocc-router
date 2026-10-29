import workspace from './workspace'
import requestList from './requestList'
import featureExtensions from './featureExtensions'
import canvasCard from './canvasCard'
import creativeGotocc from './creativeGotocc'
import balanceCashback from './balanceCashback'
import adminAccessKeys from './adminAccessKeys'

import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import channelMonitorV2 from './channelMonitorV2'
import channelMonitorV3 from './channelMonitorV3'
import batchImage from './batchImage'
import asyncImage from './asyncImage'
import admin from './admin'
import misc from './misc'
import team from './team'
import gotocc from './gotocc'
import fingerprint from './fingerprint'
import inspection from './inspection'
import billingRates from './billingRates'

export default {
  ...workspace,
  ...requestList,
  ...inspection,
  ...billingRates,
  ...featureExtensions,
  ...canvasCard,
  ...balanceCashback,
  ...creativeGotocc,
  adminAccessKeys,

  ...landing,
  ...common,
  ...dashboard,
  ...channelMonitorV2,
  ...channelMonitorV3,
  ...batchImage,
  ...asyncImage,
  admin,
  ...misc,
  ...team,
  ...gotocc,
  ...fingerprint,
}
