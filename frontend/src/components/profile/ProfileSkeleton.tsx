import { Skeleton } from '@/components/ui/skeleton'

export function ProfileSkeleton() {
  return (
    <div className="max-w-[960px] mx-auto px-6 py-8 space-y-6">
      <div className="bg-surface-card border border-white/[0.04] rounded-xl p-6">
        <div className="flex flex-col sm:flex-row gap-5">
          <div className="flex justify-center sm:justify-start">
            <Skeleton className="w-20 h-20 rounded-lg" />
          </div>
          <div className="flex-1 space-y-3">
            <Skeleton className="h-6 w-48 mx-auto sm:mx-0" />
            <Skeleton className="h-4 w-28 mx-auto sm:mx-0" />
            <Skeleton className="h-4 w-64 mx-auto sm:mx-0" />
            <div className="flex justify-center sm:justify-start gap-2">
              <Skeleton className="h-5 w-14" />
              <Skeleton className="h-5 w-14" />
            </div>
            <div className="flex justify-center sm:justify-start gap-6 pt-4 border-t border-white/[0.04]">
              <Skeleton className="h-12 w-16" />
              <Skeleton className="h-12 w-16" />
              <Skeleton className="h-12 w-16" />
            </div>
          </div>
        </div>
      </div>
      <div className="flex gap-6 border-b border-white/[0.04]">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-8 w-16" />
        ))}
      </div>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="space-y-6">
          <Skeleton className="h-40 w-full rounded-xl" />
          <Skeleton className="h-32 w-full rounded-xl" />
        </div>
        <div className="space-y-6">
          <Skeleton className="h-36 w-full rounded-xl" />
          <Skeleton className="h-36 w-full rounded-xl" />
        </div>
      </div>
    </div>
  )
}
