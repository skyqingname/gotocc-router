// Creative Studio subset of TokenFlux/TokenRouter f383062d; LGPL-3.0-or-later.
import UploadIcon from './artwork/upload'
import DownloadIcon from './artwork/download'
import SquareDashedIcon from './artwork/square-dashed'
import SquarePenIcon from './artwork/square-pen'
import EraserIcon from './artwork/eraser'
import Undo2Icon from './artwork/undo-2'
import CircleIcon from './artwork/circle'
import SquareIcon from './artwork/square'
import TrashIcon from './artwork/trash'
import SettingsIcon from './artwork/settings'
import HistoryIcon from './artwork/history'
import SparklesIcon from './artwork/sparkles'
import RefreshCwIcon from './artwork/refresh-cw'
import XIcon from './artwork/x'
import PlusIcon from './artwork/plus'
import HomeIcon from './artwork/home'
import SlidersHorizontalIcon from './artwork/sliders-horizontal'
import ImageIcon from './artwork/image'
import BrushIcon from './artwork/brush'
import LoaderCircleIcon from './artwork/loader-circle'
import ChevronDownIcon from './artwork/chevron-down'
import CheckIcon from './artwork/check'
import ArrowUpIcon from './artwork/arrow-up'
import CircleAlertIcon from './artwork/circle-alert'
import InfoIcon from './artwork/info'

export const icons = {
  upload: UploadIcon,
  download: DownloadIcon,
  selection: SquareDashedIcon,
  edit: SquarePenIcon,
  eraser: EraserIcon,
  undo: Undo2Icon,
  circle: CircleIcon,
  square: SquareIcon,
  trash: TrashIcon,
  cog: SettingsIcon,
  history: HistoryIcon,
  sparkles: SparklesIcon,
  refresh: RefreshCwIcon,
  x: XIcon,
  plus: PlusIcon,
  home: HomeIcon,
  sliders: SlidersHorizontalIcon,
  modalityImage: ImageIcon,
  brush: BrushIcon,
  loader: LoaderCircleIcon,
  chevronDown: ChevronDownIcon,
  check: CheckIcon,
  arrowUp: ArrowUpIcon,
  exclamationCircle: CircleAlertIcon,
  infoCircle: InfoIcon,
}
export type IconName = keyof typeof icons
