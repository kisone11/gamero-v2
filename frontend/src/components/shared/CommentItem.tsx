import { useState } from 'react'
import { Link } from 'react-router-dom'
import { ThumbsUp, Reply, Edit, Trash2 } from 'lucide-react'
import { Avatar } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { cn } from '@/lib/utils'
import { timeAgo } from '@/lib/time'

/** Normalized comment data — bridges LogComment and PostComment type differences */
export interface CommentItemData {
  id: number
  content: string
  authorName: string
  authorUsername: string
  authorAvatar?: string | null
  likeCount: number
  isLiked: boolean
  isDeleted: boolean
  createdAt: string
  isAuthor: boolean
  parentAuthorName?: string
  replies?: CommentItemData[]
}

interface CommentItemProps {
  comment: CommentItemData
  currentUser: { id: number } | null
  depth?: number
  onLike: (commentId: number, isLiked: boolean) => void
  onDelete: (commentId: number) => void
  onReply: (comment: CommentItemData) => void
  onEdit?: (commentId: number, content: string) => void
}

export function CommentItem({
  comment,
  currentUser,
  depth = 0,
  onLike,
  onDelete,
  onEdit,
  onReply,
}: CommentItemProps) {
  const [editing, setEditing] = useState(false)
  const [editContent, setEditContent] = useState(comment.content)
  const [deleteOpen, setDeleteOpen] = useState(false)

  if (comment.isDeleted) {
    return (
      <div className={cn('py-3', depth > 0 && 'ml-8 pl-4 border-l border-white/[0.04]')}>
        <p className="text-body text-text-muted italic">[该评论已被删除]</p>
      </div>
    )
  }

  const authorUrl = `/u/${comment.authorUsername || comment.authorName}`

  return (
    <div className={cn(depth > 0 && 'ml-8 pl-4 border-l border-white/[0.04]')}>
      <div className="py-3 group">
        <div className="flex items-start gap-3">
          <Link to={authorUrl} className="shrink-0">
            <Avatar src={comment.authorAvatar} name={comment.authorName} size="sm" />
          </Link>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1">
              <Link to={authorUrl} className="text-body font-semibold text-text-primary hover:text-amber transition-colors">
                {comment.authorName}
              </Link>
              {comment.parentAuthorName && (
                <span className="text-small text-text-muted">回复 @{comment.parentAuthorName}</span>
              )}
              <span className="text-small text-text-muted font-mono">{timeAgo(comment.createdAt)}</span>
            </div>

            {editing ? (
              <div className="space-y-2">
                <textarea
                  value={editContent}
                  onChange={(e) => setEditContent(e.target.value)}
                  className="w-full bg-surface-deep border border-white/[0.08] rounded-lg px-3 py-2 text-body text-text-primary placeholder:text-text-muted resize-none focus:outline-none focus:border-amber"
                  rows={3} autoFocus
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
                      onEdit?.(comment.id, editContent); setEditing(false)
                    }
                    if (e.key === 'Escape') { setEditing(false); setEditContent(comment.content) }
                  }}
                />
                <div className="flex gap-2">
                  <Button size="sm" variant="primary" onClick={() => { onEdit?.(comment.id, editContent); setEditing(false) }}>保存</Button>
                  <Button size="sm" variant="ghost" onClick={() => { setEditing(false); setEditContent(comment.content) }}>取消</Button>
                  <span className="text-caption text-text-muted self-center">Ctrl+Enter 提交 · Esc 取消</span>
                </div>
              </div>
            ) : (
              <p className="text-body text-text-secondary leading-relaxed whitespace-pre-wrap break-words">
                {comment.content}
              </p>
            )}

            {!editing && (
              <div className="flex items-center gap-3 mt-2">
                <button
                  onClick={() => onLike(comment.id, comment.isLiked)}
                  className={cn('flex items-center gap-1 text-small transition-colors', comment.isLiked ? 'text-danger' : 'text-text-muted hover:text-text-secondary')}
                >
                  <ThumbsUp className={cn('w-3.5 h-3.5', comment.isLiked && 'fill-current')} />
                  {comment.likeCount > 0 && comment.likeCount}
                </button>
                {currentUser && (
                  <button onClick={() => onReply(comment)} className="flex items-center gap-1 text-small text-text-muted hover:text-amber transition-colors">
                    <Reply className="w-3.5 h-3.5" />回复
                  </button>
                )}
                {comment.isAuthor && (
                  <>
                    {onEdit && (
                      <button onClick={() => { setEditing(true); setEditContent(comment.content) }} className="flex items-center gap-1 text-small text-text-muted hover:text-amber transition-colors">
                        <Edit className="w-3.5 h-3.5" />编辑
                      </button>
                    )}
                    <button onClick={() => setDeleteOpen(true)} className="flex items-center gap-1 text-small text-text-muted hover:text-danger transition-colors">
                      <Trash2 className="w-3.5 h-3.5" />删除
                    </button>
                  </>
                )}
              </div>
            )}
          </div>
        </div>
      </div>

      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent>
          <DialogHeader><DialogTitle>删除评论</DialogTitle><DialogDescription>确定要删除这条评论吗？此操作不可撤销。</DialogDescription></DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setDeleteOpen(false)}>取消</Button>
            <Button variant="danger" onClick={() => { onDelete(comment.id); setDeleteOpen(false) }}>确认删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {comment.replies && comment.replies.length > 0 && (
        <div>
          {comment.replies.map((reply) => (
            <CommentItem
              key={reply.id}
              comment={reply}
              currentUser={currentUser}
              depth={depth + 1}
              onLike={onLike}
              onDelete={onDelete}
              onEdit={onEdit}
              onReply={onReply}
            />
          ))}
        </div>
      )}
    </div>
  )
}
