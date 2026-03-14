import { Component, type ReactNode } from 'react'
import { Result, Button } from 'antd'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
}

export default class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, error: null }

  static getDerivedStateFromError(error: Error) {
    return { hasError: true, error }
  }

  render() {
    if (this.state.hasError) {
      return (
        <Result
          status="error"
          title="页面出错了"
          subTitle={this.state.error?.message || '发生了未知错误'}
          extra={
            <Button
              type="primary"
              onClick={() => this.setState({ hasError: false, error: null })}
            >
              重试
            </Button>
          }
        />
      )
    }
    return this.props.children
  }
}
