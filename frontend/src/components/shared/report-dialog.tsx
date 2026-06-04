import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Flag, MessageSquare, EyeOff, AlertTriangle, Ban, HelpCircle, CheckCircle } from 'lucide-react'
import { communityApi } from '@/api/community'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { toast } from '@/stores/toastStore'

const REASONS = [
  { value: 'spam', label: '垃圾广告', icon: MessageSquare },
  { value: 'porn', label: '色情内容', icon: EyeOff },
  { value: 'abuse', label: '辱骂攻击', icon: AlertTriangle },
  { value: 'violation', label: '违规内容', icon: Ban },
  { value: 'other', label: '其他', icon: HelpCircle },
]

const MAX_SUPPLEMENT = 200

interface Props {
  targetType: string
  targetId: number
  trigger?: React.ReactNode
}

export function ReportDialog({ targetType, targetId, trigger }: Props) {
  const [open, setOpen] = useState(false)
  const [reason, setReason] = useState('')
  const [supplement, setSupplement] = useState('')
  const [submitted, setSubmitted] = useState(false)

  const reportMut = useMutation({
    mutationFn: () => communityApi.createReport({ target_type: targetType as any, target_id: targetId, reason, supplement: supplement.trim() || undefined }),
    onSuccess: () => {
      setSubmitted(true)
      setTimeout(() => {
        setOpen(false)
        setSubmitted(false)
        setReason('')
        setSupplement('')
      }, 2000)
    },
    onError: (err: any) => toast.error(err?.response?.data?.message || '举报失败'),
  })

  const handleClose = () => {
    if (reportMut.isPending) return
    setOpen(false)
    setSubmitted(false)
    setReason('')
    setSupplement('')
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      {trigger || (
        <Button variant="ghost" size="sm" onClick={() => setOpen(true)}>
          <Flag className="w-3.5 h-3.5" /> 举报
        </Button>
      )}
      <DialogContent>
        {submitted ? (
          <div className="flex flex-col items-center justify-center py-8 space-y-4">
            <div className="w-14 h-14 rounded-full bg-green-500/10 flex items-center justify-center">
              <CheckCircle className="w-7 h-7 text-green-500" />
            </div>
            <div className="text-center">
              <p className="text-h3 text-text-primary mb-1">举报已提交</p>
              <p className="text-body text-text-muted">我们会尽快处理</p>
            </div>
          </div>
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>举报内容</DialogTitle>
              <DialogDescription>请选择举报原因，我们会尽快处理</DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-2">
                {REASONS.map(r => {
                  const Icon = r.icon
                  const isSelected = reason === r.value
                  return (
                    <button
                      key={r.value}
                      onClick={() => setReason(r.value)}
                      className={`flex items-center gap-2.5 px-3 py-2.5 rounded-lg border text-body transition-colors ${
                        isSelected
                          ? 'bg-amber/10 text-amber border-amber/30'
                          : 'bg-surface-void text-text-secondary border-white/[0.04] hover:border-white/[0.10] hover:text-text-primary'
                      }`}
                    >
                      <Icon className={`w-4 h-4 shrink-0 ${isSelected ? 'text-amber' : 'text-text-muted'}`} />
                      <span className="font-medium">{r.label}</span>
                    </button>
                  )
                })}
              </div>
              <div className="relative">
                <textarea
                  placeholder="补充说明（选填）"
                  value={supplement}
                  onChange={e => {
                    if (e.target.value.length <= MAX_SUPPLEMENT) {
                      setSupplement(e.target.value)
                    }
                  }}
                  rows={2}
                  className="w-full px-3 py-2 bg-surface-void border border-white/[0.06] rounded-lg text-body text-text-primary placeholder:text-text-muted resize-none focus:outline-none focus:border-amber/40"
                />
                <span className={`absolute bottom-2 right-2 text-caption ${supplement.length >= MAX_SUPPLEMENT ? 'text-danger' : 'text-text-muted'}`}>
                  {supplement.length}/{MAX_SUPPLEMENT}
                </span>
              </div>
            </div>
            <DialogFooter>
              <Button variant="ghost" size="sm" onClick={handleClose}>取消</Button>
              <Button
                size="sm"
                loading={reportMut.isPending}
                disabled={!reason}
                onClick={() => reportMut.mutate()}
              >
                提交举报
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}
