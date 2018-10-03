import * as Collections from 'typescript-collections';

export interface ImHistoryMessage {
  From?: string;
  Msg?: string;
  IsTime?: boolean;
  Timestamp?: number;
}

export interface RecentItem {
  name: string;
  icon: HeadColorMatch;
  last: string;
  unRead?: number;
}

export interface UserInfo {
  Icon?: HeadColorMatch;
}
export interface HeadColorMatch {
  bg?: string;
  color?: string;
}
