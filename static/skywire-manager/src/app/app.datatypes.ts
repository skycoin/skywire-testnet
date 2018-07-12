export interface Node {
  key: string;
  type: string;
  send_bytes: number;
  recv_bytes: number;
  last_ack_time: number;
  start_time: number;
}
