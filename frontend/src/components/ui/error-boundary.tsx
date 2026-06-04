import { Component } from 'react'
import { Link } from 'react-router-dom'
import { AlertTriangle, RefreshCw } from 'lucide-react'
import { Button } from './button'

interface Props {
  children: React.ReactNode
  fallback?: React.ReactNode
}

interface State {
  hasError: boolean
  error?: Error
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback
      return (
        <div className="min-h-screen bg-surface-void flex items-center justify-center p-6">
          <div className="text-center max-w-md">
            <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-danger/10 flex items-center justify-center">
              <AlertTriangle className="w-8 h-8 text-danger" />
            </div>
            <h1 className="text-[18px] font-bold text-text-primary mb-2">页面出了点问题</h1>
            <p className="text-[13px] text-text-muted mb-6">
              {this.state.error?.message || '发生了意外错误'}
            </p>
            <div className="flex gap-3 justify-center">
              <Button variant="secondary" size="sm" onClick={() => window.location.reload()}>
                <RefreshCw className="w-3.5 h-3.5" /> 刷新页面
              </Button>
              <Link to="/feed">
                <Button variant="ghost" size="sm">返回首页</Button>
              </Link>
            </div>
          </div>
        </div>
      )
    }
    return this.props.children
  }
}
