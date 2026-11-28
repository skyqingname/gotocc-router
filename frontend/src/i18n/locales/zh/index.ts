import workspace from './workspace'
import requestList from './requestList'
import featureExtensions from './featureExtensions'
import canvasCard from './canvasCard'
import balanceCashback from './balanceCashback'
import adminAccessKeys from './adminAccessKeys'
import codexCli from './codexCli'
import codexCliRuntime from './codexCliRuntime'

import mcp from './mcp'
import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import channelMonitorV2 from './channelMonitorV2'
import channelMonitorV3 from './channelMonitorV3'
import asyncImage from './asyncImage'
import admin from './admin'
import misc from './misc'
import team from './team'
import gotocc from './gotocc'
import fingerprint from './fingerprint'
import inspection from './inspection'
import billingRates from './billingRates'
import nativeAccess from './nativeAccess'
import keyConsole from './keyConsole'
import consoleGuide from './consoleGuide'
import consoleTips from './consoleTips'
import accountHub from './accountHub'
import checkin from './checkin'
import serviceStatus from './serviceStatus'

export default {
  ...workspace,
  ...requestList,
  ...inspection,
  ...billingRates,
  ...nativeAccess,
  ...keyConsole,
  ...consoleGuide,
  ...consoleTips,
  ...accountHub,
  ...checkin,
  ...serviceStatus,
  ...featureExtensions,
  ...canvasCard,
  ...balanceCashback,
  ...codexCli,
  ...codexCliRuntime,
  adminAccessKeys,

  mcp,
  ...landing,
  ...common,
  ...dashboard,
  ...channelMonitorV2,
  ...channelMonitorV3,
  ...asyncImage,
  admin,
  ...misc,
  ...team,
  ...gotocc,
  ...fingerprint,
}
