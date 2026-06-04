import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'

export function ReportSkeleton() {
  return (
    <Card padding="md">
      <div className="space-y-3">
        <div className="flex items-center gap-3">
          <Skeleton className="h-8 w-8 rounded-lg" />
          <div className="space-y-1.5">
            <Skeleton className="h-3.5 w-28" />
            <Skeleton className="h-2.5 w-20" />
          </div>
        </div>
        <Skeleton className="h-3 w-full" />
        <div className="flex gap-2 pt-1">
          <Skeleton className="h-6 w-14 rounded-md" />
          <Skeleton className="h-6 w-14 rounded-md" />
          <Skeleton className="h-6 w-20 rounded-md" />
        </div>
      </div>
    </Card>
  )
}

export function UserSkeleton() {
  return (
    <Card padding="md">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Skeleton className="h-10 w-10 rounded-lg" />
          <div className="space-y-1.5">
            <Skeleton className="h-3.5 w-28" />
            <Skeleton className="h-2.5 w-16" />
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Skeleton className="h-7 w-14 rounded-md" />
          <Skeleton className="h-7 w-14 rounded-md" />
        </div>
      </div>
    </Card>
  )
}

export function StatSkeleton() {
  return (
    <Card padding="md">
      <div className="flex items-center gap-3">
        <Skeleton className="h-10 w-10 rounded-lg" />
        <div className="space-y-1.5">
          <Skeleton className="h-3 w-16" />
          <Skeleton className="h-5 w-20" />
        </div>
      </div>
    </Card>
  )
}

export function TopicSkeleton() {
  return (
    <Card padding="md">
      <div className="flex items-center gap-3">
        <Skeleton className="h-9 w-9 rounded-lg" />
        <div className="flex-1 space-y-1.5">
          <Skeleton className="h-4 w-32" />
          <Skeleton className="h-3 w-48" />
        </div>
        <Skeleton className="h-7 w-14 rounded-md" />
        <Skeleton className="h-7 w-14 rounded-md" />
      </div>
    </Card>
  )
}

export function LogSkeleton() {
  return (
    <Card padding="md">
      <div className="flex items-center gap-3">
        <Skeleton className="h-9 w-9 rounded-lg" />
        <div className="flex-1 space-y-1.5">
          <Skeleton className="h-4 w-48" />
          <Skeleton className="h-3 w-32" />
        </div>
        <div className="flex gap-2">
          <Skeleton className="h-7 w-14 rounded-md" />
          <Skeleton className="h-7 w-14 rounded-md" />
        </div>
      </div>
    </Card>
  )
}
