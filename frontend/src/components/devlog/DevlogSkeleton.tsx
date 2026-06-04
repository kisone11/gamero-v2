import { Card } from '@/components/ui/card'
import { Skeleton, SkeletonText } from '@/components/ui/skeleton'

export function DevlogSkeleton() {
  return (
    <div className="max-w-[1060px] mx-auto px-6 py-8">
      <div className="flex gap-8">
        <main className="flex-1 min-w-0 max-w-[720px]">
          <Card padding="lg" className="space-y-4">
            <div className="flex items-center justify-between">
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-8 w-20 rounded-lg" />
            </div>
            <div className="flex gap-2">
              <Skeleton className="h-5 w-16 rounded-full" />
              <Skeleton className="h-5 w-12 rounded-full" />
            </div>
            <Skeleton className="h-8 w-full" />
            <div className="flex items-center gap-3">
              <Skeleton className="w-10 h-10 rounded-lg" />
              <div className="space-y-1.5">
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-3 w-16" />
              </div>
            </div>
            <Skeleton className="h-56 w-full rounded-xl" />
            <SkeletonText lines={8} />
            <div className="flex gap-2">
              <Skeleton className="h-9 w-20 rounded-lg" />
              <Skeleton className="h-9 w-20 rounded-lg" />
              <Skeleton className="h-9 w-20 rounded-lg" />
            </div>
          </Card>
        </main>
        <aside className="hidden lg:block w-[280px] shrink-0 space-y-4">
          <Skeleton className="h-24 rounded-xl" />
          <Skeleton className="h-64 rounded-xl" />
          <Skeleton className="h-48 rounded-xl" />
        </aside>
      </div>
    </div>
  )
}
