import { Card } from '@/components/ui/card'
import { Skeleton, SkeletonText } from '@/components/ui/skeleton'

export function PostSkeleton() {
  return (
    <div className="max-w-[1060px] mx-auto px-6 py-8">
      <div className="flex gap-8">
        <main className="flex-1 min-w-0 max-w-[720px]">
          <Card padding="lg" className="space-y-4">
            <Skeleton className="h-4 w-20" />
            <Skeleton className="h-6 w-24 rounded-full" />
            <Skeleton className="h-8 w-full" />
            <div className="flex items-center gap-3">
              <Skeleton className="w-10 h-10 rounded-lg" />
              <div className="space-y-1.5">
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-3 w-16" />
              </div>
            </div>
            <SkeletonText lines={6} />
            <div className="flex gap-2">
              <Skeleton className="h-9 w-20 rounded-lg" />
              <Skeleton className="h-9 w-20 rounded-lg" />
              <Skeleton className="h-9 w-20 rounded-lg" />
            </div>
          </Card>
          <Card padding="lg" className="mt-6 space-y-4">
            <Skeleton className="h-5 w-16" />
            <Skeleton className="h-20 w-full" />
          </Card>
        </main>
        <aside className="hidden lg:block w-[280px] shrink-0 space-y-4">
          <Skeleton className="h-64 rounded-xl" />
          <Skeleton className="h-48 rounded-xl" />
        </aside>
      </div>
    </div>
  )
}
