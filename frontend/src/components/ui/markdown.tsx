import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { cn } from '@/lib/utils'

interface Props {
  content: string
  className?: string
}

export function Markdown({ content, className }: Props) {
  if (!content) return null
  return (
    <div className={cn('prose prose-invert max-w-none', className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          h1: ({ children, ...props }) => <h1 className="text-[20px] font-bold text-text-primary mt-6 mb-3" {...props}>{children}</h1>,
          h2: ({ children, ...props }) => <h2 className="text-[17px] font-bold text-text-primary mt-5 mb-2" {...props}>{children}</h2>,
          h3: ({ children, ...props }) => <h3 className="text-[15px] font-semibold text-text-primary mt-4 mb-2" {...props}>{children}</h3>,
          p: ({ children, ...props }) => <p className="text-[14px] text-text-secondary leading-relaxed mb-3" {...props}>{children}</p>,
          ul: ({ children, ...props }) => <ul className="list-disc list-inside text-[14px] text-text-secondary mb-3 space-y-1" {...props}>{children}</ul>,
          ol: ({ children, ...props }) => <ol className="list-decimal list-inside text-[14px] text-text-secondary mb-3 space-y-1" {...props}>{children}</ol>,
          li: ({ children, ...props }) => <li className="text-[14px] text-text-secondary" {...props}>{children}</li>,
          a: ({ children, href, ...props }) => (
            <a href={href} target="_blank" rel="noopener noreferrer" className="text-amber hover:underline" {...props}>{children}</a>
          ),
          code: ({ children, className: codeClass, ...props }: any) => {
            const isInline = !codeClass
            if (isInline) {
              return <code className="px-1.5 py-0.5 bg-surface-void text-amber rounded text-[13px] font-mono" {...props}>{children}</code>
            }
            return (
              <pre className="bg-surface-void border border-white/[0.04] rounded-lg p-4 overflow-x-auto my-3">
                <code className="text-[13px] text-text-secondary font-mono" {...props}>{children}</code>
              </pre>
            )
          },
          blockquote: ({ children, ...props }) => (
            <blockquote className="border-l-2 border-amber/30 pl-4 my-3 text-[14px] text-text-muted italic" {...props}>{children}</blockquote>
          ),
          img: ({ src, alt, ...props }) => (
            <img src={src} alt={alt} className="rounded-lg max-w-full my-3 border border-white/[0.04]" {...props} />
          ),
          table: ({ children, ...props }) => (
            <div className="overflow-x-auto my-3">
              <table className="w-full text-[13px] border-collapse" {...props}>{children}</table>
            </div>
          ),
          th: ({ children, ...props }) => <th className="border border-white/[0.06] px-3 py-2 text-left font-semibold text-text-primary bg-surface-void" {...props}>{children}</th>,
          td: ({ children, ...props }) => <td className="border border-white/[0.06] px-3 py-2 text-text-secondary" {...props}>{children}</td>,
          hr: (props) => <hr className="border-white/[0.04] my-4" {...props} />,
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  )
}
