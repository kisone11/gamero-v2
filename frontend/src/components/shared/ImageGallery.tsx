import PhotoSwipe from 'photoswipe'
import 'photoswipe/dist/photoswipe.css'

interface Props {
  images: string[]
  thumbnail?: boolean  // show as clickable thumbnails
  maxShow?: number     // max thumbnails to show (default 3)
  className?: string
}

export function ImageGallery({ images, thumbnail = false, maxShow = 3, className = '' }: Props) {
  const open = (index: number) => {
    if (images.length === 0) return
    const items = images.map(url => ({
      src: url,
      w: 1200,
      h: 900,
    }))
    const pswp = new PhotoSwipe({
      dataSource: items,
      index,
      bgOpacity: 0.9,
      showHideAnimationType: 'fade',
    })
    pswp.init()
  }

  if (!thumbnail || images.length === 0) return null

  const showImages = images.slice(0, maxShow)
  const extraCount = images.length - maxShow

  return (
    <div className={`flex gap-1.5 px-1.5 pb-1.5 ${className}`}>
      {showImages.map((url, i) => (
        <button
          key={i}
          onClick={(e) => { e.preventDefault(); e.stopPropagation(); open(i) }}
          className="w-[calc((100%-0.75rem)/3)] aspect-[4/3] max-h-[100px] bg-black/20 rounded-md overflow-hidden cursor-pointer hover:opacity-90 transition-opacity border border-white/[0.04] shrink-0"
        >
          <img src={url} alt="" className="w-full h-full object-cover" loading="lazy" />
        </button>
      ))}
      {extraCount > 0 && (
        <button
          onClick={(e) => { e.preventDefault(); e.stopPropagation(); open(maxShow) }}
          className="w-[calc((100%-0.75rem)/3)] aspect-[4/3] max-h-[100px] bg-black/20 rounded-md flex items-center justify-center text-small text-text-muted font-mono cursor-pointer hover:bg-black/30 transition-colors border border-white/[0.04] shrink-0"
        >
          +{extraCount}
        </button>
      )}
    </div>
  )
}

export function openLightbox(images: string[], index = 0) {
  if (images.length === 0) return
  const items = images.map(url => ({ src: url, w: 1200, h: 900 }))
  const pswp = new PhotoSwipe({
    dataSource: items,
    index,
    bgOpacity: 0.9,
    showHideAnimationType: 'fade',
  })
  pswp.init()
}
