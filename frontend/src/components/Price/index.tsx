interface PriceProps {
  value: number       // cents
  original?: number   // cents, optional strikethrough
  size?: 'sm' | 'md' | 'lg'
}

const sizes = { sm: 14, md: 18, lg: 24 }

export default function Price({ value, original, size = 'md' }: PriceProps) {
  const fontSize = sizes[size]
  return (
    <span>
      <span style={{ color: 'var(--color-accent)', fontWeight: 700, fontSize }}>
        ¥{(value / 100).toFixed(2)}
      </span>
      {original && original > value && (
        <span style={{
          color: 'var(--color-text-secondary)',
          textDecoration: 'line-through',
          fontSize: fontSize - 4,
          marginLeft: 4,
        }}>
          ¥{(original / 100).toFixed(2)}
        </span>
      )}
    </span>
  )
}
