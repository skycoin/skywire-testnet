export interface Node {
  key?: string;
  addr?: string;
  type: string;
  send_bytes: number;
  recv_bytes: number;
  last_ack_time: number;
  start_time: number;
}

export interface NodeApp {
  key: string;
  attributes: string[]|null;
  allow_nodes: any;
}
