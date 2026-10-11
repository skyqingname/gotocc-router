// Derived from TokenFlux/TokenRouter f383062dd5fd534f462887acb70ea241ce0d8278. Copyright (c) 2026 Wesley Liddick & TokenFlux. LGPL-3.0-or-later. GoToCC adaptation.
export default {
  "studioGotocc": {
    "history": "Creation history",
    "description": "Generate, edit and inpaint images on your canvas",
    "active": "{count} generations in progress",
    "processing": "Generating in the background",
    "selectKey": "Select an API key to use its group and balance.",
    "readOnlyKey": "This key can currently read history only. Select an active image key to generate.",
    "storageHint": "Results are stored on this site. The canvas and uploads are stored in this user’s browser.",
    "resetHint": "Clear the current canvas and unused local image cache. Download a backup of uploaded images first. Server results and task history will remain.",
    "toCanvas": "Add to canvas",
    "reuse": "Use prompt and settings",
    "partial": "Generated {actual} of {requested} requested images.",
    "quality": {
      "auto": "Auto",
      "low": "Low",
      "medium": "Medium",
      "high": "High"
    },
    "background": {
      "auto": "Auto",
      "opaque": "Opaque",
      "transparent": "Transparent"
    },
    "chooseKeyInSettings": "Select an API key in the settings at the top left.",
    "actualUsagePricing": "Billed by usage",
    "settings": "Studio settings",
    "customSize": "Custom pixel size",
    "backHome": "Back to home"
  },
  "creative": {
    "title": "Creative Studio",
    "description": "Generate and edit on a canvas. Results remain stored on this site.",
    "panel": {
      "model": "Model",
      "selectImageHint": "Select an image on the canvas before generating.",
      "maskHint": "Select an image on the canvas first, then paint the area to redraw with the brush.",
      "imageSize": "Image size",
      "aspectRatio": "Aspect ratio",
      "quality": "Quality",
      "background": "Background",
      "thinkingLevel": "Thinking level",
      "estimatedCost": "{cost} / image",
      "uploadSource": "Upload image",
      "promptPlaceholder": "Describe the image you want to create...",
      "studioDisabled": "Creative Studio is disabled. Please contact your administrator to enable it.",
      "noModelsAvailable": "No image generation models are available. Please ask your administrator to configure a group with image generation enabled."
    },
    "composer": {
      "send": "Send",
      "model": "Model",
      "params": "Params",
      "operation": "Operation",
      "selectModel": "Select a model",
      "selectModelFirst": "Select a model first.",
      "autoRatio": "Auto ratio"
    },
    "empty": {
      "title": "Start with a description",
      "description": "Type a prompt below to generate an image, or upload one to edit or inpaint it.",
      "upload": "Upload image",
      "examples": {
        "poster": {
          "label": "Retro travel poster",
          "prompt": "A 1970s retro travel poster of a seaside town at sunset, warm orange tones, grainy paper texture, large sans-serif title"
        },
        "product": {
          "label": "Product shot",
          "prompt": "Studio product shot of a frosted glass perfume bottle on a light gray background, soft side lighting, subtle reflections, simple composition"
        },
        "illustration": {
          "label": "Flat illustration",
          "prompt": "An orange cat reading a book by the window, flat illustration style, muted colors, indoor plants, afternoon sunlight"
        }
      }
    },
    "operations": {
      "generate": "Generate",
      "edit": "Edit",
      "inpaint": "Inpaint"
    },
    "cropper": {
      "title": "Crop image",
      "hint": "Drag to adjust the crop area. The cropped image is stored only in this browser.",
      "confirm": "Apply crop",
      "skip": "Skip cropping"
    },
    "canvas": {
      "brushSize": "Brush size",
      "shapeRound": "Round stroke",
      "shapeSquare": "Square stroke",
      "boxSelect": "Select by dragging a box",
      "paintToggleOn": "Start painting",
      "paintToggleOff": "Pause painting to pan",
      "undoMask": "Undo last stroke",
      "clearMask": "Clear all paint",
      "downloadSelected": "Download selected image",
      "removeSelected": "Remove selected image",
      "reset": "Clear canvas",
      "backToDashboard": "Back to dashboard",
      "settings": "Settings",
      "inpaintPickHint": "Click an image on the canvas, then paint the area to redraw",
      "editPickHint": "Tap an image as reference, or drag a box to select multiple images",
      "maskPaintHint": "Paint over the area to redraw. The purple strokes are sent as the mask."
    },
    "error": {
      "loadModelsFailed": "Failed to load creative models. Please try again later.",
      "noModel": "Please select a model first.",
      "operationNotSupported": "The selected model does not support this operation.",
      "promptTooLong": "Prompt exceeds the server limit.",
      "assetTooLarge": "The image exceeds the server size or format limit.",
      "sourceRequired": "Edit and inpaint operations require at least one source image.",
      "maskRequired": "Inpaint requires a mask. Select an image and paint the area with the brush first.",
      "referenceLimit": "This model supports at most {max} reference images.",
      "submitFailed": "Failed to submit the creative run. Please retry.",
      "historyFailed": "Failed to load run history. Check this browser storage permission and retry.",
      "workspaceUnavailable": "This browser workspace is unavailable. Allow local storage for this site and retry.",
      "clearFailed": "Failed to clear local data.",
      "loadImageFailed": "Failed to load the image.",
      "quotaExceeded": "Local storage is full. Please download your assets before continuing.",
      "saveFailed": "The canvas could not be saved locally. Download your assets before trying again later.",
      "dropInvalid": "The dropped history image could not be recognized.",
      "dropUnsupported": "Only PNG, JPEG, or WebP images are supported.",
      "dropHistoryUnavailable": "This history image is no longer available in this browser."
    },
    "operationsDesc": {
      "generate": "Generate an image from the prompt alone",
      "edit": "Edit using the selected canvas image as reference",
      "inpaint": "Paint over the selected image to redraw that area"
    },
    "qualities": {
      "low": "Low",
      "medium": "Medium",
      "high": "High",
      "auto": "Auto"
    },
    "backgrounds": {
      "auto": "Auto",
      "opaque": "Opaque",
      "transparent": "Transparent"
    },
    "thinkingLevels": {
      "minimal": "Minimal",
      "high": "High"
    }
  }
}
