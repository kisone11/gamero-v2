import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, ShieldAlert, Trash2 } from 'lucide-react'
import { adminApi } from '@/api/admin'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { EmptyState } from '@/components/ui/empty-state'
import { Input } from '@/components/ui/input'
import { toast } from '@/stores/toastStore'

export function SensitiveWordsTab() {
  const queryClient = useQueryClient()
  const [input, setInput] = useState('')

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['admin', 'sensitive-words'],
    queryFn: () => adminApi.getSensitiveWords(),
  })

  const words = data?.words ?? []

  const addMut = useMutation({
    mutationFn: (newWords: string[]) => adminApi.addSensitiveWords(newWords),
    onSuccess: () => {
      setInput('')
      queryClient.invalidateQueries({ queryKey: ['admin', 'sensitive-words'] })
      toast.success('敏感词已添加')
    },
    onError: () => toast.error('添加失败'),
  })

  const deleteMut = useMutation({
    mutationFn: (word: string) => adminApi.deleteSensitiveWord(word),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'sensitive-words'] })
      toast.success('敏感词已删除')
    },
    onError: () => toast.error('删除失败'),
  })

  const handleAdd = () => {
    const newWords = input.split(',').map((word) => word.trim()).filter(Boolean)
    if (newWords.length === 0) {
      toast.error('请输入敏感词，多个词用英文逗号分隔')
      return
    }
    addMut.mutate(newWords)
  }

  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Enter') handleAdd()
  }

  return (
    <div>
      <div className="mb-4 flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div>
          <h3 className="text-h3 text-text-primary">敏感词管理</h3>
          <p className="mt-1 text-caption text-text-muted">管理社区内容过滤词库，新增后会写入后端持久化敏感词表。</p>
        </div>
        <div className="flex gap-2 md:w-[420px]">
          <Input
            placeholder="输入敏感词，多个用英文逗号分隔"
            value={input}
            onChange={(event) => setInput(event.target.value)}
            onKeyDown={handleKeyDown}
            fullWidth
          />
          <Button size="sm" onClick={handleAdd} loading={addMut.isPending}>添加</Button>
        </div>
      </div>

      {isLoading && (
        <div className="grid gap-2 md:grid-cols-3">
          {Array.from({ length: 9 }).map((_, index) => (
            <div key={index} className="h-12 animate-pulse rounded-xl bg-white/[0.03]" />
          ))}
        </div>
      )}

      {isError && !isLoading && (
        <EmptyState
          icon={<AlertTriangle className="h-7 w-7 text-text-muted" />}
          title="加载失败"
          description="无法加载敏感词列表，请重试"
          action={{ label: '重新加载', onClick: () => refetch() }}
        />
      )}

      {!isLoading && !isError && words.length === 0 && (
        <EmptyState
          icon={<ShieldAlert className="h-7 w-7 text-text-muted" />}
          title="暂无敏感词"
          description="添加后会用于内容发布过滤"
        />
      )}

      {!isLoading && !isError && words.length > 0 && (
        <div className="grid gap-2 md:grid-cols-3">
          {words.map((word) => (
            <Card key={word} padding="sm" hover={false}>
              <div className="flex items-center justify-between gap-3">
                <span className="truncate text-small font-medium text-text-primary">{word}</span>
                <Button
                  variant="ghost"
                  size="sm"
                  loading={deleteMut.isPending}
                  onClick={() => deleteMut.mutate(word)}
                  aria-label={`删除敏感词 ${word}`}
                >
                  <Trash2 className="h-3.5 w-3.5 text-danger" />
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
