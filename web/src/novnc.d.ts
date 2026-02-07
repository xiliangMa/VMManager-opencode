/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_URL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare module '@novnc/novnc' {
  interface RFBOptions {
    credentials?: {
      username?: string
      password?: string
      privateKey?: string
      certificate?: string
    }
    retry?: boolean
    reconnectDelay?: number
    reconnectTimeout?: number
    shared?: boolean
    viewOnly?: boolean
    localCursor?: boolean
    styles?: string
    repeaterID?: string
    logger?: object
  }

  class RFB {
    constructor(
      container: HTMLElement,
      url: string,
      options?: RFBOptions
    )

    readonly screen: HTMLElement
    readonly canvas: HTMLCanvasElement
    readonly controlState: string
    readonly display: object
    readonly keyboard: object
    readonly pointer: object

    connect(): void
    disconnect(): void
    sendCredentials(obj: object): void
    focus(): void
    blur(): void
    machineOutChar(code: number, keysym: number): void
    clipboardPaste(text: string): void
    requestDesktopSize(width: number, height: number): void
    sendCtrlAltDel(): void
    sendKey(code: number, keysym: number): void

    addEventListener(type: string, handler: (e: any) => void): void
    removeEventListener(type: string, handler: (e: any) => void): void

    get_display(): object
    get_keyboard(): object
    get_pointer(): object
  }

  export = RFB
}
