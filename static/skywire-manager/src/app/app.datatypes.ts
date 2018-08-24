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

export interface NodeStatusInfo extends Node, NodeInfo {
  online: boolean;
}

export interface NodeData {
  node: Node;
  info: NodeInfo;
  apps: NodeApp[];
  allNodes: NodeStatusInfo[];
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

export interface LogMessage {
  time: number;
  msg: string;
}

export interface AutoStartConfig {
  sshs: boolean;
  sshc: boolean;
  sshc_conf_nodeKey: string;
  sshc_conf_appKey: string;
  sshc_conf_discovery: string;
  sockss: boolean;
  socksc: boolean;
  socksc_conf_nodeKey: string;
  socksc_conf_appKey: string;
  socksc_conf_discovery: string;
}

export interface Keypair {
  nodeKey: string;
  appKey: string;
}

export interface SearchResult {
  result: SearchResultItem[];
  seq: number;
  count: number;
}

export interface SearchResultItem {
  node_key: string;
  app_key: string;
  location: string;
  version: string;
  node_version: string[];
}

export interface DiscoveryAddress {
  domain: string;
  publicKey: string;
}
