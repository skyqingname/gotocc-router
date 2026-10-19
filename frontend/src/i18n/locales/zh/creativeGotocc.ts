// Derived from TokenFlux/TokenRouter f383062dd5fd534f462887acb70ea241ce0d8278. Copyright (c) 2026 Wesley Liddick & TokenFlux. LGPL-3.0-or-later. GoToCC adaptation.
export default {
  "studioGotocc": {
    "history": "创作记录",
    "description": "在画布中生成、编辑与重绘图片",
    "active": "{count} 个任务正在生成",
    "processing": "后台生成中，可以继续创作",
    "selectKey": "先选择 API Key，使用其对应分组与额度。",
    "readOnlyKey": "此 Key 当前仅可查看历史；生成需要可用的生图 Key。",
    "storageHint": "作品保存在本站；画布和上传素材保存在当前用户的本机浏览器。",
    "resetHint": "清空当前画布和不再使用的本机图片缓存。上传素材请先下载备份；服务端作品与任务记录仍然保留。",
    "toCanvas": "放入画布",
    "reuse": "填入输入框",
    "partial": "请求 {requested} 张，实际生成 {actual} 张。",
    "quality": {
      "auto": "自动",
      "low": "低",
      "medium": "中",
      "high": "高"
    },
    "background": {
      "auto": "自动",
      "opaque": "不透明",
      "transparent": "透明"
    },
    "chooseKeyInSettings": "请先在左上角设置中选择 API Key。",
    "estimatedCostTotal": "{cost} / 张 · 共 {total}",
    "actualUsagePricing": "按实际用量",
    "settings": "创作台设置",
    "customSize": "自定义像素尺寸",
    "backHome": "返回首页"
  },
  "creative": {
    "title": "创作台",
    "description": "在画布中生成与编辑图片，作品继续保存在本站。",
    "panel": {
      "model": "模型",
      "selectImageHint": "请先在画布中选中一张图片，再开始生成。",
      "maskHint": "请先在画布中选中一张图片，然后用画笔涂抹要重绘的区域。",
      "imageSize": "图片尺寸",
      "aspectRatio": "画面比例",
      "quality": "画质",
      "background": "背景",
      "thinkingLevel": "思考强度",
      "estimatedCost": "{cost} / 张",
      "uploadSource": "上传图片",
      "promptPlaceholder": "描述想要生成的画面...",
      "studioDisabled": "创作台功能已关闭，请联系管理员开启。",
      "noModelsAvailable": "暂无可用的图片生成模型，请联系管理员配置支持图片生成的分组。"
    },
    "composer": {
      "send": "发送",
      "model": "模型",
      "params": "参数",
      "operation": "操作",
      "selectModel": "选择模型",
      "selectModelFirst": "请先选择模型。",
      "autoRatio": "自动比例"
    },
    "empty": {
      "title": "从一句描述开始",
      "description": "在下方输入提示词生成图片，或上传图片后做图生图和局部重绘。",
      "upload": "上传图片",
      "examples": {
        "poster": {
          "label": "复古旅行海报",
          "prompt": "一张 1970 年代风格的复古旅行海报，海边小镇，暖橙色夕阳，颗粒纸张质感，大号无衬线标题字"
        },
        "product": {
          "label": "产品棚拍图",
          "prompt": "磨砂玻璃香水瓶的产品棚拍图，浅灰背景，柔和侧光，瓶身有细腻反光，构图简洁"
        },
        "illustration": {
          "label": "扁平插画",
          "prompt": "一只在窗边看书的橘猫，扁平插画风格，低饱和配色，室内绿植，午后阳光"
        }
      }
    },
    "operations": {
      "generate": "文生图",
      "edit": "图生图",
      "inpaint": "局部重绘"
    },
    "cropper": {
      "title": "裁剪图片",
      "hint": "拖动调整裁剪区域，裁剪结果仅保存在当前浏览器。",
      "confirm": "确认裁剪",
      "skip": "跳过裁剪"
    },
    "canvas": {
      "brushSize": "画笔粗细",
      "shapeRound": "圆头笔迹",
      "shapeSquare": "方头笔迹",
      "boxSelect": "框选画布对象",
      "paintToggleOn": "开始涂抹",
      "paintToggleOff": "暂停涂抹，移动视角",
      "undoMask": "撤销上一笔",
      "clearMask": "清除全部涂抹",
      "downloadSelected": "下载选中图片",
      "removeSelected": "移除选中图片",
      "reset": "清空画布",
      "backToDashboard": "返回仪表盘",
      "settings": "设置",
      "inpaintPickHint": "点击画布中的一张图片，在其上涂抹要重绘的区域",
      "editPickHint": "点击图片选择参考图，或直接拖拽框选多张",
      "maskPaintHint": "在图片上涂抹要重绘的区域，紫色部分会作为 mask 提交"
    },
    "error": {
      "loadModelsFailed": "模型目录加载失败，请稍后重试。",
      "noModel": "请先选择模型。",
      "operationNotSupported": "所选模型不支持该操作。",
      "promptTooLong": "提示词超过服务端允许的长度。",
      "assetTooLarge": "图片超过服务端允许的大小或格式限制。",
      "sourceRequired": "图生图 / 局部重绘至少需要一张源图。",
      "maskRequired": "局部重绘需要 mask，请先选中图片并用画笔绘制要重绘的区域。",
      "referenceLimit": "当前模型最多支持 {max} 张参考图。",
      "submitFailed": "提交生成失败，请重试。",
      "historyFailed": "历史记录加载失败，请检查当前浏览器存储权限后重试。",
      "workspaceUnavailable": "当前浏览器工作区不可用，请允许本站使用本地存储后重试。",
      "clearFailed": "清空本机数据失败。",
      "loadImageFailed": "图片加载失败。",
      "quotaExceeded": "本地存储空间不足，请先下载备份素材。",
      "saveFailed": "画布未能保存到本机，请先下载备份素材，稍后重试。",
      "dropInvalid": "无法识别拖入的历史图片。",
      "dropUnsupported": "仅支持 PNG、JPEG 或 WebP 图片。",
      "dropHistoryUnavailable": "历史图片已不在本机，无法拖入画布。"
    },
    "operationsDesc": {
      "generate": "直接根据提示词生成图片",
      "edit": "以画布中选中的图片为参考进行编辑",
      "inpaint": "涂抹选中图片的区域并重新绘制"
    },
    "qualities": {
      "low": "低",
      "medium": "中",
      "high": "高",
      "auto": "自动"
    },
    "backgrounds": {
      "auto": "自动",
      "opaque": "不透明",
      "transparent": "透明"
    },
    "thinkingLevels": {
      "minimal": "最少",
      "high": "高"
    }
  }
}
