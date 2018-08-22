import {NodeInfo} from "../app.datatypes";

/**
 * Node is online if at least one discovery is seeing it.
 */
function isOnline(nodeInfo: NodeInfo): boolean
{
  return Object.keys(nodeInfo.discoveries).some((discovery) =>
  {
    return nodeInfo.discoveries[discovery] === true;
  });
}

export
{
  isOnline
}
