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

export interface NodeInfo {
  app_feedbacks: NodeFeedback[]|null;
  transports: NodeTransport[]|null;
  discoveries: NodeDiscovery|null;
  os: string;
  tag: string;
  version: string;
}

export interface NodeDiscovery {
  [key: string]: boolean;
}

export interface NodeTransport {
  from_node: string;
  to_node: string;
  from_app: string;
  to_app: string;
  upload_bandwidth: number;
  download_bandwidth: number;
  upload_total: number;
  download_total: number;
}

export interface NodeFeedback {
  key: string;
  port: number;
  failed: boolean;
  unread: number;
}

export interface ClientConnection {
  label: string;
  nodeKey: string;
  appKey: string;
  count: number;
}
