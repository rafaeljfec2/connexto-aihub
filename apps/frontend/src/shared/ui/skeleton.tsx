import { cn } from './cn'

type SkeletonProps = Readonly<{
  className?: string
  motionSafe?: boolean
}>

export function Skeleton({ className, motionSafe = false }: SkeletonProps) {
  return (
    <div
      className={cn(
        motionSafe ? 'motion-safe:animate-pulse' : 'animate-pulse',
        'rounded-md bg-raised/80',
        className,
      )}
      aria-hidden
    />
  )
}
