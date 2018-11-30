import {Node, NodeInfo} from '../app.datatypes';

/**
 * Node is "discovered" when every one of its discovery servers can see it
 */
function isDiscovered(nodeInfo: NodeInfo): boolean {
  const discoveries = Object.keys(nodeInfo.discoveries);

  if (discoveries.length === 0) {
    return false;
  }

  return discoveries.every((discovery) => {
    return nodeInfo.discoveries[discovery] === true;
  });
}

/**
 * (1) Return a name corresponding to the node's IP
 *
 * Manager (IP:192.168.0.2)
 * Node1 (IP:192.168.0.3)
 * Node2 (IP:192.168.0.4)
 * Node3 (IP:192.168.0.5)
 * Node4 (IP:192.168.0.6)
 * Node5 (IP:192.168.0.7)
 * Node6 (IP:192.168.0.8)
 * Node7 (IP:192.168.0.9)
 *
 * @param {Node} node
 * @returns {string}
 */

const MANAGER_CODE = 2;

function getNodeLabel(node: Node): string {
  let nodeLabel = null;
  try {
    const ipWithourPort = getNodeIp(node),
          nodeNumber = getNodeNumber(node);

    if (nodeNumber === MANAGER_CODE) {
      nodeLabel = 'Manager';
    } else if (nodeNumber > MANAGER_CODE && nodeNumber < 8) {
      nodeLabel = `Node ${nodeNumber - MANAGER_CODE}`;
    } else {
      nodeLabel = ipWithourPort;
    }
  } catch (e) {}

  return nodeLabel;
}

function getNodeIp(node: Node): string {
  return node.addr.split(':')[0];
}

function getNodeNumber(node: Node): number {
  return parseInt(getNodeIp(node).split('.')[3], 10);
}

function isManager(node: Node) {
  return getNodeNumber(node) === MANAGER_CODE;
}

export
{
  isDiscovered,
  getNodeLabel,
  isManager
};
