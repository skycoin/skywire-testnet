import { Injectable } from '@angular/core';

@Injectable()
export class UserService {
  SSHCLIENTINFO = '_SKYWIRE_SSHCLIENTINFO';
  SOCKETCLIENTINFO = '_SKYWIRE_SOCKETCLIENTINFO';
  HOMENODELABLE = '_SKYWIRE_HOMENODELABEL';
  constructor() { }
  saveHomeLabel(nodeKey: string, label: string) {
    let homeLabels = this.get(this.HOMENODELABLE);
    if (!homeLabels) {
      homeLabels = {};
    }
    homeLabels[nodeKey] = label;
    localStorage.setItem(this.HOMENODELABLE, JSON.stringify(homeLabels));
  }
  get(key: string) {
    return JSON.parse(localStorage.getItem(key));
  }
}
// export interface LinkedList

