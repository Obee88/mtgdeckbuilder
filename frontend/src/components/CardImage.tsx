import { useState, useEffect, useCallback } from 'react'
import { createPortal } from 'react-dom'

interface Props {
  src: string
  alt: string
  className?: string
}

export default function CardImage({ src, alt, className }: Props) {
  const [preview, setPreview] = useState<{ x: number; y: number } | null>(null)

  const show = useCallback((e: React.MouseEvent) => {
    if (e.button !== 2) return
    e.preventDefault()
    setPreview({ x: e.clientX, y: e.clientY })
  }, [])

  const hide = useCallback(() => setPreview(null), [])

  // Hide on any mouseup anywhere (user released right-click)
  useEffect(() => {
    if (!preview) return
    window.addEventListener('mouseup', hide)
    return () => window.removeEventListener('mouseup', hide)
  }, [preview, hide])

  // Compute position: keep preview inside viewport
  function getStyle() {
    if (!preview) return {}
    const W = 265 // card image width
    const H = 370 // card image height approx
    const pad = 12
    let left = preview.x + 16
    let top = preview.y - H / 2
    if (left + W > window.innerWidth - pad) left = preview.x - W - 16
    if (top < pad) top = pad
    if (top + H > window.innerHeight - pad) top = window.innerHeight - H - pad
    return { left, top, width: W }
  }

  return (
    <>
      <img
        src={src}
        alt={alt}
        className={className}
        onMouseDown={show}
        onContextMenu={(e) => e.preventDefault()}
        draggable={false}
      />
      {preview &&
        createPortal(
          <img
            src={src}
            alt={alt}
            className="fixed z-50 rounded-xl shadow-2xl border border-gray-600 pointer-events-none select-none"
            style={getStyle()}
            draggable={false}
          />,
          document.body
        )}
    </>
  )
}
