import { Injectable } from '@angular/core';

const KEY_REFRESH_SECONDS = 'KEY_REFRESH_SECONDS';
const KEY_DEFAULT_LANG = 'KEY_DEFAULT_LANG';

@Injectable({
  providedIn: 'root'
})
export class StorageService {
  private storage: Storage;

  constructor() {
    this.storage = localStorage;
  }

  private static nodeLabelNamespace(nodeKey: string): string {
    return `${nodeKey}-label`;
  }

  setNodeLabel(nodeKey: string, nodeLabel: string): void {
    this.storage.setItem(StorageService.nodeLabelNamespace(nodeKey), nodeLabel);
  }

  getNodeLabel(nodeKey: string): string {
    return this.storage.getItem(StorageService.nodeLabelNamespace(nodeKey));
  }

  setRefreshTime(seconds: number) {
    this.storage.setItem(KEY_REFRESH_SECONDS, seconds.toString());
  }

  getRefreshTime(): number {
    return parseInt(this.storage.getItem(KEY_REFRESH_SECONDS), 10) || 5;
  }

  setDefaultLanguage(lang: string): void {
    this.storage.setItem(KEY_DEFAULT_LANG, lang);
  }

  getDefaultLanguage(): string {
    return this.storage.getItem(KEY_DEFAULT_LANG) || 'en';
  }
}
