import { DriveStep } from 'driver.js'

export interface RoutedDriveStep extends DriveStep {
  route?: {
    path: string
    query?: Record<string, string>
  }
}

/**
 * 管理员完整引导流程
 * 交互式引导：指引用户实际操作
 * @param t 国际化函数
 * @param isSimpleMode 是否为简易模式（简易模式下会过滤分组相关步骤）
 */
export const getAdminSteps = (t: (key: string) => string, isSimpleMode = false): DriveStep[] => {
  const allSteps: DriveStep[] = [
  // ========== 欢迎介绍 ==========
  {
    popover: {
      title: t('onboarding.admin.welcome.title'),
      description: t('onboarding.admin.welcome.description'),
      align: 'center',
      nextBtnText: t('onboarding.admin.welcome.nextBtn'),
      prevBtnText: t('onboarding.admin.welcome.prevBtn')
    }
  },

  // ========== 第一部分：创建分组 ==========
  {
    element: '#sidebar-group-manage',
    popover: {
      title: t('onboarding.admin.groupManage.title'),
      description: t('onboarding.admin.groupManage.description'),
      side: 'right',
      align: 'center',
      showButtons: ['close'],
    }
  },
  {
    element: '[data-tour="groups-create-btn"]',
    popover: {
      title: t('onboarding.admin.createGroup.title'),
      description: t('onboarding.admin.createGroup.description'),
      side: 'bottom',
      align: 'end',
      showButtons: ['close']
    }
  },
  {
    element: '[data-tour="group-form-name"]',
    popover: {
      title: t('onboarding.admin.groupName.title'),
      description: t('onboarding.admin.groupName.description'),
      side: 'right',
      align: 'start',
      showButtons: ['next', 'previous']
    }
  },
  {
    element: '[data-tour="group-form-platform"]',
    popover: {
      title: t('onboarding.admin.groupPlatform.title'),
      description: t('onboarding.admin.groupPlatform.description'),
      side: 'right',
      align: 'start',
      showButtons: ['next', 'previous']
    }
  },
  {
    element: '[data-tour="group-form-multiplier"]',
    popover: {
      title: t('onboarding.admin.groupMultiplier.title'),
      description: t('onboarding.admin.groupMultiplier.description'),
      side: 'right',
      align: 'start',
      showButtons: ['next', 'previous']
    }
  },
  {
    element: '[data-tour="group-form-exclusive"]',
    popover: {
      title: t('onboarding.admin.groupExclusive.title'),
      description: t('onboarding.admin.groupExclusive.description'),
      side: 'top',
      align: 'start',
      showButtons: ['next', 'previous']
    }
  },
  {
    element: '[data-tour="group-form-submit"]',
    popover: {
      title: t('onboarding.admin.groupSubmit.title'),
      description: t('onboarding.admin.groupSubmit.description'),
      side: 'left',
      align: 'center',
      showButtons: ['close']
    }
  },

  // ========== 第二部分：创建账号授权 ==========
  {
    element: '#sidebar-channel-manage',
    popover: {
      title: t('onboarding.admin.accountManage.title'),
      description: t('onboarding.admin.accountManage.description'),
      side: 'right',
      align: 'center',
      showButtons: ['close']
    }
  },
  {
    element: '[data-tour="accounts-create-btn"]',
    popover: {
      title: t('onboarding.admin.createAccount.title'),
      description: t('onboarding.admin.createAccount.description'),
      side: 'bottom',
      align: 'end',
      showButtons: ['close']
    }
  },
  {
    element: '[data-tour="account-form-name"]',
    popover: {
      title: t('onboarding.admin.accountName.title'),
      description: t('onboarding.admin.accountName.description'),
      side: 'right',
      align: 'start',
      showButtons: ['next', 'previous']
    }
  },
  {
    element: '[data-tour="account-form-platform"]',
    popover: {
      title: t('onboarding.admin.accountPlatform.title'),
      description: t('onboarding.admin.accountPlatform.description'),
      side: 'right',
      align: 'start',
      showButtons: ['next', 'previous']
    }
  },
  {
    element: '[data-tour="account-form-type"]',
    popover: {
      title: t('onboarding.admin.accountType.title'),
      description: t('onboarding.admin.accountType.description'),
      side: 'right',
      align: 'start',
      showButtons: ['next', 'previous']
    }
  },
  {
    element: '[data-tour="account-form-priority"]',
    popover: {
      title: t('onboarding.admin.accountPriority.title'),
      description: t('onboarding.admin.accountPriority.description'),
      side: 'top',
      align: 'start',
      showButtons: ['next', 'previous']
    }
  },
  {
    element: '[data-tour="account-form-groups"]',
    popover: {
      title: t('onboarding.admin.accountGroups.title'),
      description: t('onboarding.admin.accountGroups.description'),
      side: 'top',
      align: 'center',
      showButtons: ['next', 'previous']
    }
  },
  {
    element: '[data-tour="account-form-submit"]',
    popover: {
      title: t('onboarding.admin.accountSubmit.title'),
      description: t('onboarding.admin.accountSubmit.description'),
      side: 'left',
      align: 'center',
      showButtons: ['close']
    }
  },

  // ========== 第三部分：创建API密钥 ==========
  {
    element: '[data-tour="sidebar-my-keys"]',
    popover: {
      title: t('onboarding.admin.keyManage.title'),
      description: t('onboarding.admin.keyManage.description'),
      side: 'right',
      align: 'center',
      showButtons: ['close']
    }
  },
  {
    element: '[data-tour="keys-create-btn"]',
    popover: {
      title: t('onboarding.admin.createKey.title'),
      description: t('onboarding.admin.createKey.description'),
      side: 'bottom',
      align: 'end',
      showButtons: ['close']
    }
  },
  {
    element: '[data-tour="key-form-name"]',
    popover: {
      title: t('onboarding.admin.keyName.title'),
      description: t('onboarding.admin.keyName.description'),
      side: 'right',
      align: 'start',
      showButtons: ['next', 'previous']
    }
  },
  {
    element: '[data-tour="key-form-group"]',
    popover: {
      title: t('onboarding.admin.keyGroup.title'),
      description: t('onboarding.admin.keyGroup.description'),
      side: 'right',
      align: 'start',
      showButtons: ['next', 'previous']
    }
  },
  {
    element: '[data-tour="key-form-submit"]',
    popover: {
      title: t('onboarding.admin.keySubmit.title'),
      description: t('onboarding.admin.keySubmit.description'),
      side: 'left',
      align: 'center',
      showButtons: ['close']
    }
  }
  ]

  // 简易模式下过滤分组相关步骤
  if (isSimpleMode) {
    return allSteps.filter(step => {
      const element = step.element as string | undefined
      // 过滤掉分组管理和账号分组选择相关步骤
      return !element || (
        !element.includes('sidebar-group-manage') &&
        !element.includes('groups-create-btn') &&
        !element.includes('group-form-') &&
        !element.includes('account-form-groups')
      )
    })
  }

  return allSteps
}

/**
 * 普通用户引导流程
 */
export const getUserSteps = (t: (key: string) => string): DriveStep[] => [
  {
    popover: {
      title: t('onboarding.user.welcome.title'),
      description: t('onboarding.user.welcome.description'),
      align: 'center',
      nextBtnText: t('onboarding.user.welcome.nextBtn'),
      prevBtnText: t('onboarding.user.welcome.prevBtn')
    }
  },
  {
    element: '[data-tour="sidebar-my-keys"]',
    popover: {
      title: t('onboarding.user.keyManage.title'),
      description: t('onboarding.user.keyManage.description'),
      side: 'right',
      align: 'center',
      showButtons: ['close']
    }
  },
  {
    element: '[data-tour="keys-create-btn"]',
    popover: {
      title: t('onboarding.user.createKey.title'),
      description: t('onboarding.user.createKey.description'),
      side: 'bottom',
      align: 'end',
      showButtons: ['close']
    }
  },
  {
    element: '[data-tour="key-form-name"]',
    popover: {
      title: t('onboarding.user.keyName.title'),
      description: t('onboarding.user.keyName.description'),
      side: 'right',
      align: 'start',
      showButtons: ['next', 'previous']
    }
  },
  {
    element: '[data-tour="key-form-group"]',
    popover: {
      title: t('onboarding.user.keyGroup.title'),
      description: t('onboarding.user.keyGroup.description'),
      side: 'right',
      align: 'start',
      showButtons: ['next', 'previous']
    }
  },
  {
    element: '[data-tour="key-form-submit"]',
    popover: {
      title: t('onboarding.user.keySubmit.title'),
      description: t('onboarding.user.keySubmit.description'),
      side: 'left',
      align: 'center',
      showButtons: ['close']
    }
  }
]

/**
 * 团队功能导览只展示入口和只读区域，不触发任何数据写入操作。
 */
export const getTeamSteps = (t: (key: string) => string, isOwner: boolean, hasTeam = true): RoutedDriveStep[] => {
  if (!hasTeam) {
    return [
      {
        route: { path: '/team' },
        popover: {
          title: t('onboarding.team.welcome.title'),
          description: t('onboarding.team.welcome.noTeamDescription'),
          align: 'center',
          nextBtnText: t('onboarding.team.welcome.nextBtn')
        }
      },
      {
        route: { path: '/team' },
        element: '[data-tour="team-create-form"]',
        popover: {
          title: t('onboarding.team.createTeam.title'),
          description: t('onboarding.team.createTeam.description'),
          side: 'top',
          align: 'center'
        }
      },
      {
        route: { path: '/team' },
        element: '[data-tour="sidebar-my-keys"]',
        popover: {
          title: t('onboarding.team.keyPage.title'),
          description: t('onboarding.team.keyPage.noTeamDescription'),
          side: 'right',
          align: 'center'
        }
      },
      {
        route: { path: '/team' },
        element: '[data-tour="sidebar-usage"]',
        popover: {
          title: t('onboarding.team.usagePage.title'),
          description: t('onboarding.team.usagePage.noTeamDescription'),
          side: 'right',
          align: 'center',
          doneBtnText: t('onboarding.team.done')
        }
      }
    ]
  }

  const overviewSteps: RoutedDriveStep[] = isOwner
    ? [
        {
          route: { path: '/team' },
          element: '[data-tour="team-members"]',
          popover: {
            title: t('onboarding.team.members.title'),
            description: t('onboarding.team.members.description'),
            side: 'bottom',
            align: 'center'
          }
        },
        {
          route: { path: '/team' },
          element: '[data-tour="team-invitations"]',
          popover: {
            title: t('onboarding.team.invitations.title'),
            description: t('onboarding.team.invitations.description'),
            side: 'bottom',
            align: 'center'
          }
        },
        {
          route: { path: '/team' },
          element: '[data-tour="team-settings-tab"]',
          popover: {
            title: t('onboarding.team.settings.title'),
            description: t('onboarding.team.settings.description'),
            side: 'bottom',
            align: 'center'
          }
        }
      ]
    : [
        {
          route: { path: '/team' },
          element: '[data-tour="team-limit-progress"]',
          popover: {
            title: t('onboarding.team.limits.title'),
            description: t('onboarding.team.limits.description'),
            side: 'top',
            align: 'center'
          }
        }
      ]

  return [
    {
      route: { path: '/team' },
      popover: {
        title: t('onboarding.team.welcome.title'),
        description: t(isOwner ? 'onboarding.team.welcome.ownerDescription' : 'onboarding.team.welcome.memberDescription'),
        align: 'center',
        nextBtnText: t('onboarding.team.welcome.nextBtn')
      }
    },
    ...overviewSteps,
    {
      route: { path: '/team' },
      element: '[data-tour="sidebar-my-keys"]',
      popover: {
        title: t('onboarding.team.keyPage.title'),
        description: t('onboarding.team.keyPage.description'),
        side: 'right',
        align: 'center'
      }
    },
    {
      route: { path: '/keys', query: { scope: 'team' } },
      element: '[data-tour="keys-scope-switch"]',
      popover: {
        title: t('onboarding.team.keyScope.title'),
        description: t('onboarding.team.keyScope.description'),
        side: 'bottom',
        align: 'end'
      }
    },
    {
      route: { path: '/keys', query: { scope: 'team' } },
      element: '[data-tour="keys-create-btn"]',
      popover: {
        title: t('onboarding.team.createKey.title'),
        description: t('onboarding.team.createKey.description'),
        side: 'bottom',
        align: 'end'
      }
    },
    {
      route: { path: '/keys', query: { scope: 'team' } },
      element: '[data-tour="sidebar-usage"]',
      popover: {
        title: t('onboarding.team.usagePage.title'),
        description: t('onboarding.team.usagePage.description'),
        side: 'right',
        align: 'center'
      }
    },
    {
      route: { path: '/team' },
      element: isOwner ? '[data-tour="team-member-usage-charts"]' : '[data-tour="team-usage-records"]',
      popover: {
        title: t(isOwner ? 'onboarding.team.ownerUsage.title' : 'onboarding.team.memberUsage.title'),
        description: t(isOwner ? 'onboarding.team.ownerUsage.description' : 'onboarding.team.memberUsage.description'),
        side: 'top',
        align: 'center',
        doneBtnText: t('onboarding.team.done')
      }
    }
  ]
}
