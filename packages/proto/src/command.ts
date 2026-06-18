// co-browsing 命令协议（visitor-sdk 端，与 admin/server 同源）
// 详见 docs/progress/2026-06-17-slice-1e-spec.md §命令协议

export type CommandType =
  | 'cursor_highlight'
  | 'click'
  | 'scroll'
  | 'fill_input'
  | 'navigate'
  | 'release_control'
  | 'show_popup'    // 1g
  | 'chat_message'; // 1g

export interface CommandCursor { x: number; y: number; name: string; }
export interface CommandClick { node_id: number; x: number; y: number; }
export interface CommandScroll { x: number; y: number; }
export interface CommandFillInput { node_id: number; value: string; }
export interface CommandNavigate { url: string; }

// 1g
export interface CommandPopup {
  title: string;
  body: string;
  action_label?: string;
  action_url?: string;
  dismissible: boolean;
}

export interface CommandChatMessage {
  message_id: number;
  content: string;
}

export interface CommandPayload {
  type: CommandType;
  ts: number;
  cursor?: CommandCursor;
  click?: CommandClick;
  scroll?: CommandScroll;
  fill_input?: CommandFillInput;
  navigate?: CommandNavigate;
  popup?: CommandPopup;
  chat?: CommandChatMessage;
}
